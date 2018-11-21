package iolang

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// A Message is the fundamental syntactic element and functionality of Io.
type Message struct {
	Object
	// Text of this message.
	Text string
	// This message's arguments.
	Args []*Message
	// Next and previous messages.
	Next, Prev *Message

	// Cached value of this message. If this is non-nil, it is used instead
	// of activating the message.
	Memo Interface

	// Label of the message, generally the name of the file from which it was
	// parsed, if any.
	Label string
	// One-based line and column numbers within the file at which the message
	// was parsed.
	Line, Col int
}

// Activate returns the message.
func (m *Message) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	return m
}

// Clone returns a clone of the message with the same text only.
func (m *Message) Clone() Interface {
	return &Message{
		Object: Object{Slots: Slots{}, Protos: []Interface{m}},
		Text:   m.Text,
	}
}

// IdentMessage creates a message of a given identifier. Additional messages
// may be passed as arguments.
func (vm *VM) IdentMessage(s string, args ...*Message) *Message {
	return &Message{
		Object: *vm.CoreInstance("Message"),
		Text:   s,
		Args:   args,
	}
}

// StringMessage creates a message carrying a string value.
func (vm *VM) StringMessage(s string) *Message {
	return &Message{
		Object: *vm.CoreInstance("Message"),
		Text:   strconv.Quote(s),
		Memo:   vm.NewString(s),
	}
}

// NumberMessage creates a message carrying a numeric value.
func (vm *VM) NumberMessage(v float64) *Message {
	return &Message{
		Object: *vm.CoreInstance("Message"),
		Text:   strconv.FormatFloat(v, 'g', -1, 64),
		Memo:   vm.NewNumber(v),
	}
}

// CachedMessage creates a message carrying a cached value.
func (vm *VM) CachedMessage(v Interface) *Message {
	return &Message{
		Object: *vm.CoreInstance("Message"),
		Text:   vm.AsString(v),
		Memo:   v,
	}
}

// DeepCopy creates a copy of the message linked to copies of each message
// forward.
func (m *Message) DeepCopy() *Message {
	if m == nil {
		return nil
	}
	// We can't use vm.CoreInstance because we won't have access to a VM
	// everywhere we need it, e.g. Block.Clone(). Instead, steal the protos
	// from the message we're copying.
	fm := &Message{
		Object: Object{Slots: Slots{}, Protos: append([]Interface{}, m.Protos...)},
		Text:   m.Text,
		Args:   make([]*Message, len(m.Args)),
		Prev:   m.Prev,
		Memo:   m.Memo,
	}
	for i, arg := range m.Args {
		fm.Args[i] = arg.DeepCopy()
	}
	for pm, nm := fm, m.Next; nm != nil; pm, nm = pm.Next, nm.Next {
		pm.Next = &Message{
			Object: Object{Slots: Slots{}, Protos: append([]Interface{}, nm.Protos...)},
			Text:   nm.Text,
			Args:   make([]*Message, len(nm.Args)),
			Prev:   pm,
			Memo:   nm.Memo,
		}
		for i, arg := range nm.Args {
			pm.Next.Args[i] = arg.DeepCopy()
		}
	}
	return fm
}

// ArgCount returns the number of arguments to the message.
func (m *Message) ArgCount() int {
	return len(m.Args)
}

// AssertArgCount returns an error if the message does not have the given
// number of arguments. name is the name of the message used in the generated
// error message.
func (m *Message) AssertArgCount(name string, n int) error {
	if len(m.Args) != n {
		return fmt.Errorf("%s must have %d arguments", name, n)
	}
	return nil
}

// ArgAt returns the argument at position n, or nil if the position is out of
// bounds.
func (m *Message) ArgAt(n int) *Message {
	if n >= len(m.Args) || n < 0 {
		return nil
	}
	return m.Args[n]
}

// NumberArgAt evaluates the nth argument and returns it as a Number. If a
// return expression or an exception occurs during evaluation, the result will
// be nil, and the control flow object will be returned. If the evaluated
// result is not a Number, the result will be nil and an exception will be
// returned.
func (m *Message) NumberArgAt(vm *VM, locals Interface, n int) (*Number, Interface) {
	v, ok := CheckStop(m.EvalArgAt(vm, locals, n), LoopStops)
	if !ok {
		return nil, v
	}
	if num, ok := v.(*Number); ok {
		return num, nil
	}
	// Not the expected type, so return an error.
	return nil, vm.RaiseExceptionf("argument %d to %s must be Number, not %s", n, m.Text, vm.TypeName(v))
}

