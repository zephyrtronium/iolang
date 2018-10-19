package iolang

import (
	"bytes"
	"fmt"
	"strconv"
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

	// Cached value of this message. If this is non-nil, it is used instead of
	// activating the message.
	Memo Interface
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

// NumberArgAt evaluates the nth argument and returns it as a Number. If it is
// not a Number, then the result is nil, and an error is returned.
func (m *Message) NumberArgAt(vm *VM, locals Interface, n int) (*Number, error) {
	v := m.EvalArgAt(vm, locals, n)
	if num, ok := v.(*Number); ok {
		return num, nil
	}
	// Not the expected type, so return an error.
	if err, ok := v.(error); ok && !IsIoError(err) {
		return nil, err
	}
	return nil, vm.NewExceptionf("argument %d to %s must be of type Number, not %s", n, m.Text, vm.TypeName(v))
}

// StringArgAt evaluates the nth argument and returns it as a String. If it is
// not a String, then the result is nil, and an error is returned.
func (m *Message) StringArgAt(vm *VM, locals Interface, n int) (*String, error) {
	v := m.EvalArgAt(vm, locals, n)
	if str, ok := v.(*String); ok {
		return str, nil
	}
	// Not the expected type, so return an error.
	if err, ok := v.(error); ok && !IsIoError(err) {
		return nil, err
	}
	return nil, vm.NewExceptionf("argument %d to %s must be of type Sequence, not %s", n, m.Text, vm.TypeName(v))
}

// ListArgAt evaluates the nth argument and returns it as a List. If it is not
// a List, then the result is nil, and an error is returned.
func (m *Message) ListArgAt(vm *VM, locals Interface, n int) (*List, error) {
	v := m.EvalArgAt(vm, locals, n)
	if lst, ok := v.(*List); ok {
		return lst, nil
	}
	// Not the expected type, so return an error.
	if err, ok := v.(error); ok && !IsIoError(err) {
		return nil, err
	}
	return nil, vm.NewExceptionf("argument %d to %s must be of type List, not %s", n, m.Text, vm.TypeName(v))
}

// AsStringArgAt evaluates the nth argument, then activates its asString slot
// for a string representation. If the result is not a string, then the result
// is nil, and an error is returned.
func (m *Message) AsStringArgAt(vm *VM, locals Interface, n int) (*String, error) {
	v := m.EvalArgAt(vm, locals, n)
	if asString, proto := GetSlot(v, "asString"); proto != nil {
		switch rr := asString.(type) {
		case Actor:
			if str, ok := vm.SimpleActivate(rr, v, locals, "asString").(*String); ok {
				return str, nil
			}
		case *String:
			return rr, nil
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
	result = target
	for m != nil {
		if m.Memo != nil {
			// It is the parser's responsibility to set memos for literals.
			result = m.Memo
			target = result
		} else {
			if !m.IsTerminator() {
				if newtarget, proto := GetSlot(target, m.Text); proto != nil {
					// We have the slot.
					switch a := newtarget.(type) {
					case Stop:
						a.Result = result
						return a
					case Actor:
						var ok bool
						result, ok = CheckStop(a.Activate(vm, target, locals, m), NoStop)
						if !ok {
							return result
						}
					default:
						result = newtarget
					}
				} else if forward, fp := GetSlot(target, "forward"); fp != nil {
					if a, ok := forward.(Actor); ok {
						result = a.Activate(vm, target, locals, m)
					} else {
						return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Text)
					}
				} else {
					return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Text)
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
	slots := Slots{
		"asString": vm.NewTypedCFunction(MessageAsString),
		"type":     vm.NewString("Message"),
	}
	SetSlot(vm.Core, "Message", &Message{Object: *vm.ObjectWith(slots)})
}

// MessageAsString is a Message method.
//
// asString creates a string representation of an object.
func MessageAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	b := bytes.Buffer{}
	target.(*Message).stringRecurse(vm, &b)
	return vm.NewString(b.String())
}
