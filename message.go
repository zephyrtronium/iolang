package iolang

import (
	"bytes"
	"fmt"
)

// A Message is the fundamental syntactic element and functionality of Io.
type Message struct {
	Object
	// Type and value of this message.
	Symbol Symbol
	// This message's arguments.
	Args []*Message
	// Next and previous messages.
	Next, Prev *Message

	// Cached value of this message. If this is non-nil, it is used instead of
	// activating the message.
	Memo Interface
}

// A Symbol is what I called the type and name of a message. This should change
// eventually.
type Symbol struct {
	Kind   SymKind
	Text   string  // Identifier text.
	Num    float64 // Numeric value.
	String string  // String value.
}

// SymKind is the type of a Message's Symbol.
type SymKind int

const (
	NoSym     SymKind = iota // Invalid symbol.
	SemiSym                  // Expression-terminating message.
	IdentSym                 // Identifier.
	NumSym                   // Numeric literal.
	StringSym                // String literal.
)

// IdentMessage creates a message of a given identifier. Additional messages
// may be passed as arguments.
func (vm *VM) IdentMessage(s string, args ...*Message) *Message {
	return &Message{
		Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}},
		Symbol: Symbol{Kind: IdentSym, Text: s},
		Args:   args,
	}
}

// StringMessage creates a message carrying a string value.
func (vm *VM) StringMessage(s string) *Message {
	return &Message{
		Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}},
		Symbol: Symbol{Kind: StringSym, String: s},
		Memo:   vm.NewString(s),
	}
}

// NumberMessage creates a message carrying a numeric value.
func (vm *VM) NumberMessage(v float64) *Message {
	return &Message{
		Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}},
		Symbol: Symbol{Kind: NumSym, Num: v},
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
	return nil, vm.NewExceptionf("argument %d to %s must be of type Number, not %s", n, m.Symbol.Text, vm.TypeName(v))
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
	return nil, vm.NewExceptionf("argument %d to %s must be of type Sequence, not %s", n, m.Symbol.Text, vm.TypeName(v))
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
	return nil, vm.NewExceptionf("argument %d to %s cannot be converted to string", n, m.Symbol.Text)
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
			result = m.Memo
		} else {
			switch m.Symbol.Kind {
			case IdentSym:
				if newtarget, proto := GetSlot(target, m.Symbol.Text); proto != nil {
					// We have the slot.
					switch a := newtarget.(type) {
					case Stop:
						a.Result = result
						return a
					case Actor:
						result = a.Activate(vm, target, locals, m)
					default:
						result = newtarget
					}
				} else if forward, fp := GetSlot(target, "forward"); fp != nil {
					if a, ok := forward.(Actor); ok {
						// result = vm.SimpleActivate(a, target, locals, "forward")
						result = a.Activate(vm, target, locals, m)
					} else {
						return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Symbol.Text)
					}
				} else {
					return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Symbol.Text)
				}
			case SemiSym:
				target = firstTarget
			case NumSym:
				// Numbers and strings should be in memo, but just in case.
				result = vm.NewNumber(m.Symbol.Num)
			case StringSym:
				result = vm.NewString(m.Symbol.String)
			default:
				panic(fmt.Sprintf("iolang: invalid Symbol: %#v", m.Symbol))
			}
			if result == nil {
				// No message should evaluate to something that is not an Io
				// object, so we want to convert nil to vm.Nil.
				result = vm.Nil
			}
		}
		if m.Symbol.Kind != SemiSym {
			target = result
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
	switch m.Symbol.Kind {
	case SemiSym, IdentSym:
		return m.Symbol.Text
	case NumSym:
		return fmt.Sprintf("%g", m.Symbol.Num)
	case StringSym:
		return m.Symbol.String
	default:
		panic(fmt.Sprintf("iolang: invalid Symbol: %#v", m.Symbol))
	}
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
			switch m.Symbol.Kind {
			case SemiSym:
				b.WriteString("; ")
			case IdentSym:
				b.WriteString(m.Symbol.Text)
				if len(m.Args) > 0 {
					b.WriteByte('(')
					m.Args[0].stringRecurse(vm, b)
					for _, arg := range m.Args[1:] {
						b.WriteString(", ")
						arg.stringRecurse(vm, b)
					}
					b.WriteByte(')')
				}
			case NumSym:
				fmt.Fprint(b, m.Symbol.Num)
			case StringSym:
				fmt.Fprintf(b, "%q", m.Symbol.String)
			default:
				panic("iolang: unknown symbol kind")
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
		"asString": vm.NewTypedCFunction(MessageAsString, "MessageAsString()"),
	}
	vm.DefaultSlots["Message"] = slots
}

// MessageAsString is a Message method.
//
// asString creates a string representation of an object.
func MessageAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	b := bytes.Buffer{}
	target.(*Message).stringRecurse(vm, &b)
	return vm.NewString(b.String())
}
