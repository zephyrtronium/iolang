package iolang

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
func (vm *VM) OpShuffle(m *Message) *Message {
	m.shuffleRecurse(vm, Operator{Prec: int(^uint(0) >> 1)})
	return m
}

/* ok so if the current token is not an operator then it's unaffected
if its precedence is lower than the last operator then it is more binding
	and the argument is part of the current message
if its precedence is the same then it falls to associativity; left is higher
if it's higher then it is less binding and the argument is part of the last operator
	with precedence < current or == current and right-assoc */
func (m *Message) shuffleRecurse(vm *VM, prev Operator) (end *Message) {
	x := m
	for x != nil {
		switch x.Symbol.Kind {
		case SemiSym:
			return x
		case IdentSym:
			op, ok := vm.Operators.Operators[x.Symbol.Text]
			if !ok {
				op, ok = vm.Operators.Assigns[x.Symbol.Text]
			}
			if ok {
				if op.Prec < prev.Prec || op.Prec == prev.Prec && prev.Right {
					// The operator we've found is more binding than the
					// previous, so it remains part of the argument.
					t := x.Next.shuffleRecurse(vm, op)
					if t != nil {
						if x.Prev != nil {
							x.Prev.Next = t.Next
						}
						x = t.Next
						t.Next = nil
					} else {
						if x.Prev != nil {
							x.Prev.Next = nil
						}
						return nil
					}
				} else {
					// The operator is less binding, so we return to the
					// previous level for it to decide.
					return x
				}
				break
			}
			fallthrough
		case NumSym, StringSym:
			if x == m {
				// This is the first message following an operator, so it is
				// the start of its argument.
				if x.Prev != nil {
					x.Prev.Args = append(x.Prev.Args, x)
					x.Prev = nil
				}
			}
		}
		x = x.Next
	}
	return x
}
