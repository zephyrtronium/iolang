package internal

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// A Message is the fundamental syntactic element and functionality of Io.
//
// NOTE: Unlike most other primitive types in iolang, Message values are NOT
// synchronized. It is a race condition to modify a message that might be in
// use, such as 'call message' or any message object in a scope other than the
// locals of the innermost currently executing block.
type Message struct {
	// Text is the name of this message.
	Text string
	// Args are the message's argument messages.
	Args []*Message
	// Next and Prev are links to the following and previous messages.
	Next, Prev *Message

	// Memo is the message's cached value. If non-nil, this is used instead of
	// performing the message.
	Memo *Object

	// Label is the message's label, generally the name of the file from which
	// it was parsed, if any.
	Label string
	// Line and Col are the one-based line and column numbers within the file
	// at which the message was parsed.
	Line, Col int
}

// tagMessage is the Tag type for Message objects.
type tagMessage struct{}

func (tagMessage) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	ok, proto := vm.GetSlot(self, "isActivatable")
	if proto == nil || !vm.AsBool(ok) {
		return self
	}
	return vm.Stop(vm.Perform(target, locals, self.Value.(*Message)))
}

func (tagMessage) CloneValue(value interface{}) interface{} {
	m := value.(*Message)
	return &Message{Text: m.Text, Label: m.Label}
}

// String returns "Message".
func (tagMessage) String() string {
	return "Message"
}

// MessageTag is the Tag for Message objects. Activate performs the message as
// an inline method if its isActivatable slot evaluates to true; otherwise, it
// returns self. CloneValue creates a new Message with the same text and label.
// (The Message proto has a custom clone method that does the right thing.)
var MessageTag tagMessage

// IdentMessage creates a message of a given identifier. Additional messages
// may be passed as arguments.
func (vm *VM) IdentMessage(s string, args ...*Message) *Message {
	return &Message{
		Text: s,
		Args: args,
	}
}

// StringMessage creates a message carrying a string value.
func (vm *VM) StringMessage(s string) *Message {
	return &Message{
		Text: strconv.Quote(s),
		Memo: vm.NewString(s),
	}
}

// NumberMessage creates a message carrying a numeric value.
func (vm *VM) NumberMessage(v float64) *Message {
	return &Message{
		Text: strconv.FormatFloat(v, 'g', -1, 64),
		Memo: vm.NewNumber(v),
	}
}

// CachedMessage creates a message carrying a cached value.
func (vm *VM) CachedMessage(v *Object) *Message {
	return &Message{
		Text: vm.AsString(v),
		Memo: v,
	}
}

// MessageObject returns an Object with the given Message value. If msg is nil,
// the result is nil.
func (vm *VM) MessageObject(msg *Message) *Object {
	if msg == nil {
		return nil
	}
	return vm.ObjectWith(nil, vm.CoreProto("Message"), msg, MessageTag)
}

// DeepCopy creates a copy of the message linked to copies of each message
// forward.
func (m *Message) DeepCopy() *Message {
	if m == nil {
		return nil
	}
	fm := &Message{
		Text: m.Text,
		Args: make([]*Message, len(m.Args)),
		Prev: m.Prev,
		Memo: m.Memo,
	}
	for i, arg := range m.Args {
		fm.Args[i] = arg.DeepCopy()
	}
	for pm, nm := fm, m.Next; nm != nil; pm, nm = pm.Next, nm.Next {
		pm.Next = &Message{
			Text: nm.Text,
			Args: make([]*Message, len(nm.Args)),
			Prev: pm,
			Memo: nm.Memo,
		}
		for i, arg := range nm.Args {
			pm.Next.Args[i] = arg.DeepCopy()
		}
	}
	return fm
}

// ArgCount returns the number of arguments to the message.
func (m *Message) ArgCount() int {
	if m == nil {
		return 0
	}
	return len(m.Args)
}

// AssertArgCount returns an error if the message does not have the given
// number of arguments. name is the name of the message used in the generated
// error message.
func (m *Message) AssertArgCount(name string, n int) error {
	if m.ArgCount() != n {
		return fmt.Errorf("%s must have %d arguments", name, n)
	}
	return nil
}

// ArgAt returns the argument at position n, or nil if the position is out of
// bounds.
func (m *Message) ArgAt(n int) (r *Message) {
	if 0 <= n && n < m.ArgCount() {
		// m is guaranteed to be non-nil because ArgCount >= 1.
		r = m.Args[n]
	}
	return r
}

