package iolang

/*
This file is for converting lexer tokens into messages. If you're looking
for operator precedence parsing, check optable.go.
*/

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Parse converts Io source code into a message chain.
func (vm *VM) Parse(source io.Reader) (msg *Message, err error) {
	src := bufio.NewReader(source)
	tokens := make(chan token)
	go lex(src, tokens)
	_, msg, err = vm.parseRecurse(-1, src, tokens)
	return
}

func (vm *VM) parseRecurse(open rune, src *bufio.Reader, tokens chan token) (tok token, msg *Message, err error) {
	msg = &Message{Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}}}
	m := msg
	defer func() {
		if msg.Symbol.Kind == NoSym {
			// We didn't parse any messages.
			msg = nil
		} else {
			// If the final message wasn't already one, add a SemiSym to
			// conclude the expression.
			if open == -1 && m.Prev.Symbol.Kind != SemiSym {
				m.Symbol.Kind = SemiSym
			} else {
				m.Prev.Next = nil
			}
		}
	}()
	for tok = range tokens {
		switch tok.Kind {
		case badToken:
			err = tok.Err
			return
		case semiToken:
			if m.IsStart() {
				// empty statement
				continue
			}
			// TODO: if previous token is in the OperatorTable, ignore newline
			m.Symbol = Symbol{Kind: SemiSym, Text: string(tok.Value)}
		case identToken:
			// TODO: handle operator precedence
			m.Symbol = Symbol{Kind: IdentSym, Text: string(tok.Value)}
		case openToken:
			switch tok.Value {
			case "(":
				if m.IsStart() {
					// This is a call to the empty string slot.
					m.Symbol = Symbol{Kind: IdentSym}
				} else {
					// These are the arguments for the previous message.
					m = m.Prev
					m.Next = nil
				}
			case "[":
				m.Symbol = Symbol{Kind: IdentSym, Text: "squareBrackets"}
			case "{":
				m.Symbol = Symbol{Kind: IdentSym, Text: "curlyBrackets"}
			}
			var atok token
			var amsg *Message
			for atok, amsg, err = vm.parseRecurse(rune(tok.Value[0]), src, tokens); atok.Kind == commaToken; atok, amsg, err = vm.parseRecurse(rune(tok.Value[0]), src, tokens) {
				if amsg == nil {
					err = fmt.Errorf("empty argument")
				}
				if err != nil {
					tok = atok
					return
				}
				m.Args = append(m.Args, amsg)
			}
			if len(m.Args) == 1 {
				if m.Args[0] == nil {
					// There were no actual arguments.
					m.Args = nil
				}
			} else if len(m.Args) > 1 {
				if m.Args[len(m.Args)-1] == nil {
					err = fmt.Errorf("empty argument")
				}
			}
			if err != nil {
				tok = atok
				return
			}
			m.Args = append(m.Args, amsg)
		case closeToken:
			// I care about matching brackets, even though the original Io
			// implementation is quite happy to parse (2].
			switch open {
			case '(':
				if tok.Value != ")" {
					err = fmt.Errorf("expected ')', got '%s'", tok.Value)
				}
			case '[':
				if tok.Value != "]" {
					err = fmt.Errorf("expected ']', got '%s'", tok.Value)
				}
			case '{':
				if tok.Value != "}" {
					err = fmt.Errorf("expected '}', got '%s'", tok.Value)
				}
			default:
				err = fmt.Errorf("unexpected '%s'", tok.Value)
			}
			return
		case commaToken:
			if open == -1 {
				err = fmt.Errorf("bro you can't just comma like that")
			}
			return
		case numberToken:
			var f float64
			f, err = strconv.ParseFloat(tok.Value, 64)
			if err != nil {
				if err.(*strconv.NumError).Err == strconv.ErrRange {
					err = nil
				} else {
					return
				}
			}
			m.Symbol = Symbol{Kind: NumSym, Num: f}
			m.Memo = vm.NewNumber(f)
		case hexToken:
			var x int64
			var f float64
			x, err = strconv.ParseInt(tok.Value, 0, 64)
			f = float64(x)
			if err != nil {
				if err.(*strconv.NumError).Err == strconv.ErrRange {
					err = nil
					if tok.Value[0] == '-' {
						f = math.Inf(-1)
					} else {
						f = math.Inf(1)
					}
				} else {
					return
				}
			}
			m.Symbol = Symbol{Kind: NumSym, Num: f}
			m.Memo = vm.NewNumber(f)
		case stringToken:
			var s string
			s, err = strconv.Unquote(tok.Value)
			if err != nil {
				return
			}
			m.Symbol = Symbol{Kind: StringSym, String: s}
			m.Memo = vm.NewString(s)
		case triquoteToken:
			m.Symbol = Symbol{Kind: StringSym, String: tok.Value[3 : len(tok.Value)-3]}
			m.Memo = vm.NewString(tok.Value[3 : len(tok.Value)-3])
		}
		m.Next = &Message{
			Object: Object{Slots: vm.DefaultSlots["Message"], Protos: []Interface{vm.BaseObject}},
			Prev:   m,
		}
		m = m.Next
	}
	return
}

// DoString parses and executes a string.
func (vm *VM) DoString(src string) Interface {
	return vm.DoReader(strings.NewReader(src))
}

// DoReader parses and executes an io.Reader.
func (vm *VM) DoReader(src io.Reader) Interface {
	msg, err := vm.Parse(src)
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(msg); err != nil {
		return err
	}
	return vm.DoMessage(msg, vm.BaseObject)
}

// DoMessage evaluates a message.
func (vm *VM) DoMessage(msg *Message, locals Interface) Interface {
	r := msg.Eval(vm, locals)
	if stop, ok := r.(Stop); ok {
		return stop.Result
	}
	return r
}

// IsStart determines whether this message is the start of a "statement." This
// is true if it has no previous link or if the previous link is a SemiSym.
func (m *Message) IsStart() bool {
	return m.Prev.IsTerminator()
}

// IsTerminator determines whether this message is the end of an expression.
// This is true if it is nil or its symbol is a SemiSym.
func (m *Message) IsTerminator() bool {
	return m == nil || m.Symbol.Kind == SemiSym
}
