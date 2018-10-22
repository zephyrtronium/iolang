package iolang

import (
	"bytes"
	"fmt"
	"sort"
	"text/tabwriter"
)

// OpTable is an Object which manages message-shuffling operators in Io. Each
// VM's OperatorTable is a singleton.
type OpTable struct {
	Object
	Operators map[string]Operator
}

// Clone generates a shallow copy of the OpTable.
func (o *OpTable) Clone() Interface {
	return &OpTable{
		Object:    Object{Slots: Slots{}, Protos: []Interface{o}},
		Operators: o.Operators,
	}
}

// String generates a string representation of the operators in the table.
func (o *OpTable) String() string {
	ops := o.Operators
	s := make(opSorter, 0, len(ops))
	a := make(opSorter, 0)
	for k, v := range ops {
		if v.Calls == "" {
			s = append(s, opToSort{k, v})
		} else {
			a = append(a, opToSort{k, v})
		}
	}
	sort.Sort(s)
	sort.Sort(a)
	var b bytes.Buffer
	b.WriteString("Operators\n")
	w := tabwriter.NewWriter(&b, 4, 0, 1, ' ', 0)
	if len(s) > 0 {
		prev := s[0]
		fmt.Fprintf(w, "\t%d\t%s", prev.op.Prec, prev.name)
		for _, v := range s[1:] {
			if prev.op != v.op {
				fmt.Fprintf(w, "\n\t%d", v.op.Prec)
			}
			fmt.Fprintf(w, "\t%s", v.name)
			prev = v
		}
	}
	w.Flush()
	b.WriteString("\n\nAssign Operators\n")
	w.Init(&b, 3, 0, 1, ' ', 0)
	if len(a) > 0 {
		for _, v := range a {
			fmt.Fprintf(w, "\t%s\t%s\n", v.name, v.op.Calls)
		}
	}
	w.Flush()
	return b.String()
}

// Operator defines an Io operator.
type Operator struct {
	// For assign operators, the slot the operator calls. This must be the
	// empty string for operators that are not assign operators.
	Calls string
	// Precedence. Lower is more binding.
	Prec int
}

// leastBindingOp is the least binding operator, used internally to manage the
// operator shuffling stack.
var leastBindingOp = Operator{Prec: int((^uint(0)) >> 1)}

// MoreBinding determines whether this Operator is at least as binding as
// another.
func (op Operator) MoreBinding(than Operator) bool {
	return op.Prec <= than.Prec
}

func (vm *VM) initOpTable() {
	slots := Slots{
		"addAssignOperator":    vm.NewTypedCFunction(OperatorTableAddAssignOperator),
		"addOperator":          vm.NewTypedCFunction(OperatorTableAddOperator),
		"asString":             vm.NewTypedCFunction(OperatorTableAsString),
		"precedenceLevelCount": vm.NewNumber(32), // not really
		"type":                 vm.NewString("OperatorTable"),
	}
	// The VM already created an OpTable so that initObject() can refer to it
	// to create the slot on BaseObject.
	vm.Operators.Object = Object{Slots: slots, Protos: []Interface{vm.BaseObject}}
	vm.Operators.Operators = map[string]Operator{
		"?":      Operator{"", 0},
		"@":      Operator{"", 0},
		"@@":     Operator{"", 0},
		"**":     Operator{"", 1},
		"%":      Operator{"", 2},
		"*":      Operator{"", 2},
		"/":      Operator{"", 2},
		"+":      Operator{"", 3},
		"-":      Operator{"", 3},
		"<<":     Operator{"", 4},
		">>":     Operator{"", 4},
		"<":      Operator{"", 5},
		"<=":     Operator{"", 5},
		">":      Operator{"", 5},
		">=":     Operator{"", 5},
		"!=":     Operator{"", 6},
		"==":     Operator{"", 6},
		"&":      Operator{"", 7},
		"^":      Operator{"", 8},
		"|":      Operator{"", 9},
		"&&":     Operator{"", 10},
		"and":    Operator{"", 10},
		"or":     Operator{"", 11},
		"||":     Operator{"", 11},
		"..":     Operator{"", 12},
		"%=":     Operator{"", 13},
		"&=":     Operator{"", 13},
		"*=":     Operator{"", 13},
		"+=":     Operator{"", 13},
		"-=":     Operator{"", 13},
		"/=":     Operator{"", 13},
		"<<=":    Operator{"", 13},
		">>=":    Operator{"", 13},
		"^=":     Operator{"", 13},
		"|=":     Operator{"", 13},
		"return": Operator{"", 14},

		// Assign operators.
		"::=": Operator{"newSlot", -1},
		":=":  Operator{"setSlot", -1},
		"=":   Operator{"updateSlot", -1},
	}
	// Even though there isn't actually any way to create a new OperatorTable,
	// still create a proto in Core.
	SetSlot(vm.Core, "OperatorTable", vm.Operators)
}