// MessageArgAt evaluates the nth argument and returns it as a Message. If a
// stop occurs during evaluation, the Message will be nil, and the stop status
// and result will be returned. If the evaluated result is not a Message, the
// result will be nil, and an exception will be returned with an ExceptionStop.
func (m *Message) MessageArgAt(vm *VM, locals *Object, n int) (*Message, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		msg, ok := v.Value.(*Message)
		v.Unlock()
		if ok {
			return msg, nil, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Message, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return nil, v, s
}

// EvalArgAt evaluates the nth argument.
func (m *Message) EvalArgAt(vm *VM, locals *Object, n int) (result *Object, control Stop) {
	return m.ArgAt(n).Eval(vm, locals)
}

// Eval evaluates a message in the context of the given VM. This is a proxy to
// Send using locals as the target.
//
// NOTE: It is unsafe to call this while holding the lock of any object.
func (m *Message) Eval(vm *VM, locals *Object) (result *Object, control Stop) {
	return m.Send(vm, locals, locals)
}

// Send evaluates a message in the context of the given VM, targeting an
// object. After each message in the chain, this checks the VM's Control
// channel and returns if there is a waiting signal.
//
// NOTE: It is unsafe to call this while holding the lock of any object.
func (m *Message) Send(vm *VM, target, locals *Object) (result *Object, control Stop) {
	firstTarget := target
	for m != nil {
		if m.Memo != nil {
			// If there is a memo, the message automatically becomes it instead
			// of performing.
			result = m.Memo
			target = result
		} else if !m.IsTerminator() {
			result, control = vm.Perform(target, locals, m)
			if control != NoStop {
				return result, control
			}
			target = result
		} else {
			target = firstTarget
		}
		next := m.Next
		m = next
	}
	if result == nil {
		result = vm.Nil
	}
	return result, NoStop
}

// Perform executes a single message and checks for control flow signals. Any
// received control flow except NoStop and ResumeStop overrides the perform
// result.
//
// NOTE: It is unsafe to call this while holding the lock of any object.
func (vm *VM) Perform(target, locals *Object, msg *Message) (result *Object, control Stop) {
	vm.DebugMessage(target, locals, msg)
	var v, proto *Object
	if v, proto = vm.GetSlot(target, msg.Text); proto == nil {
		var forward, fp *Object
		if forward, fp = vm.GetSlot(target, "forward"); fp == nil {
			return vm.NewExceptionf("%v does not respond to %s", vm.TypeName(target), msg.Name()), ExceptionStop
		}
		v, proto = forward, fp
	}
	// We always activate and then check vm.Control, rather than making
	// activating the select default, because we want to catch control flow
	// from this activation as well.
	result = v.Activate(vm, target, locals, proto, msg)
	if result == nil {
		result = vm.Nil
	}
	select {
	case stop := <-vm.Control:
		switch stop.Control {
		case NoStop, ResumeStop:
			// Yield.
			runtime.Gosched()
		case ContinueStop, BreakStop, ReturnStop, ExceptionStop, ExitStop:
			// Return the stop.
			return stop.Result, stop.Control
		case PauseStop:
			result, control = vm.doPause(result)
		default:
			panic(fmt.Sprintf("invalid status in received stop %#v", stop))
		}
	default: // No waiting stop; continue as normal.
	}
	return result, control
}

// doPause handles a PauseStop. Returns any RemoteStop with real control flow
// received, otherwise (result, NoStop) if pause/resume concluded normally.
func (vm *VM) doPause(result *Object) (*Object, Stop) {
	vm.Sched.pause <- vm
	for {
		switch stop := <-vm.Control; stop.Control {
		case NoStop, PauseStop: // do nothing
		case ContinueStop, BreakStop, ReturnStop, ExceptionStop:
			vm.Sched.Start(vm)
			return stop.Result, stop.Control
		case ExitStop:
			return nil, ExitStop
		case ResumeStop:
			// Add ourselves back into the scheduler, then check whether we
			// have any real control flow waiting. We get one chance, otherwise
			// we assume NoStop.
			vm.Sched.Start(vm)
			runtime.Gosched()
			select {
			case stop = <-vm.Control:
				switch stop.Control {
				case NoStop, ResumeStop: // do nothing
				case ContinueStop, BreakStop, ReturnStop, ExceptionStop, ExitStop:
					// Return the stop.
					return stop.Result, stop.Control
				case PauseStop:
					// Return normal, but pause after the next-ish message this
					// coroutine performs. Resend the stop from a separate
					// goroutine to make sure we continue.
					go func() { vm.Control <- stop }()
				}
			default: // do nothing
			}
			return result, NoStop
		default:
			panic(fmt.Sprintf("iolang: invalid status in received stop %#v", stop))
		}
	}
}

// InsertAfter links another message to follow this one.
func (m *Message) InsertAfter(next *Message) {
	if m == nil {
		return
	}
	if m.Next != nil {
		m.Next.Prev = next
	}
	if next != nil {
		next.Next = m.Next
		next.Prev = m
	}
	m.Next = next
}

// IsStart determines whether this message is the start of a "statement." This
// is true if it has no previous link or if the previous link is a terminator.
// If m is nil, then m has no previous link, hence this returns true.
func (m *Message) IsStart() bool {
	if m == nil {
		return true
	}
	return m.Prev.IsTerminator()
}

// IsTerminator determines whether this message is the end of an expression.
// This is true if it is nil or it is a semicolon or newline.
func (m *Message) IsTerminator() bool {
	if m == nil {
		return true
	}
	return m.Text == ";" || m.Text == "\n"
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
	slots := Slots{
		"appendArg":                  vm.NewCFunction(MessageAppendArg, MessageTag),
		"appendCachedArg":            vm.NewCFunction(MessageAppendCachedArg, MessageTag),
		"argAt":                      vm.NewCFunction(MessageArgAt, MessageTag),
		"argCount":                   vm.NewCFunction(MessageArgCount, MessageTag),
		"argsEvaluatedIn":            vm.NewCFunction(MessageArgsEvaluatedIn, MessageTag),
		"arguments":                  vm.NewCFunction(MessageArguments, MessageTag),
		"asMessageWithEvaluatedArgs": vm.NewCFunction(MessageAsMessageWithEvaluatedArgs, MessageTag),
		"asString":                   vm.NewCFunction(MessageAsString, MessageTag),
		"cachedResult":               vm.NewCFunction(MessageCachedResult, MessageTag),
		"characterNumber":            vm.NewCFunction(MessageCharacterNumber, MessageTag),
		"clone":                      vm.NewCFunction(MessageClone, MessageTag),
		"doInContext":                vm.NewCFunction(MessageDoInContext, MessageTag),
		"fromString":                 vm.NewCFunction(MessageFromString, nil),
		"hasCachedResult":            vm.NewCFunction(MessageHasCachedResult, MessageTag),
		"isEndOfLine":                vm.NewCFunction(MessageIsEndOfLine, MessageTag),
		"label":                      vm.NewCFunction(MessageLabel, MessageTag),
		"last":                       vm.NewCFunction(MessageLast, MessageTag),
		"lastBeforeEndOfLine":        vm.NewCFunction(MessageLastBeforeEndOfLine, MessageTag),
		"lineNumber":                 vm.NewCFunction(MessageLineNumber, MessageTag),
		"name":                       vm.NewCFunction(MessageName, MessageTag),
		"next":                       vm.NewCFunction(MessageNext, MessageTag),
		"nextIgnoreEndOfLines":       vm.NewCFunction(MessageNextIgnoreEndOfLines, MessageTag),
		"opShuffle":                  vm.NewCFunction(MessageOpShuffle, MessageTag),
		"previous":                   vm.NewCFunction(MessagePrevious, MessageTag),
		"removeCachedResult":         vm.NewCFunction(MessageRemoveCachedResult, MessageTag),
		"setArguments":               vm.NewCFunction(MessageSetArguments, MessageTag),
		"setCachedResult":            vm.NewCFunction(MessageSetCachedResult, MessageTag),
		"setCharacterNumber":         vm.NewCFunction(MessageSetCharacterNumber, MessageTag),
		"setLabel":                   vm.NewCFunction(MessageSetLabel, MessageTag),
		"setLineNumber":              vm.NewCFunction(MessageSetLineNumber, MessageTag),
		"setName":                    vm.NewCFunction(MessageSetName, MessageTag),
		"setNext":                    vm.NewCFunction(MessageSetNext, MessageTag),
		"type":                       vm.NewString("Message"),
	}
	slots["opShuffleC"] = slots["opShuffle"]
	vm.coreInstall("Message", slots, &Message{Memo: vm.Nil}, MessageTag)
}

// MessageAppendArg is a Message method.
//
// appendArg adds a message as an argument to the message.
func MessageAppendArg(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	m := target.Value.(*Message)
	target.Unlock()
	nm, exc, stop := msg.MessageArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m.Args = append(m.Args, nm)
	return target
}

// MessageAppendCachedArg is a Message method.
//
// appendCachedArg adds a value as an argument to the message.
func MessageAppendCachedArg(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	m.Args = append(m.Args, vm.CachedMessage(r))
	return target
}

// MessageArgAt is a Message method.
//
// argAt returns the nth argument, or nil if out of bounds.
func MessageArgAt(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	r := m.ArgAt(int(n))
	return vm.MessageObject(r)
}

// MessageArgCount is a Message method.
//
// argCount returns the number of arguments to the message.
func MessageArgCount(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(float64(target.Value.(*Message).ArgCount()))
}

// MessageArgsEvaluatedIn is a Message method.
//
// argsEvaluatedIn returns a list containing the message arguments evaluated in
// the context of the given object.
func MessageArgsEvaluatedIn(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	ctx, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(ctx, stop)
	}
	l := make([]*Object, m.ArgCount())
	for k, v := range m.Args {
		r, stop := v.Eval(vm, ctx)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		l[k] = r
	}
	return vm.NewList(l...)
}

