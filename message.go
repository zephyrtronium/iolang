package iolang

import (
	"bytes"
	"fmt"
	"runtime"
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
func (m *Message) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return m, NoStop
}

// Clone returns a clone of the message with the same text only.
func (m *Message) Clone() Interface {
	return &Message{
		Object: Object{Protos: []Interface{m}},
		Text:   m.Text,
	}
}

// IdentMessage creates a message of a given identifier. Additional messages
// may be passed as arguments.
func (vm *VM) IdentMessage(s string, args ...*Message) *Message {
	return &Message{
		Object: Object{Protos: vm.CoreProto("Message")},
		Text:   s,
		Args:   args,
	}
}

// StringMessage creates a message carrying a string value.
func (vm *VM) StringMessage(s string) *Message {
	return &Message{
		Object: Object{Protos: vm.CoreProto("Message")},
		Text:   strconv.Quote(s),
		Memo:   vm.NewString(s),
	}
}

// NumberMessage creates a message carrying a numeric value.
func (vm *VM) NumberMessage(v float64) *Message {
	return &Message{
		Object: Object{Protos: vm.CoreProto("Message")},
		Text:   strconv.FormatFloat(v, 'g', -1, 64),
		Memo:   vm.NewNumber(v),
	}
}

// CachedMessage creates a message carrying a cached value.
func (vm *VM) CachedMessage(v Interface) *Message {
	return &Message{
		Object: Object{Protos: vm.CoreProto("Message")},
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
	// We can't use vm.CoreProto because we won't have access to a VM
	// everywhere we need it, e.g. Block.Clone(). Instead, steal the protos
	// from the message we're copying.
	fm := &Message{
		Object: Object{Protos: append([]Interface{}, m.Protos...)},
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
			Object: Object{Protos: append([]Interface{}, nm.Protos...)},
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

// MessageArgAt evaluates the nth argument and returns it as a Message. If a
// stop occurs during evaluation, the Message will be nil, and the stop status
// and result will be returned. If the evaluated result is not a Message, the
// result will be nil, and an exception will be raised.
func (m *Message) MessageArgAt(vm *VM, locals Interface, n int) (*Message, Interface, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		if m, ok := v.(*Message); ok {
			return m, nil, NoStop
		}
		// Not the expected type, so return an error.
		v, s = vm.RaiseExceptionf("argument %d to %s must be List, not %s", n, m.Text, vm.TypeName(v))
	}
	return nil, v, s
}

// EvalArgAt evaluates the nth argument.
func (m *Message) EvalArgAt(vm *VM, locals Interface, n int) (result Interface, control Stop) {
	return m.ArgAt(n).Eval(vm, locals)
}

// Eval evaluates a message in the context of the given VM. This is a proxy to
// Send using locals as the target.
func (m *Message) Eval(vm *VM, locals Interface) (result Interface, control Stop) {
	return m.Send(vm, locals, locals)
}

// Send evaluates a message in the context of the given VM, targeting an
// object.
func (m *Message) Send(vm *VM, target, locals Interface) (result Interface, control Stop) {
	firstTarget := target
	for m != nil {
		select {
		case stop := <-vm.Stop:
			switch stop.Control {
			case NoStop, ResumeStop:
				// Yield.
				runtime.Gosched()
			case ContinueStop, BreakStop, ReturnStop:
				// Return the current value.
				if result == nil {
					result = vm.Nil
				}
				return result, NoStop
			case ExceptionStop:
				// Return the exception itself.
				return stop.Result, stop.Control
			case PauseStop:
				// Pause until we receive a ResumeStop.
				vm.Sched.pause <- vm
				for stop.Control != ResumeStop {
					switch stop = <-vm.Stop; stop.Control {
					case NoStop, PauseStop: // do nothing
					case ContinueStop, BreakStop, ReturnStop:
						if result == nil {
							result = vm.Nil
						}
						return result, NoStop
					case ExceptionStop:
						return stop.Result, stop.Control
					case ResumeStop:
						// Add ourselves back into the scheduler.
						vm.Sched.Start(vm)
					default:
						panic(fmt.Sprintf("invalid status in received stop %#v", stop))
					}
				}
			default:
				panic(fmt.Sprintf("invalid status in received stop %#v", stop))
			}
		default: // No waiting stop; continue as normal.
		}
		if m.Memo != nil {
			// If there is a memo, the message automatically becomes it instead
			// of performing.
			result = m.Memo
			target = result
		} else if !m.IsTerminator() {
			result, control = vm.Perform(target, locals, m)
			if control != NoStop {
				if control == ExceptionStop {
					if e, ok := result.(*Exception); ok {
						e.Stack = append(e.Stack, m)
					}
				}
				return result, control
			}
			target = result
		} else {
			target = firstTarget
		}
		m = m.Next
	}
	if result == nil {
		result = vm.Nil
	}
	return result, NoStop
}

// Perform executes a single message. The result may be a Stop.
func (vm *VM) Perform(target, locals Interface, msg *Message) (result Interface, control Stop) {
	if v, proto := vm.GetSlot(target, msg.Name()); proto != nil {
		x, s := v.Activate(vm, target, locals, proto, msg)
		if x != nil {
			return x, s
		}
		return vm.Nil, s
	}
	if forward, fp := vm.GetSlot(target, "forward"); fp != nil {
		x, s := forward.Activate(vm, target, locals, fp, msg)
		if x != nil {
			return x, s
		}
		return vm.Nil, s
	}
	return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), msg.Name())
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

// Name returns the name of the message, which is its text if it is non-nil.
func (m *Message) Name() string {
	if m != nil {
		return m.Text
	}
	return "<nil message>"
}

// String generates a diagnostic string representation of this message.
func (m *Message) String() string {
	return "message-" + m.Name()
}

func (m *Message) stringRecurse(vm *VM, b *bytes.Buffer) {
	if m == nil {
		b.WriteString("<nil message>")
		return
	}
	for m != nil {
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
		if !m.IsTerminator() && m.Next != nil {
			b.WriteByte(' ')
		}
		if m.Text == ";" {
			b.WriteByte('\n')
		}
		m = m.Next
	}
}

func (vm *VM) initMessage() {
	var kind *Message
	slots := Slots{
		"appendArg":                  vm.NewCFunction(MessageAppendArg, kind),
		"appendCachedArg":            vm.NewCFunction(MessageAppendCachedArg, kind),
		"argAt":                      vm.NewCFunction(MessageArgAt, kind),
		"argCount":                   vm.NewCFunction(MessageArgCount, kind),
		"argsEvaluatedIn":            vm.NewCFunction(MessageArgsEvaluatedIn, kind),
		"arguments":                  vm.NewCFunction(MessageArguments, kind),
		"asMessageWithEvaluatedArgs": vm.NewCFunction(MessageAsMessageWithEvaluatedArgs, kind),
		"asString":                   vm.NewCFunction(MessageAsString, kind),
		"cachedResult":               vm.NewCFunction(MessageCachedResult, kind),
		"characterNumber":            vm.NewCFunction(MessageCharacterNumber, kind),
		"clone":                      vm.NewCFunction(MessageClone, kind),
		"doInContext":                vm.NewCFunction(MessageDoInContext, kind),
		"fromString":                 vm.NewCFunction(MessageFromString, nil),
		"hasCachedResult":            vm.NewCFunction(MessageHasCachedResult, kind),
		"isEndOfLine":                vm.NewCFunction(MessageIsEndOfLine, kind),
		"label":                      vm.NewCFunction(MessageLabel, kind),
		"last":                       vm.NewCFunction(MessageLast, kind),
		"lastBeforeEndOfLine":        vm.NewCFunction(MessageLastBeforeEndOfLine, kind),
		"lineNumber":                 vm.NewCFunction(MessageLineNumber, kind),
		"name":                       vm.NewCFunction(MessageName, kind),
		"next":                       vm.NewCFunction(MessageNext, kind),
		"nextIgnoreEndOfLines":       vm.NewCFunction(MessageNextIgnoreEndOfLines, kind),
		"opShuffle":                  vm.NewCFunction(MessageOpShuffle, kind),
		"previous":                   vm.NewCFunction(MessagePrevious, kind),
		"removeCachedResult":         vm.NewCFunction(MessageRemoveCachedResult, kind),
		"setArguments":               vm.NewCFunction(MessageSetArguments, kind),
		"setCachedResult":            vm.NewCFunction(MessageSetCachedResult, kind),
		"setCharacterNumber":         vm.NewCFunction(MessageSetCharacterNumber, kind),
		"setLabel":                   vm.NewCFunction(MessageSetLabel, kind),
		"setLineNumber":              vm.NewCFunction(MessageSetLineNumber, kind),
		"setName":                    vm.NewCFunction(MessageSetName, kind),
		"setNext":                    vm.NewCFunction(MessageSetNext, kind),
		"type":                       vm.NewString("Message"),
	}
	slots["opShuffleC"] = slots["opShuffle"]
	vm.SetSlot(vm.Core, "Message", &Message{Object: *vm.ObjectWith(slots)})
}

// MessageAppendArg is a Message method.
//
// appendArg adds a message as an argument to the message.
func MessageAppendArg(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	nm, err, stop := msg.MessageArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m.Args = append(m.Args, nm)
	return target, NoStop
}

// MessageAppendCachedArg is a Message method.
//
// appendCachedArg adds a value as an argument to the message.
func MessageAppendCachedArg(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	m.Args = append(m.Args, vm.CachedMessage(r))
	return target, NoStop
}

// MessageArgAt is a Message method.
//
// argAt returns the nth argument, or nil if out of bounds.
func MessageArgAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r := m.ArgAt(int(n.Value))
	if r != nil {
		return r, NoStop
	}
	return vm.Nil, NoStop
}