// shufLevel is a linked stack item to manage the messages to which to attach.
type shufLevel struct {
	op  Operator
	m   *Message
	up  *shufLevel
	typ int
}

func (ll *shufLevel) String() string {
	k := 0
	for nl := ll; nl != nil; nl = nl.up {
		k++
	}
	var typ string
	switch ll.typ {
	case levArg:
		typ = "levArg"
	case levAttach:
		typ = "levAttach"
	case levNew:
		typ = "levNew"
	}
	if ll.op == leastBindingOp {
		return fmt.Sprintf("shufLevel{leastBindingOp m=%s depth=%d typ=%s}", ll.m.Name(), k, typ)
	}
	if ll.op.Calls == "" {
		return fmt.Sprintf("shufLevel{asgn=%s m=%s depth=%d typ=%s}", ll.op.Calls, ll.m.Name(), k, typ)
	}
	return fmt.Sprintf("shufLevel{prec=%d m=%s depth=%d typ=%s}", ll.op.Prec, ll.m.Name(), k, typ)
}

// Level types, indicating the meaning of attaching a message to this level.
const (
	levAttach = iota // Attach to this message.
	levArg           // Add an argument.
	levNew           // New level without a message.
)

// pop unlinks the stack until a level at least as binding as op is found,
// returning the new top of the stack.
func (ll *shufLevel) pop(op Operator) *shufLevel {
	for ll != nil && ll.up != nil && ll.op.MoreBinding(op) && ll.typ != levArg {
		ll.finish()
		ll, ll.up, ll.m = ll.up, nil, nil
	}
	return ll
}

// clear fully clears the stack to prepare for the next top-level message.
func (ll *shufLevel) clear() *shufLevel {
	for ll != nil && ll.up != nil {
		ll.finish()
		ll, ll.up, ll.m = ll.up, nil, nil
	}
	// ll is now the top of the stack, which we need to reset.
	ll.finish()
	ll.m = nil
	ll.typ = levNew
	return ll
}

// attach attaches a message to this level in the correct way for its type.
func (ll *shufLevel) attach(m *Message) {
	if ll.m == nil {
		ll.m = m
		return
	}
	switch ll.typ {
	case levAttach:
		// Normally we would do ll.m.InsertAfter(m), but here, setting m.Next
		// to the current ll.m.Next will cause an infinite loop.
		ll.m.Next = m
		if m != nil {
			m.Prev = ll.m
		}
	case levArg:
		ll.m.Args = append(ll.m.Args, m)
	case levNew:
		ll.m = m
	}
}

// attachReplace attaches a message to this level, then makes the message the
// new level target.
func (ll *shufLevel) attachReplace(m *Message) {
	ll.attach(m)
	ll.m, ll.typ = m, levAttach
}

// push attaches a new level to the top of the stack, returning the new top.
func (ll *shufLevel) push(m *Message, op Operator) *shufLevel {
	ll.attachReplace(m)
	return &shufLevel{
		op:  op,
		m:   m,
		up:  ll,
		typ: levArg,
	}
}

// finish declares a stack level to be done processing.
func (ll *shufLevel) finish() {
	if m := ll.m; m != nil {
		m.InsertAfter(nil)
		if len(m.Args) == 1 {
			a := m.Args[0]
			if a.Name() == "" && len(a.Args) == 1 && a.Next == nil {
				// We added a () for grouping, but we don't need it anymore.
				m.Args[0] = a.Args[0]
				a.Args = nil
			}
		}
	}
}

