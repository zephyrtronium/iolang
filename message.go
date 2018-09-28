package iolang

import (
	"bytes"
	"fmt"
)

type Message struct {
	Object
	Symbol     Symbol
	Args       []*Message
	Next, Prev *Message

	Memo Interface
}

type Symbol struct {
	Kind   SymKind
	Text   string
	Num    float64
	String string
}

type SymKind int

const (
	NoSym SymKind = iota
	SemiSym
	IdentSym
	NumSym
	StringSym
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

// StringMessage creates message carrying a string value.
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

func (m *Message) AssertArgCount(name string, n int) error {
	if len(m.Args) != n {
		return fmt.Errorf("%s must have %d arguments", name, n)
	}
	return nil
}

func (m *Message) ArgAt(n int) *Message {
	if n >= len(m.Args) || n < 0 {
		return nil
	}
	return m.Args[n]
}

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

func (m *Message) EvalArgAt(vm *VM, locals Interface, n int) Interface {
	return m.ArgAt(n).Eval(vm, locals)
}

// Evaluate a message in the context of the given VM. This is a proxy to Send
// using locals as the target.
func (m *Message) Eval(vm *VM, locals Interface) (result Interface) {
	return m.Send(vm, locals, locals)
}

// Send evaluates a message in the context of the given VM, targeting an
// object. If target is nil, it becomes the locals. A nil Message evaluates to
// vm.Nil.
func (m *Message) Send(vm *VM, locals, target Interface) (result Interface) {
	result = vm.Nil
	if target == nil {
		target = locals
	}
	for m != nil {
		if m.Memo != nil {
			result = m.Memo
		} else {
			// fmt.Println("target:", vm.AsString(target))
			switch m.Symbol.Kind {
			case IdentSym:
				// fmt.Println("ident:", m.Symbol.Text)
				if newtarget, proto := GetSlot(target, m.Symbol.Text); proto != nil {
					// We have the slot.
					// fmt.Println("we have the slot")
					switch a := newtarget.(type) {
					case Stop:
						a.Result = result
						return a
					case Actor:
						result = a.Activate(vm, target, locals, m)
					default:
						result = newtarget
					}
					// fmt.Println("target goes from", vm.AsString(target), "to", vm.AsString(newtarget))
					target = newtarget
				} else if forward, fp := GetSlot(target, "forward"); fp != nil {
					// fmt.Println("forwarding", m.Symbol.Text)
					if a, ok := forward.(Actor); ok {
						result = vm.SimpleActivate(a, target, locals, "forward")
					} else {
						return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Symbol.Text)
					}
				} else {
					// fmt.Println("couldn't find", m.Symbol.Text)
					return vm.NewExceptionf("%s does not respond to %s", vm.TypeName(target), m.Symbol.Text)
				}
			case SemiSym:
				target = locals
			case NumSym:
				// Numbers and strings should be in memo, but just in case.
				result = vm.NewNumber(m.Symbol.Num)
			case StringSym:
				result = vm.NewString(m.Symbol.String)
			default:
				panic(fmt.Sprintf("iolang: invalid Symbol: %#v", m.Symbol))
			}
		}
		// fmt.Println("evaluated")
		if result == nil {
			// No message should evaluate to something that is not an Io
			// object, so we want to convert nil to vm.Nil.
			result = vm.Nil
		}
		if m.Symbol.Kind != SemiSym {
			target = result
		}
		// fmt.Println("m goes from", m.Name(), "to", m.Next.Name())
		m = m.Next
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
		"asString": vm.NewCFunction(MessageAsString, "MessageAsString()"),
	}
	vm.DefaultSlots["Message"] = slots
}

func MessageAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	b := bytes.Buffer{}
	target.(*Message).stringRecurse(vm, &b)
	return vm.NewString(b.String())
}