// MessageArgCount is a Message method.
//
// argCount returns the number of arguments to the message.
func MessageArgCount(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	return vm.NewNumber(float64(m.ArgCount())), NoStop
}

// MessageArgsEvaluatedIn is a Message method.
//
// argsEvaluatedIn returns a list containing the message arguments evaluated in
// the context of the given object.
func MessageArgsEvaluatedIn(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	ctx, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return ctx, stop
	}
	l := make([]Interface, m.ArgCount())
	for k, v := range m.Args {
		r, stop := v.Eval(vm, ctx)
		if stop != NoStop {
			return r, stop
		}
		l[k] = r
	}
	return vm.NewList(l...), NoStop
}

// MessageArguments is a Message method.
//
// arguments returns a list of the arguments to the message as messages.
func MessageArguments(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	l := make([]Interface, m.ArgCount())
	for k, v := range m.Args {
		l[k] = v
	}
	return vm.NewList(l...), NoStop
}

// MessageAsMessageWithEvaluatedArgs is a Message method.
//
// asMessageWithEvaluatedArgs creates a copy of the message with its arguments
// evaluated.
func MessageAsMessageWithEvaluatedArgs(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	nm := &Message{
		Object: Object{Protos: vm.CoreProto("Message")},
		Text:   m.Text,
		Args:   make([]*Message, m.ArgCount()),
		Next:   m.Next,
		Prev:   m.Prev,
	}
	if msg.ArgCount() > 0 {
		r, stop := msg.EvalArgAt(vm, locals, 0)
		if stop != NoStop {
			return r, stop
		}
		locals = r
	}
	for k, v := range m.Args {
		r, stop := v.Eval(vm, locals)
		if stop != NoStop {
			return r, stop
		}
		nm.Args[k] = vm.CachedMessage(r)
	}
	return nm, NoStop
}

