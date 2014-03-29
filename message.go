package iolang

import "fmt"

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

func (m *Message) AssertArgCount(name string, n int) error {
	if len(m.Args) != n {
		return fmt.Errorf("%s must have %d arguments", name, n)
	}
	return nil
}

func (m *Message) ArgAt(n int) *Message {
	if n >= len(m.Args) {
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

// Evaluate a message in the context of the given VM. A nil message evaluates
// to vm.Nil.
func (m *Message) Eval(vm *VM, locals Interface) (result Interface) {
	result = vm.Nil
	target := locals
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
					switch a := newtarget.(type) {
					case Stop:
						a.Result = result
						return a
					case Actor:
						// TODO: provide a Call
						result = a.Activate(vm, target, locals, m)
					default:
						result = newtarget
					}
					target = newtarget
				} else if forward, fp := GetSlot(target, "forward"); fp != nil {
					// fmt.Println("forwarding", m.Symbol.Text)
					if a, ok := forward.(Actor); ok {
						// TODO: provide a Call
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
		if result == nil {
			// No message should evaluate to something that is not an Io
			// object, so we want to convert nil to vm.Nil.
			result = vm.Nil
		}
		if m.Symbol.Kind != SemiSym {
			target = result
		}
		m = m.Next
	}
	return result
}

func (m *Message) Name() string {
	switch m.Symbol.Kind {
	case SemiSym, IdentSym:
		return m.Symbol.Text
	case NumSym:
		return fmt.Sprintf("%d", m.Symbol.Num)
	case StringSym:
		return m.Symbol.String
	default:
		panic(fmt.Sprintf("iolang: invalid Symbol: %#v", m.Symbol))
	}
}

func (m *Message) String() string {
	return "message"
}