// MessageArguments is a Message method.
//
// arguments returns a list of the arguments to the message as messages.
func MessageArguments(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	l := make([]*Object, m.ArgCount())
	for k, v := range m.Args {
		l[k] = vm.MessageObject(v)
	}
	return vm.NewList(l...)
}

// MessageAsMessageWithEvaluatedArgs is a Message method.
//
// asMessageWithEvaluatedArgs creates a copy of the message with its arguments
// evaluated.
func MessageAsMessageWithEvaluatedArgs(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	nm := &Message{
		Text: m.Text,
		Args: make([]*Message, m.ArgCount()),
		Next: m.Next,
		Prev: m.Prev,
	}
	if msg.ArgCount() > 0 {
		r, stop := msg.EvalArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		locals = r
	}
	for k, v := range m.Args {
		r, stop := v.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		nm.Args[k] = vm.CachedMessage(r)
	}
	return vm.MessageObject(nm)
}

// MessageAsString is a Message method.
//
// asString creates a string representation of an object.
func MessageAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	b := bytes.Buffer{}
	m := target.Value.(*Message)
	m.stringRecurse(vm, &b)
	return vm.NewString(b.String())
}

// MessageCachedResult is a Message method.
//
// cachedResult returns the cached value to which the message evaluates, or nil
// if there is not one, though this may also mean that nil is cached.
func MessageCachedResult(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return m.Memo
}