// MessageAsString is a Message method.
//
// asString creates a string representation of an object.
func MessageAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	b := bytes.Buffer{}
	target.(*Message).stringRecurse(vm, &b)
	return vm.NewString(b.String()), NoStop
}

// MessageCachedResult is a Message method.
//
// cachedResult returns the cached value to which the message evaluates, or nil
// if there is not one, though this may also mean that nil is cached.
func MessageCachedResult(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	if m.Memo == nil {
		return vm.Nil, NoStop
	}
	return m.Memo, NoStop
}

// MessageCharacterNumber is a Message method.
//
// characterNumber returns the column number of the character within the line
// at which the message was parsed.
func MessageCharacterNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(target.(*Message).Col)), NoStop
}

// MessageClone is a Message method.
//
// clone creates a deep copy of the message.
func MessageClone(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target.(*Message).DeepCopy(), NoStop
}

// MessageDoInContext is a Message method.
//
// doInContext evaluates the message in the context of the given object,
// optionally with a given locals. If the locals aren't given, the context is
// the locals.
func MessageDoInContext(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	ctx, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return ctx, stop
	}
	if msg.ArgCount() > 1 {
		locals, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return locals, stop
		}
	} else {
		locals = ctx
	}
	return m.Send(vm, ctx, locals)
}

// MessageFromString is a Message method.
//
// fromString parses the string into a message chain.
func MessageFromString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	m, err := vm.Parse(strings.NewReader(s.String()), "<string>")
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return m, NoStop
}