// doLevel shuffles one level. The new stack top, extra messages to be
// shuffled, and any syntax error are returned.
func (ll *shufLevel) doLevel(vm *VM, ops *OpTable, m *Message) (nl *shufLevel, next []*Message, err *Exception) {
	if op, ok := ops.Operators[m.Name()]; ok {
		if op.Calls != "" {
			// Assignment operator.
			lhs := ll.m
			if lhs == nil {
				// Assigning to nothing is illegal.
				err = vm.NewExceptionf("%s assigning to nothing", m.Name())
				return ll, nil, err
			}
			if len(m.Args) > 1 {
				// Assignment operators are allowed to have only zero or
				// one argument.
				err = vm.NewExceptionf("too many arguments to %s", m.Name())
				return ll, nil, err
			}
			if m.Next.IsTerminator() && len(m.Args) == 0 {
				// Assigning nothing to something is illegal.
				err = vm.NewExceptionf("%s requires a value to assign", m.Name())
				return ll, nil, err
			}
			if len(lhs.Args) > 0 {
				// Assigning to a call used to be illegal, but a recent change
				// allows expressions like `a(b, c) := d`, tranforming into
				// `setSlot(a(b, c), d)`. This was to enable a Python-style
				// multiple assignment syntax like
				// `target [a, b, c] <- list(x, y, z)` to accomplish
				// `target do(a = x; b = y; c = z)`.
				//
				// I'm not implementing this for a few reasons. First, the
				// meaning of lhs in this form is different in a non-obvious
				// way, as it is normally converted by name to a string; using
				// an existing assignment operator and accidentally or
				// unknowingly triggering this syntax will produce unexpected
				// results. Second, the implmentation of this technique
				// involves creating a deep copy of the entire message chain
				// forward, meaning if a file begins with this type of
				// assignment, the runtime will allocate (actually three)
				// copies of *every message in the file*, recursively, causing
				// essentially unbounded memory and stack usage. Third, the
				// current syntax assumes only a single message follows the
				// assignment operator, so `data(i,j) = Number constants pi`
				// will transform to
				// `assignOp(data(i,j), Number) constants pi`.
				//
				// I will note, however, that I would prefer that setSlot and
				// friends' first argument be the message rather than the
				// name thereof, so that a syntax like this wanted to be could
				// be implemented sanely and safely. This would be the time and
				// place to make that change, but I don't think it's a good
				// idea to diverge so far from Io early in development.
				err = vm.NewExceptionf("message preceding %s must have no args", m.Name())
				return ll, nil, err
			}

			// Handle `a := (b c) d ; e` as follows:
			//  1. Move op arg to a separate message: a :=() (b c) d ; e
			//  2. Give lhs arguments: a("a", (...)) :=() (b c) d ; e
			//  3. Change lhs name: setSlot("a", (...)) :=() (b c) d ; e
			//  4. Move msgs up to terminator: setSlot("a", (b c) d) := ; e
			//  5. Remove operator message: setSlot("a", (b c) d) ; e

			// 1. Move the operator argument, if it exists, to a separate
			// message.
			if len(m.Args) > 0 {
				m.InsertAfter(vm.IdentMessage("", m.Args...))
				m.Args = nil
			}
			// 2. Give lhs its arguments. The first is the name of the
			// slot to which we're assigning (assuming a built-in
			// assignment operator), and the second is the value to give
			// it. We'll also need to shuffle that value later.
			lhs.Args = []*Message{vm.StringMessage(lhs.Name()), m.Next}
			next = append(next, m.Next)
			// 3. Change lhs's name to the assign operator's call.
			lhs.Text = op.Calls
			// 4. Move messages up to but not including the next terminator
			// into the assignment's second argument. Really, we already
			// moved it there; we're finding the message to be the next
			// after lhs.
			last := m.Next
			for !last.Next.IsTerminator() {
				last = last.Next
			}
			if last.Next != nil {
				last.Next.Prev = lhs
			}
			lhs.Next = last.Next
			last.Next = nil

			// 5. Remove the operator message.
			m.Next = lhs.Next

			// It's legal to do something like `1 := x`, so we need to make
			// sure that x will be evaluated when that happens.
			lhs.Memo = nil
		} else {
			// Non-assignment operator.
			if len(m.Args) > 0 {
				// `a + (b - c) * d` is initially parsed as `b - c` being
				// the argument to +. In order to have order of operations
				// make sense, we need to move that argument to a separate
				// message, so we have `a +() (b - c) * d`, which we can
				// then shuffle into `a +((b - c) *(d))`.
				m.InsertAfter(vm.IdentMessage("", m.Args...))
				m.Args = nil
			}
			ll = ll.pop(op).push(m, op)
		}
	} else if m.IsTerminator() {
		ll = ll.pop(leastBindingOp)
		ll.attachReplace(m)
	} else {
		// Non-operator identifier or literal.
		ll.attachReplace(m)
	}
	return ll, next, nil
}

