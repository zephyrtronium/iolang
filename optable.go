package iolang

// import "fmt"

type OpTable struct {
	Object
	Operators map[string]Operator
	Assigns   map[string]Operator
}

type Operator struct {
	// The slot the operator accesses; this is always the same as the name
	// except for assign operators.
	Calls string
	// Precedence. Lower is more binding.
	Prec int
	// Associativity. Right-associativity is more binding than
	// left-associativity for operators of equal precedence.
	Right bool
}

func (op Operator) MoreBinding(than Operator) bool {
	return op.Prec < than.Prec || op.Prec == than.Prec && than.Right
}

func (vm *VM) initOpTable() {
	vm.Operators.Operators = map[string]Operator{
		"?":      Operator{"?", 0, false},
		"@":      Operator{"@", 0, false},
		"@@":     Operator{"@@", 0, false},
		"**":     Operator{"**", 1, true},
		"%":      Operator{"%", 2, false},
		"*":      Operator{"*", 2, false},
		"/":      Operator{"/", 2, false},
		"+":      Operator{"+", 3, false},
		"-":      Operator{"-", 3, false},
		"<<":     Operator{"<<", 4, false},
		">>":     Operator{">>", 4, false},
		"<":      Operator{"<", 5, false},
		"<=":     Operator{"<=", 5, false},
		">":      Operator{">", 5, false},
		">=":     Operator{">=", 5, false},
		"!=":     Operator{"!=", 6, false},
		"==":     Operator{"==", 6, false},
		"&":      Operator{"&", 7, false},
		"^":      Operator{"^", 8, false},
		"|":      Operator{"|", 9, false},
		"&&":     Operator{"&&", 10, false},
		"and":    Operator{"and", 10, false},
		"or":     Operator{"or", 11, false},
		"||":     Operator{"||", 11, false},
		"..":     Operator{"..", 12, false},
		"%=":     Operator{"%=", 13, true},
		"&=":     Operator{"&=", 13, true},
		"*=":     Operator{"*=", 13, true},
		"+=":     Operator{"+=", 13, true},
		"-=":     Operator{"-=", 13, true},
		"/=":     Operator{"/=", 13, true},
		"<<=":    Operator{"<<=", 13, true},
		">>=":    Operator{">>=", 13, true},
		"^=":     Operator{"^=", 13, true},
		"|=":     Operator{"|=", 13, true},
		"return": Operator{"return", 14, false},
	}
	vm.Operators.Assigns = map[string]Operator{
		"::=": Operator{"newSlot", 13, true},
		":=":  Operator{"setSlot", 13, true},
		"=":   Operator{"updateSlot", 13, true},
	}
}

// Perform operator-precedence reordering of a message according to the
// OpTable in the VM.
func (vm *VM) OpShuffle(m *Message) {
	vm.shuffleRecurse(m, Operator{Prec: int(^uint(0) >> 1)})
}

/*
1. if the current token is not an operator then it's unaffected
2. if its precedence is lower than the last operator then it is more binding
	and the argument is part of the current message
3. if its precedence is the same then it falls to associativity; left is higher
4. if it's higher then it is less binding and the argument is part of the last
	operator with precedence < current or == current and right-assoc
5. 1 + (1+x) * 2 = 1 +((1 +(x)) *(2)) -->
	whenever an operator has arguments and the next operator is more binding,
	add it to the end of the argument
recursively add the following messages until an operator that is less binding
	than the current is reached to the argument of the current operator
*/
func (vm *VM) shuffleRecurse(m *Message, current Operator) (last *Message) {
	if m == nil {
		return nil
	}
	for x, op := vm.nextOp(m); x != nil; x, op = vm.nextOp(m) {
		if len(x.Args) > 0 {
			// If the current operator has an argument already and the next
			// operator is more binding, then the next is part of the argument
			// of the current.
			for next, op2 := vm.nextOp(x); next != nil && op2.MoreBinding(op); next, op2 = vm.nextOp(x) {
				last = vm.shuffleRecurse(next, op2)
				x.Args[0] = &Message{
					Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}},
					Symbol: Symbol{Kind: IdentSym, Text: ""},
					Args:   []*Message{x.Args[0]},
					Next:   next,
				}
				next.Prev = x.Args[0]
				x.Next = last
				if last != nil && last.Prev != nil {
					last.Prev.Next = nil
				}
			}
		} else {
			// Recursively add to the argument of the current operator the
			// messages between the current and next operators until an
			// operator that is less binding than the current is reached.
			if op.MoreBinding(current) {
				last = vm.shuffleRecurse(x, op)
				m.Args = []*Message{m.Next}
				m.Next.Prev = nil
				m.Next = last
				if last != nil && last.Prev != nil {
					last.Prev.Next = nil
				}
				// Some extra work is done here because we already have the
				// next operator but discard it and find it again in the
				// post-loop, but I want to get this working before I care.
				// Besides, the next operator will always be the next message
				// because we just linked it thus.
			} else {
				return x
			}
		}
	}
	if m.Next != nil {
		m.Args = []*Message{m.Next}
		m.Next.Prev = nil
		m.Next = nil
	}
	return nil
}

func (vm *VM) nextOp(m *Message) (*Message, Operator) {
	if m == nil {
		return nil, Operator{}
	}
	for _, arg := range m.Args {
		vm.OpShuffle(arg)
	}
	m = m.Next
	for m != nil {
		for _, arg := range m.Args {
			vm.OpShuffle(arg)
		}
		switch m.Symbol.Kind {
		case SemiSym:
			return m, Operator{}
		case IdentSym:
			if op, ok := vm.Operators.Operators[m.Symbol.Text]; ok {
				return m, op
			} else if op, ok = vm.Operators.Assigns[m.Symbol.Text]; ok {
				return m, op
			}
		}
		m = m.Next
	}
	return nil, Operator{}
}