// StringArgAt evaluates the nth argument and returns it as a Sequence. If a
// return expression or an exception occurs during evaluation, the result will
// be nil, and the control flow object will be returned. If the evaluated
// result is not a Sequence, the result will be nil and an exception will be
// returned.
func (m *Message) StringArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface) {
	v, ok := CheckStop(m.EvalArgAt(vm, locals, n), LoopStops)
	if !ok {
		return nil, v
	}
	if str, ok := v.(*Sequence); ok {
		return str, nil
	}
	// Not the expected type, so return an error.
	return nil, vm.RaiseExceptionf("argument %d to %s must be Sequence, not %s", n, m.Text, vm.TypeName(v))
}

// ListArgAt evaluates the nth argument and returns it as a List. If a return
// expression or an exception occurs during evaluation, the result will be nil,
// and the control flow object will be returned. If the evaluated result is not
// a List, the result will be nil, and an exception will be returned.
func (m *Message) ListArgAt(vm *VM, locals Interface, n int) (*List, Interface) {
	v, ok := CheckStop(m.EvalArgAt(vm, locals, n), LoopStops)
	if !ok {
		return nil, v
	}
	if lst, ok := v.(*List); ok {
		return lst, nil
	}
	// Not the expected type, so return an error.
	return nil, vm.RaiseExceptionf("argument %d to %s must be List, not %s", n, m.Text, vm.TypeName(v))
}

// AsStringArgAt evaluates the nth argument, then activates its asString slot
// for a string representation. If the result is not a string, then the result
// is nil, and an error is returned.
func (m *Message) AsStringArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface) {
	v := m.EvalArgAt(vm, locals, n)
	if asString, proto := GetSlot(v, "asString"); proto != nil {
		r, ok := CheckStop(asString.Activate(vm, locals, locals, vm.IdentMessage("asString")), LoopStops)
		if !ok {
			return nil, r
		}
		if s, ok := r.(*Sequence); ok {
			return s, nil
		}
	}
	return nil, vm.NewExceptionf("argument %d to %s cannot be converted to string", n, m.Text)
}

// EvalArgAt evaluates the nth argument.
func (m *Message) EvalArgAt(vm *VM, locals Interface, n int) Interface {
	return m.ArgAt(n).Eval(vm, locals)
}

// Eval evaluates a message in the context of the given VM. This is a proxy to
// Send using locals as the target.
func (m *Message) Eval(vm *VM, locals Interface) (result Interface) {
	return m.Send(vm, locals, locals)
}

// Send evaluates a message in the context of the given VM, targeting an
// object.
func (m *Message) Send(vm *VM, target, locals Interface) (result Interface) {
	firstTarget := target
	for m != nil {
		if m.Memo != nil {
			// It is the parser's responsibility to set memos for literals.
			result = m.Memo
			target = result
		} else {
			if !m.IsTerminator() {
				if newtarget, proto := GetSlot(target, m.Text); proto != nil {
					// We have the slot.
					var ok bool
					result, ok = CheckStop(newtarget.Activate(vm, target, locals, m), NoStop)
					if !ok {
						return result
					}
				} else if forward, fp := GetSlot(target, "forward"); fp != nil {
					result = forward.Activate(vm, target, locals, m)
				} else {
					return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), m.Text)
				}
				if result == nil {
					// No message should evaluate to something that is not an
					// Io object, so we want to convert nil to vm.Nil.
					result = vm.Nil
				}
				target = result
			} else {
				target = firstTarget
			}
		}
		m = m.Next
	}
	if result == nil {
		result = vm.Nil
	}
	return result
}

// InsertAfter links another message to follow this one.
func (m *Message) InsertAfter(other *Message) {
	if m.Next != nil {
		m.Next.Prev = other
	}
	if other != nil {
		other.Next = m.Next
		other.Prev = m
	}
	m.Next = other
}

// IsStart determines whether this message is the start of a "statement." This
// is true if it has no previous link or if the previous link is a SemiSym.
func (m *Message) IsStart() bool {
	return m.Prev.IsTerminator()
}

// IsTerminator determines whether this message is the end of an expression.
// This is true if it is nil or it is a semicolon or newline.
func (m *Message) IsTerminator() bool {
	return m == nil || m.Text == ";" || m.Text == "\n"
}