// OpShuffle performs operator-precedence reordering of a message. If the
// message (or one of its protos) has an OperatorTable slot that contains an
// *OpTable, it is used for operators; otherwise, the VM's default OpTable is
// used.
func (vm *VM) OpShuffle(m *Message) (err *Exception) {
	if m == nil {
		return nil
	}
	if m.Name() == "__noShuffling__" {
		// We could make __noShuffling__ just an Object with an OperatorTable
		// that is empty, but doing it this way allows us to skip shuffling
		// entirely, speeding up parsing.
		// Io's treatment of __noShuffling__ is interesting: a message named
		// __noShuffling__ prevents operator shuffling as expected, but there
		// is no object so named, so you have to create it yourself, but you
		// have to use setSlot directly, because you  can't assign to
		// __noShuffling__ using an operator, because assign operator
		// transformation is handled during operator shuffling, which doesn't
		// happen because the message begins with __noShuffling__. :)
		return nil
	}
	opsx, _ := GetSlot(m, "OperatorTable")
	var ops *OpTable
	if ops, _ = opsx.(*OpTable); ops == nil {
		ops = vm.Operators
	}
	ll := &shufLevel{
		op:  leastBindingOp,
		typ: levNew,
	}
	exprs := []*Message{m}
	var next []*Message
	for len(exprs) > 0 {
		expr := exprs[len(exprs)-1]
		exprs = exprs[:len(exprs)-1]
		for {
			ll, next, err = ll.doLevel(vm, ops, expr)
			if err != nil {
				return err
			}
			exprs = append(exprs, next...)
			exprs = append(exprs, expr.Args...)
			if expr.Next == nil {
				break
			}
			expr = expr.Next
		}
		ll = ll.clear()
	}
	return nil
}

// OperatorTableAddAssignOperator is an OperatorTable method.
//
// addAssignOperator adds an assign operator:
//
//	 OperatorTable addAssignOperator(name, calls)
//
// For example, to create a <- operator that calls the send method:
//
//   io> OperatorTable addAssignOperator("<-", "send")
//   io> message(thing a <- b)
//   thing send("a", b)
func OperatorTableAddAssignOperator(vm *VM, target, locals Interface, msg *Message) Interface {
	name, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	calls, err := msg.StringArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
	}
	op := Operator{
		Calls: calls.String(),
		Prec:  -1,
	}
	target.(*OpTable).Operators[name.String()] = op
	return target
}

// OperatorTableAddOperator is an OperatorTable method.
//
// addOperator adds a binary operator:
//
//   OperatorTable addOperator(name, precedence)
//
// For example, to create a :* operator with the same precedence as the *
// operator:
//
//   OperatorTable addOperator(":*", 2)
func OperatorTableAddOperator(vm *VM, target, locals Interface, msg *Message) Interface {
	name, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	prec, err := msg.NumberArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
	}
	op := Operator{
		Calls: "",
		Prec:  int(prec.Value),
	}
	target.(*OpTable).Operators[name.String()] = op
	return target
}

// OperatorTableAsString is an OperatorTable method.
//
// asString creates a string representation of an object.
func OperatorTableAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(target.(*OpTable).String())
}

// opToSort is a type for sorting operators by precedence.
type opToSort struct {
	name string
	op   Operator
}

// opSorter is a type for sorting operators by precedence.
type opSorter []opToSort

func (o opSorter) Len() int {
	return len(o)
}

func (o opSorter) Less(i, j int) bool {
	if o[i].op.MoreBinding(o[j].op) {
		// Strictly less.
		return true
	}
	if o[j].op.MoreBinding(o[i].op) {
		// Strictly greater.
		return false
	}
	// Equal precedence, so sort them by name.
	return o[i].name < o[j].name
}

func (o opSorter) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