// MessageHasCachedResult is a Message method.
//
// hasCachedResult returns whether the message has a cached value to which the
// message will evaluate.
func MessageHasCachedResult(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(target.(*Message).Memo != nil), NoStop
}

// MessageIsEndOfLine is a Message method.
//
// isEndOfLine returns whether the message is a terminator.
func MessageIsEndOfLine(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(target.(*Message).IsTerminator()), NoStop
}

// MessageLabel is a Message method.
//
// label returns the message's label, typically the name of the file from which
// it was parsed.
func MessageLabel(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(target.(*Message).Label), NoStop
}

// MessageLast is a Message method.
//
// last returns the last message in the chain.
func MessageLast(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	for m.Next != nil {
		m = m.Next
	}
	return m, NoStop
}

// MessageLastBeforeEndOfLine is a Message method.
//
// lastBeforeEndOfLine returns the last message in the chain before a
// terminator.
func MessageLastBeforeEndOfLine(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	for !m.Next.IsTerminator() {
		m = m.Next
	}
	return m, NoStop
}

// MessageLineNumber is a Message method.
//
// lineNumber returns the line number at which the message was parsed.
func MessageLineNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(target.(*Message).Line)), NoStop
}

// MessageName is a Message method.
//
// name returns the name of the message.
func MessageName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	return vm.NewString(m.Name()), NoStop
}

// MessageNext is a Message method.
//
// next returns the next message in the chain, or nil if this is the last one.
func MessageNext(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	if m.Next != nil {
		return m.Next, NoStop
	}
	return vm.Nil, NoStop
}

// MessageNextIgnoreEndOfLines is a Message method.
//
// nextIgnoreEndOfLines returns the next message in the chain, skipping
// terminators, or nil if this is the last such message.
func MessageNextIgnoreEndOfLines(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	// I think the original returns the terminator at the end of the chain if
	// that is encountered, but that seems like a breach of contract.
	for m.Next.IsTerminator() {
		if m.Next == nil {
			return vm.Nil, NoStop
		}
		m = m.Next
	}
	return m, NoStop
}

// MessageOpShuffle is a Message method.
//
// opShuffle performs operator precedence shuffling on the message using the
// message's OperatorTable.
func MessageOpShuffle(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return m, NoStop
}

// MessagePrevious is a Message method.
//
// previous returns the previous message in the chain.
func MessagePrevious(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	if m.Prev == nil {
		return vm.Nil, NoStop
	}
	return m.Prev, NoStop
}

// MessageRemoveCachedResult is a Message method.
//
// removeCachedResult removes the cached value to which the message will
// evaluate, causing it to send to its receiver normally.
func MessageRemoveCachedResult(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	m.Memo = nil
	return target, NoStop
}

// MessageSetArguments is a Message method.
//
// setArguments sets the message's arguments to deep copies of the messages in
// the list argument.
func MessageSetArguments(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	l, err, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
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
	return m, NoStop
}

// MessageSetCachedResult is a Message method.
//
// setCachedResult sets the message's cached value, causing it to evaluate to
// that value instead of sending to its receiver.
func MessageSetCachedResult(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	m.Memo = r
	return m, NoStop
}

// MessageSetCharacterNumber is a Message method.
//
// setCharacterNumber sets the character number of the message, typically the
// column number within the line at which the message was parsed.
func MessageSetCharacterNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m.Col = int(n.Value)
	return target, NoStop
}

// MessageSetLabel is a Message method.
//
// setLabel sets the label of the message to the given string.
func MessageSetLabel(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m.Label = s.String()
	return target, NoStop
}

// MessageSetLineNumber is a Message method.
//
// setLineNumber sets the line number of the message to the given integer.
func MessageSetLineNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m.Line = int(n.Value)
	return target, NoStop
}

// MessageSetName is a Message method.
//
// setName sets the message name to the given string.
func MessageSetName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m.Text = s.String()
	return m, NoStop
}

// MessageSetNext is a Message method.
//
// setNext sets the next message in the chain. That message's previous link
// will be set to this message, if non-nil.
func MessageSetNext(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	if r == vm.Nil {
		m.Next = nil
		return target, NoStop
	}
	nm, ok := r.(*Message)
	if !ok {
		return vm.RaiseExceptionf("argument 0 to setNext must be Message, not %s", vm.TypeName(r))
	}
	m.Next = nm
	nm.Prev = m
	return target, NoStop
}