// Name generates a string representation of this message based upon its type.
func (m *Message) Name() string {
	if m == nil {
		return "<nil message>"
	}
	return m.Text
}

// String generates a diagnostic string representation of this message.
func (m *Message) String() string {
	return "message-" + m.Name()
}

func (m *Message) stringRecurse(vm *VM, b *bytes.Buffer) {
	if m == nil {
		b.WriteString("<nil>")
		return
	}
	for m != nil {
		if m.Memo != nil {
			if msg, ok := m.Memo.(*Message); ok {
				b.WriteString("<message(")
				msg.stringRecurse(vm, b)
				b.WriteString(")>")
			} else {
				b.WriteString(vm.AsString(m.Memo))
			}
		} else {
			b.WriteString(m.Text)
			if len(m.Args) > 0 {
				b.WriteByte('(')
				m.Args[0].stringRecurse(vm, b)
				for _, arg := range m.Args[1:] {
					b.WriteString(", ")
					arg.stringRecurse(vm, b)
				}
				b.WriteByte(')')
			}
		}
		if m.Next != nil {
			b.WriteByte(' ')
		}
		m = m.Next
	}
}

func (vm *VM) initMessage() {
	var exemplar *Message
	// TODO: label, lineNumber, setLabel, setLineNumber
	slots := Slots{
		"appendArg":                  vm.NewTypedCFunction(MessageAppendArg, exemplar),
		"appendCachedArg":            vm.NewTypedCFunction(MessageAppendCachedArg, exemplar),
		"argAt":                      vm.NewTypedCFunction(MessageArgAt, exemplar),
		"argCount":                   vm.NewTypedCFunction(MessageArgCount, exemplar),
		"argsEvaluatedIn":            vm.NewTypedCFunction(MessageArgsEvaluatedIn, exemplar),
		"arguments":                  vm.NewTypedCFunction(MessageArguments, exemplar),
		"asMessageWithEvaluatedArgs": vm.NewTypedCFunction(MessageAsMessageWithEvaluatedArgs, exemplar),
		"asString":                   vm.NewTypedCFunction(MessageAsString, exemplar),
		"cachedResult":               vm.NewTypedCFunction(MessageCachedResult, exemplar),
		"characterNumber":            vm.NewTypedCFunction(MessageCharacterNumber, exemplar),
		"clone":                      vm.NewTypedCFunction(MessageClone, exemplar),
		"doInContext":                vm.NewTypedCFunction(MessageDoInContext, exemplar),
		"fromString":                 vm.NewCFunction(MessageFromString),
		"hasCachedResult":            vm.NewTypedCFunction(MessageHasCachedResult, exemplar),
		"isEndOfLine":                vm.NewTypedCFunction(MessageIsEndOfLine, exemplar),
		"label":                      vm.NewTypedCFunction(MessageLabel, exemplar),
		"last":                       vm.NewTypedCFunction(MessageLast, exemplar),
		"lastBeforeEndOfLine":        vm.NewTypedCFunction(MessageLastBeforeEndOfLine, exemplar),
		"lineNumber":                 vm.NewTypedCFunction(MessageLineNumber, exemplar),
		"name":                       vm.NewTypedCFunction(MessageName, exemplar),
		"next":                       vm.NewTypedCFunction(MessageNext, exemplar),
		"nextIgnoreEndOfLines":       vm.NewTypedCFunction(MessageNextIgnoreEndOfLines, exemplar),
		"opShuffle":                  vm.NewTypedCFunction(MessageOpShuffle, exemplar),
		"previous":                   vm.NewTypedCFunction(MessagePrevious, exemplar),
		"removeCachedResult":         vm.NewTypedCFunction(MessageRemoveCachedResult, exemplar),
		"setArguments":               vm.NewTypedCFunction(MessageSetArguments, exemplar),
		"setCachedResult":            vm.NewTypedCFunction(MessageSetCachedResult, exemplar),
		"setCharacterNumber":         vm.NewTypedCFunction(MessageSetCharacterNumber, exemplar),
		"setLabel":                   vm.NewTypedCFunction(MessageSetLabel, exemplar),
		"setLineNumber":              vm.NewTypedCFunction(MessageSetLineNumber, exemplar),
		"setName":                    vm.NewTypedCFunction(MessageSetName, exemplar),
		"setNext":                    vm.NewTypedCFunction(MessageSetNext, exemplar),
		"type":                       vm.NewString("Message"),
	}
	slots["opShuffleC"] = slots["opShuffle"]
	SetSlot(vm.Core, "Message", &Message{Object: *vm.ObjectWith(slots)})
}