// MessageCharacterNumber is a Message method.
//
// characterNumber returns the column number of the character within the line
// at which the message was parsed.
func MessageCharacterNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return vm.NewNumber(float64(m.Col))
}

// MessageClone is a Message method.
//
// clone creates a deep copy of the message.
func MessageClone(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.MessageObject(target.Value.(*Message).DeepCopy())
}

// MessageDoInContext is a Message method.
//
// doInContext evaluates the message in the context of the given object,
// optionally with a given locals. If the locals aren't given, the context is
// the locals.
func MessageDoInContext(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	ctx, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(ctx, stop)
	}
	if msg.ArgCount() > 1 {
		locals, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(locals, stop)
		}
	} else {
		locals = ctx
	}
	return vm.Stop(m.Send(vm, ctx, locals))
}

// MessageFromString is a Message method.
//
// fromString parses the string into a message chain.
func MessageFromString(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m, err := vm.Parse(strings.NewReader(s), "<string>")
	if err != nil {
		return vm.IoError(err)
	}
	return vm.MessageObject(m)
}

// MessageHasCachedResult is a Message method.
//
// hasCachedResult returns whether the message has a cached value to which the
// message will evaluate.
func MessageHasCachedResult(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return vm.IoBool(m.Memo != nil)
}

// MessageIsEndOfLine is a Message method.
//
// isEndOfLine returns whether the message is a terminator.
func MessageIsEndOfLine(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return vm.IoBool(m.IsTerminator())
}

// MessageLabel is a Message method.
//
// label returns the message's label, typically the name of the file from which
// it was parsed.
func MessageLabel(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return vm.NewString(m.Label)
}

