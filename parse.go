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

// Parse converts Io source code into a message chain. The message is shuffled
// according to the VM's default operator table.
func (vm *VM) Parse(source io.Reader, label string) (msg *Message, err error) {
	var src io.RuneScanner
	if s, ok := source.(io.RuneScanner); ok {
		src = s
	} else {
		src = bufio.NewReader(source)
	}
	tokens := make(chan token)
	go lex(src, tokens)
	_, msg, err = vm.parseRecurse(-1, src, tokens, label)
	if err != nil {
		return msg, err
	}
	if err := vm.OpShuffle(vm.MessageObject(msg)); err != nil {
		return msg, err
	}
	return msg, err
}

// ParseUnshuffled converts Io source code into a message chain. No operator
// precedence parsing is applied.
func (vm *VM) ParseUnshuffled(source io.Reader, label string) (msg *Message, err error) {
	var src io.RuneScanner
	if s, ok := source.(io.RuneScanner); ok {
		src = s
	} else {
		src = bufio.NewReader(source)
	}
	tokens := make(chan token)
	go lex(src, tokens)
	_, msg, err = vm.parseRecurse(-1, src, tokens, label)
	return
}

// ParseScanner converts Io source code in a RuneScanner into a message chain.
// No operator precedence parsing is applied.
func (vm *VM) ParseScanner(src io.RuneScanner, label string) (msg *Message, err error) {
	tokens := make(chan token)
	go lex(src, tokens)
	_, msg, err = vm.parseRecurse(-1, src, tokens, label)
	return
}

func (vm *VM) parseRecurse(open rune, src io.RuneScanner, tokens chan token, label string) (tok token, msg *Message, err error) {
	msg = &Message{Label: label}
	m := msg
	defer func() {
		if msg.Text == "" && msg.Args == nil {
			// We didn't parse any messages.
			msg = nil
		} else {
			// If the final message wasn't already one, add a SemiSym to
			// conclude the expression.
			if open == -1 && !m.Prev.IsTerminator() {
				m.Text = ";"
			} else if m.Prev != nil {
				m.Prev.Next = nil
			}
		}
	}()
	for tok = range tokens {
		m.Line, m.Col = tok.Line, tok.Col
		switch tok.Kind {
		case badToken:
			err = tok.Err
			return
		case semiToken:
			if m.IsStart() {
				// empty statement
				continue
			}
			m.Text = tok.Value
		case identToken:
			m.Text = tok.Value
		case openToken:
			switch tok.Value {
			case "(":
				if m.IsStart() {
					// This is a call to the empty string slot.
					m.Text = ""
				} else {
					// These are the arguments for the previous message.
					m = m.Prev
					m.Next = nil
				}
			case "[":
				m.Text = "squareBrackets"
			case "{":
				m.Text = "curlyBrackets"
			}
			var atok token
			var amsg *Message
			atok, amsg, err = vm.parseRecurse(rune(tok.Value[0]), src, tokens, label)
			if atok.Kind == commaToken && amsg == nil {
				err = fmt.Errorf("empty argument")
			}
			for atok.Kind == commaToken {
				if err != nil {
					tok = atok
					return
				}
				m.Args = append(m.Args, amsg)
				atok, amsg, err = vm.parseRecurse(rune(tok.Value[0]), src, tokens, label)
				if amsg == nil {
					err = fmt.Errorf("empty argument")
					tok = atok
					return
				}
			}
			if atok.Kind != closeToken {
				err = fmt.Errorf("unclosed brackets")
				return
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
			if amsg != nil {
				m.Args = append(m.Args, amsg)
			}
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
			m.Text = tok.Value
			m.Memo = vm.NewNumber(f)
		case hexToken:
			var x int64
			x, err = strconv.ParseInt(tok.Value, 0, 64)
			f := float64(x)
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
			m.Text = tok.Value
			m.Memo = vm.NewNumber(f)
		case stringToken:
			var s string
			s, err = strconv.Unquote(tok.Value)
			if err != nil {
				return
			}
			m.Text = tok.Value
			m.Memo = vm.NewString(s)
		case triquoteToken:
			m.Text = tok.Value
			m.Memo = vm.NewString(tok.Value[3 : len(tok.Value)-3])
		}
		if tok.Kind != commentToken {
			m.Next = &Message{
				Prev:  m,
				Label: label,
			}
			m = m.Next
		}
	}
	return
}

// DoString parses and executes a string.
func (vm *VM) DoString(src string, label string) (Interface, Stop) {
	return vm.DoReader(strings.NewReader(src), label)
}

// DoReader parses and executes an io.Reader.
func (vm *VM) DoReader(src io.Reader, label string) (Interface, Stop) {
	msg, err := vm.Parse(src, label)
	if err != nil {
		return vm.IoError(err), ExceptionStop
	}
	return vm.DoMessage(msg, vm.Lobby)
}

// DoMessage evaluates a message.
func (vm *VM) DoMessage(msg *Message, locals Interface) (Interface, Stop) {
	return msg.Eval(vm, locals)
}

// MustDoString parses and executes a string, panicking if the result is a
// raised exception.
func (vm *VM) MustDoString(src string) Interface {
	r := strings.NewReader(src)
	msg, err := vm.Parse(r, "<string>")
	if err != nil {
		panic(fmt.Errorf("iolang: error parsing critical code: %w", err))
	}
	v, stop := msg.Eval(vm, vm.Lobby)
	if stop == ExceptionStop {
		panic(fmt.Errorf("iolang: exception executing critical code: %s", vm.AsString(v)))
	}
	return v
}