// MessageAppendArg is a Message method.
//
// appendArg adds a message as an argument to the message.
func MessageAppendArg(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	nm, ok := r.(*Message)
	if !ok {
		return vm.RaiseException("argument 0 to appendArg must be Message, not " + vm.TypeName(r))
	}
	m.Args = append(m.Args, nm)
	return target
}

// MessageAppendCachedArg is a Message method.
//
// appendCachedArg adds a value as an argument to the message.
func MessageAppendCachedArg(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	m.Args = append(m.Args, vm.CachedMessage(r))
	return target
}

// MessageArgAt is a Message method.
//
// argAt returns the nth argument, or nil if out of bounds.
func MessageArgAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	r := m.ArgAt(int(n.Value))
	if r != nil {
		return r
	}
	return vm.Nil
}

// MessageArgCount is a Message method.
//
// argCount returns the number of arguments to the message.
func MessageArgCount(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	return vm.NewNumber(float64(m.ArgCount()))
}

// MessageArgsEvaluatedIn is a Message method.
//
// argsEvaluatedIn returns a list containing the message arguments evaluated in
// the context of the given object.
func MessageArgsEvaluatedIn(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	ctx, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return ctx
	}
	l := make([]Interface, m.ArgCount())
	for k, v := range m.Args {
		r, ok := CheckStop(v.Eval(vm, ctx), LoopStops)
		if !ok {
			return r
		}
		l[k] = r
	}
	return vm.NewList(l...)
}

// MessageArguments is a Message method.
//
// arguments returns a list of the arguments to the message as messages.
func MessageArguments(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	l := make([]Interface, m.ArgCount())
	for k, v := range m.Args {
		l[k] = v
	}
	return vm.NewList(l...)
}

// MessageAsMessageWithEvaluatedArgs is a Message method.
//
// asMessageWithEvaluatedArgs creates a copy of the message with its arguments
// evaluated.
func MessageAsMessageWithEvaluatedArgs(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	nm := &Message{
		Object: *vm.CoreInstance("Message"),
		Text:   m.Text,
		Args:   make([]*Message, m.ArgCount()),
		Next:   m.Next,
		Prev:   m.Prev,
	}
	if msg.ArgCount() > 0 {
		r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
		if !ok {
			return r
		}
		locals = r
	}
	for k, v := range m.Args {
		r, ok := CheckStop(v.Eval(vm, locals), LoopStops)
		if !ok {
			return r
		}
		nm.Args[k] = vm.CachedMessage(r)
	}
	return nm
}

// MessageAsString is a Message method.
//
// asString creates a string representation of an object.
func MessageAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	b := bytes.Buffer{}
	target.(*Message).stringRecurse(vm, &b)
	return vm.NewString(b.String())
}

// MessageCachedResult is a Message method.
//
// cachedResult returns the cached value to which the message evaluates, or nil
// if there is not one, though this may also mean that nil is cached.
func MessageCachedResult(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	if m.Memo == nil {
		return vm.Nil
	}
	return m.Memo
}

// MessageCharacterNumber is a Message method.
//
// characterNumber returns the column number of the character within the line
// at which the message was parsed.
func MessageCharacterNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(target.(*Message).Col))
}

// MessageClone is a Message method.
//
// clone creates a deep copy of the message.
func MessageClone(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Message).DeepCopy()
}

// MessageDoInContext is a Message method.
//
// doInContext evaluates the message in the context of the given object,
// optionally with a given locals. If the locals aren't given, the context is
// the locals.
func MessageDoInContext(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	ctx, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return ctx
	}
	if msg.ArgCount() > 1 {
		locals, ok = CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
		if !ok {
			return locals
		}
	} else {
		locals = ctx
	}
	r, _ := CheckStop(m.Send(vm, ctx, locals), LoopStops)
	return r
}

// MessageFromString is a Message method.
//
// fromString parses the string into a message chain.
func MessageFromString(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	m, err := vm.Parse(strings.NewReader(s.String()), "<string>")
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return m
}

