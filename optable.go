package iolang

func (vm *VM) initOpTable() {
	ops := map[string]Interface{
		"?":      vm.NewNumber(0),
		"@":      vm.NewNumber(0),
		"@@":     vm.NewNumber(0),
		"**":     vm.NewNumber(1),
		"%":      vm.NewNumber(2),
		"*":      vm.NewNumber(2),
		"/":      vm.NewNumber(2),
		"+":      vm.NewNumber(3),
		"-":      vm.NewNumber(3),
		"<<":     vm.NewNumber(4),
		">>":     vm.NewNumber(4),
		"<":      vm.NewNumber(5),
		"<=":     vm.NewNumber(5),
		">":      vm.NewNumber(5),
		">=":     vm.NewNumber(5),
		"!=":     vm.NewNumber(6),
		"==":     vm.NewNumber(6),
		"&":      vm.NewNumber(7),
		"^":      vm.NewNumber(8),
		"|":      vm.NewNumber(9),
		"&&":     vm.NewNumber(10),
		"and":    vm.NewNumber(10),
		"or":     vm.NewNumber(11),
		"||":     vm.NewNumber(11),
		"..":     vm.NewNumber(12),
		"%=":     vm.NewNumber(13),
		"&=":     vm.NewNumber(13),
		"*=":     vm.NewNumber(13),
		"+=":     vm.NewNumber(13),
		"-=":     vm.NewNumber(13),
		"/=":     vm.NewNumber(13),
		"<<=":    vm.NewNumber(13),
		">>=":    vm.NewNumber(13),
		"^=":     vm.NewNumber(13),
		"|=":     vm.NewNumber(13),
		"return": vm.NewNumber(14),
	}
	asgn := map[string]Interface{
		"::=": vm.NewString("newSlot"),
		":=":  vm.NewString("setSlot"),
		"=":   vm.NewString("updateSlot"),
	}
	slots := Slots{
		"assignOperators":      vm.NewMap(asgn),
		"operators":            vm.NewMap(ops),
		"precedenceLevelCount": vm.NewNumber(leastBindingOp), // not really
		"type":                 vm.NewString("OperatorTable"),
	}
	vm.Operators = vm.ObjectWith(slots)
	vm.Core.SetSlot("OperatorTable", vm.Operators)
	// This method can be called post-initialization if both Core's and
	// Core Message's OperatorTable slots are removed. In that case, we want to
	// set the slot on both of those.
	msg, ok := vm.Core.GetLocalSlot("Message")
	if ok {
		msg.SetSlot("OperatorTable", vm.Operators)
	}
}

// leastBindingOp is the precedence of the least binding operator, used
// internally to manage the operator shuffling stack.
const leastBindingOp = 1.797693134862315708145274237317043567981e+308

// shufLevel is a linked stack item to manage the messages to which to attach.
type shufLevel struct {
	op  float64
	m   *Message
	up  *shufLevel
	typ int
}

// Level types, indicating the meaning of attaching a message to this level.
const (
	levAttach = iota // Attach to this message.
	levArg           // Add an argument.
	levNew           // New level without a message.
)

// pop unlinks the stack until a level at least as binding as op is found,
// returning the new top of the stack.
func (ll *shufLevel) pop(op float64) *shufLevel {
	for ll != nil && ll.up != nil && ll.op <= op && ll.typ != levArg {
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
func (ll *shufLevel) push(m *Message, op float64) *shufLevel {
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
func (ll *shufLevel) doLevel(vm *VM, ops, asgns map[string]Interface, m *Message) (nl *shufLevel, next []*Message, err *Exception) {
	if op, ok := asgns[m.Name()]; ok {
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
			// I'm not implementing this for a few reasons. First, the meaning
			// of lhs in this form is different in a non-obvious way, as it is
			// normally converted by name to a string; using an existing
			// assignment operator and accidentally or unknowingly triggering
			// this syntax will produce unexpected results. Second, the
			// implementation of this technique involves creating a deep copy
			// of the entire message chain forward, meaning if a file begins
			// with this type of assignment, the runtime will allocate
			// (actually three) copies of *every message in the file*,
			// recursively, causing essentially unbounded memory and stack
			// usage. Third, the current syntax assumes only a single message
			// follows the assignment operator, so
			// `data(i,j) = Number constants pi` will transform to
			// `assignOp(data(i,j), Number) constants pi`.
			//
			// I will note, however, that I would prefer that setSlot and
			// friends' first argument be the message rather than the name
			// thereof, so that a syntax like this wanted to be could be
			// implemented sanely and safely. This would be the time and place
			// to make that change, but I don't think it's a good idea to
			// diverge so far from Io early in development.
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
		calls, ok := op.(*Sequence)
		if !ok {
			err = vm.NewExceptionf("OperatorTable assignOperators at(%q) must be Sequence, not %s", m.Name(), vm.TypeName(op))
			return ll, nil, err
		}
		lhs.Text = calls.String()
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
	} else if op, ok = ops[m.Name()]; ok {
		// Non-assignment operator.
		prec, ok := op.(*Number)
		if !ok {
			err = vm.NewExceptionf("OperatorTable operators at(%q) must be Number, not %s", m.Name(), vm.TypeName(op))
			return ll, nil, err
		}
		if len(m.Args) > 0 {
			// `a + (b - c) * d` is initially parsed as `b - c` being
			// the argument to +. In order to have order of operations
			// make sense, we need to move that argument to a separate
			// message, so we have `a +() (b - c) * d`, which we can
			// then shuffle into `a +((b - c) *(d))`.
			m.InsertAfter(vm.IdentMessage("", m.Args...))
			m.Args = nil
		}
		ll = ll.pop(prec.Value).push(m, prec.Value)
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
	operators, proto := vm.GetSlot(m, "OperatorTable")
	if proto == nil {
		operators = vm.Operators
	}
	var ops, asgn *Map
	for {
		opsx, _ := vm.GetSlot(operators, "operators")
		asgnx, _ := vm.GetSlot(operators, "assignOperators")
		ops, _ = opsx.(*Map)
		asgn, _ = asgnx.(*Map)
		if ops == nil || asgn == nil {
			vm.initOpTable()
			operators, _ = vm.Core.GetLocalSlot("OperatorTable")
		} else {
			break
		}
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
			ll, next, err = ll.doLevel(vm, ops.Value, asgn.Value, expr)
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