// MessageLast is a Message method.
//
// last returns the last message in the chain.
func MessageLast(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return target
	}
	if m.Next == nil {
		return target
	}
	for m.Next != nil {
		next := m.Next
		m = next
	}
	return vm.MessageObject(m)
}

// MessageLastBeforeEndOfLine is a Message method.
//
// lastBeforeEndOfLine returns the last message in the chain before a
// terminator.
func MessageLastBeforeEndOfLine(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return target
	}
	if m.Next.IsTerminator() {
		return target
	}
	for !m.Next.IsTerminator() {
		next := m.Next
		m = next
	}
	return vm.MessageObject(m)
}

// MessageLineNumber is a Message method.
//
// lineNumber returns the line number at which the message was parsed.
func MessageLineNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	return vm.NewNumber(float64(m.Line))
}

// MessageName is a Message method.
//
// name returns the name of the message.
func MessageName(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(target.Value.(*Message).Name())
}

// MessageNext is a Message method.
//
// next returns the next message in the chain, or nil if this is the last one.
func MessageNext(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return vm.Nil
	}
	return vm.MessageObject(m.Next)
}

// MessageNextIgnoreEndOfLines is a Message method.
//
// nextIgnoreEndOfLines returns the next message in the chain, skipping
// terminators, or nil if this is the last such message.
func MessageNextIgnoreEndOfLines(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return target
	}
	if !m.Next.IsTerminator() {
		next := m.Next
		return vm.MessageObject(next)
	}
	// I think the original returns the terminator at the end of the chain if
	// that is encountered, but that seems like a breach of contract.
	for m.Next.IsTerminator() {
		next := m.Next
		m = next
	}
	return vm.MessageObject(m)
}

// MessageOpShuffle is a Message method.
//
// opShuffle performs operator precedence shuffling on the message using the
// message's OperatorTable.
func MessageOpShuffle(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return target
	}
	if err := vm.OpShuffle(target); err != nil {
		return vm.RaiseException(err)
	}
	return target
}

// MessagePrevious is a Message method.
//
// previous returns the previous message in the chain.
func MessagePrevious(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return vm.Nil
	}
	return vm.MessageObject(m.Prev)
}

// MessageRemoveCachedResult is a Message method.
//
// removeCachedResult removes the cached value to which the message will
// evaluate, causing it to send to its receiver normally.
func MessageRemoveCachedResult(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m != nil {
		m.Memo = nil
	}
	return target
}

// MessageSetArguments is a Message method.
//
// setArguments sets the message's arguments to deep copies of the messages in
// the list argument.
func MessageSetArguments(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	if m == nil {
		return target
	}
	l, obj, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	obj.Lock()
	args := make([]*Message, len(l))
	for k, v := range l {
		arg, ok := v.Value.(*Message)
		if !ok {
			obj.Unlock()
			return vm.RaiseExceptionf("argument to setArguments must be a list of only messages")
		}
		args[k] = arg
	}
	obj.Unlock()
	m.Args = args
	return target
}

// MessageSetCachedResult is a Message method.
//
// setCachedResult sets the message's cached value, causing it to evaluate to
// that value instead of sending to its receiver.
func MessageSetCachedResult(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	m.Memo = r
	return target
}

// MessageSetCharacterNumber is a Message method.
//
// setCharacterNumber sets the character number of the message, typically the
// column number within the line at which the message was parsed.
func MessageSetCharacterNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m.Col = int(n)
	return target
}

// MessageSetLabel is a Message method.
//
// setLabel sets the label of the message to the given string.
func MessageSetLabel(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m.Label = s
	return target
}

// MessageSetLineNumber is a Message method.
//
// setLineNumber sets the line number of the message to the given integer.
func MessageSetLineNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m.Line = int(n)
	return target
}

// MessageSetName is a Message method.
//
// setName sets the message name to the given string.
func MessageSetName(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m.Text = s
	return target
}

// MessageSetNext is a Message method.
//
// setNext sets the next message in the chain. That message's previous link
// will be set to this message, if non-nil.
func MessageSetNext(vm *VM, target, locals *Object, msg *Message) *Object {
	m := target.Value.(*Message)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	if r == vm.Nil {
		m.Next = nil
		return target
	}
	nm, ok := r.Value.(*Message)
	if !ok {
		return vm.RaiseExceptionf("argument 0 to setNext must be Message, not %s", vm.TypeName(r))
	}
	m.Next = nm
	nm.Prev = m
	return target
}