// MessageHasCachedResult is a Message method.
//
// hasCachedResult returns whether the message has a cached value to which the
// message will evaluate.
func MessageHasCachedResult(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(target.(*Message).Memo != nil)
}

// MessageIsEndOfLine is a Message method.
//
// isEndOfLine returns whether the message is a terminator.
func MessageIsEndOfLine(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(target.(*Message).IsTerminator())
}

// MessageLabel is a Message method.
//
// label returns the message's label, typically the name of the file from which
// it was parsed.
func MessageLabel(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(target.(*Message).Label)
}

// MessageLast is a Message method.
//
// last returns the last message in the chain.
func MessageLast(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	for m.Next != nil {
		m = m.Next
	}
	return m
}

// MessageLastBeforeEndOfLine is a Message method.
//
// lastBeforeEndOfLine returns the last message in the chain before a
// terminator.
func MessageLastBeforeEndOfLine(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	for !m.Next.IsTerminator() {
		m = m.Next
	}
	return m
}

// MessageLineNumber is a Message method.
//
// lineNumber returns the line number at which the message was parsed.
func MessageLineNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(target.(*Message).Line))
}

// MessageName is a Message method.
//
// name returns the name of the message.
func MessageName(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	return vm.NewString(m.Name())
}

// MessageNext is a Message method.
//
// next returns the next message in the chain, or nil if this is the last one.
func MessageNext(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	if m.Next != nil {
		return m.Next
	}
	return vm.Nil
}

// MessageNextIgnoreEndOfLines is a Message method.
//
// nextIgnoreEndOfLines returns the next message in the chain, skipping
// terminators, or nil if this is the last such message.
func MessageNextIgnoreEndOfLines(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	// I think the original returns the terminator at the end of the chain if
	// that is encountered, but that seems like a breach of contract.
	for m.Next.IsTerminator() {
		if m.Next == nil {
			return vm.Nil
		}
		m = m.Next
	}
	return m
}

// MessageOpShuffle is a Message method.
//
// opShuffle performs operator precedence shuffling on the message using the
// message's OperatorTable.
func MessageOpShuffle(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return m
}

// MessagePrevious is a Message method.
//
// previous returns the previous message in the chain.
func MessagePrevious(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	if m.Prev == nil {
		return vm.Nil
	}
	return m.Prev
}

// MessageRemoveCachedResult is a Message method.
//
// removeCachedResult removes the cached value to which the message will
// evaluate, causing it to send to its receiver normally.
func MessageRemoveCachedResult(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	m.Memo = nil
	return target
}

// MessageSetArguments is a Message method.
//
// setArguments sets the message's arguments to deep copies of the messages in
// the list argument.
func MessageSetArguments(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	l, stop := msg.ListArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	args := make([]*Message, len(l.Value))
	for k, v := range l.Value {
		arg, ok := v.(*Message)
		if !ok {
			return vm.RaiseException("argument to setArguments must be a list of only messages")
		}
		args[k] = arg
	}
	m.Args = args
	return m
}

// MessageSetCachedResult is a Message method.
//
// setCachedResult sets the message's cached value, causing it to evaluate to
// that value instead of sending to its receiver.
func MessageSetCachedResult(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	m.Memo = r
	return m
}

// MessageSetCharacterNumber is a Message method.
//
// setCharacterNumber sets the character number of the message, typically the
// column number within the line at which the message was parsed.
func MessageSetCharacterNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	m.Col = int(n.Value)
	return target
}

// MessageSetLabel is a Message method.
//
// setLabel sets the label of the message to the given string.
func MessageSetLabel(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	m.Label = s.String()
	return target
}

// MessageSetLineNumber is a Message method.
//
// setLineNumber sets the line number of the message to the given integer.
func MessageSetLineNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	m.Line = int(n.Value)
	return target
}

// MessageSetName is a Message method.
//
// setName sets the message name to the given string.
func MessageSetName(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	m.Text = s.String()
	return m
}

// MessageSetNext is a Message method.
//
// setNext sets the next message in the chain. That message's previous link
// will be set to this message, if non-nil.
func MessageSetNext(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Message)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	if r == vm.Nil {
		m.Next = nil
		return target
	}
	nm, ok := r.(*Message)
	if !ok {
		return vm.RaiseExceptionf("argument 0 to setNext must be Message, not %s", vm.TypeName(r))
	}
	m.Next = nm
	nm.Prev = m
	return target
}
