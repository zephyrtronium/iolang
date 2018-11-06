package iolang

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// A token is a single lexical element.
type token struct {
	Kind  tokenKind
	Value string
	Err   error

	// Line, Col int
}

type tokenKind int

const (
	badToken      tokenKind = iota
	semiToken               // semicolon and newline
	identToken              // identifier
	openToken               // open bracket: (, [, {
	closeToken              // close bracket: ), ], }
	commaToken              // comma
	numberToken             // number
	hexToken                // hexadecimal number
	stringToken             // "string"
	triquoteToken           // """string"""
	commentToken            // //, #, or /* */
)

// lexFn is a lexer state function. Each lexFn lexes a token, sends it on the
// supplied channel, and returns the next lexFn to use.
type lexFn func(src *bufio.Reader, tokens chan<- token) lexFn

// lex converts a source into a stream of tokens.
func lex(src *bufio.Reader, tokens chan<- token) {
	state := eatSpace
	for state != nil {
		state = state(src, tokens)
	}
	close(tokens)
}

// accept appends the next run of characters in src which satisfy the predicate
// to b. Returns b after appending, the first rune which did not satisfy the
// predicate, and any error that occurred. If there was no such error, the
// last rune is unread.
func accept(src *bufio.Reader, predicate func(rune) bool, b []byte) ([]byte, rune, error) {
	r, _, err := src.ReadRune()
	for {
		if err != nil {
			return b, r, err
		}
		if !predicate(r) {
			break
		}
		b = append(b, string(r)...)
		r, _, err = src.ReadRune()
	}
	src.UnreadRune()
	return b, r, nil
}

// lexsend is a shortcut for sending a token with error checking. It returns
// eatSpace as the default lexing function.
func lexsend(err error, tokens chan<- token, good token) lexFn {
	if err != nil && err != io.EOF {
		good.Kind = badToken
		good.Err = err
	}
	tokens <- good
	if err != nil {
		return nil
	}
	return eatSpace
}

// eatSpace consumes space and decides the next lexFn to use.
func eatSpace(src *bufio.Reader, tokens chan<- token) lexFn {
	_, r, err := accept(src, func(r rune) bool { return strings.ContainsRune(" \r\f\t\v", r) }, nil)
	if err != nil {
		if err != io.EOF {
			tokens <- token{
				Kind:  badToken,
				Value: string(r),
				Err:   err,
			}
		}
		return nil
	}
	switch {
	case r == ';', r == '\n':
		src.ReadRune()
		tokens <- token{
			Kind:  semiToken,
			Value: string(r),
		}
		return eatSpace
	case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', r == '_', r >= 0x80:
		return lexIdent
	case strings.ContainsRune("!$%&'*+-/:<=>?@\\^|~", r):
		return lexOp
	case strings.ContainsRune("([{", r):
		src.ReadRune()
		tokens <- token{
			Kind:  openToken,
			Value: string(r),
		}
		return eatSpace
	case strings.ContainsRune(")]}", r):
		src.ReadRune()
		tokens <- token{
			Kind:  closeToken,
			Value: string(r),
		}
		return eatSpace
	case r == ',':
		src.ReadRune()
		tokens <- token{
			Kind:  commaToken,
			Value: ",",
		}
		return eatSpace
	case '0' <= r && r <= '9':
		return lexNumber
	case r == '.':
		// . can be either a number or an identifier, because Dumbledore.
		peek, _ := src.Peek(2)
		if len(peek) > 1 && '0' <= peek[1] && peek[1] <= '9' {
			return lexNumber
		}
		return lexIdent
	case r == '"':
		return lexString
	case r == '#':
		return lexHashComment
	}
	tokens <- token{
		Kind:  badToken,
		Value: string(r),
		Err:   fmt.Errorf("lexer encountered invalid character %q", r),
	}
	return nil
}

// lexIdent lexes an identifier, which consists of a-z, A-Z, 0-9, _, ., and all
// runes greater than 0x80.
func lexIdent(src *bufio.Reader, tokens chan<- token) lexFn {
	b, _, err := accept(src, func(r rune) bool {
		return 'a' <= r && r <= 'z' ||
			'A' <= r && r <= 'Z' ||
			'0' <= r && r <= '9' ||
			r == '_' || r == '.' || r >= 0x80
	}, nil)
	return lexsend(err, tokens, token{Kind: identToken, Value: string(b)})
}

// lexOp lexes an operator, which consists of !$%&'*+-/:<=>?@\^|~
func lexOp(src *bufio.Reader, tokens chan<- token) lexFn {
	b, _, err := accept(src, func(r rune) bool {
		return strings.ContainsRune("!$%&'*+-/:<=>?@\\^|~", r)
	}, nil)
	switch string(b) {
	case "//":
		return lexSlashSlashComment
	case "/*":
		return lexSlashStarComment
	}
	return lexsend(err, tokens, token{Kind: identToken, Value: string(b)})
}

// lexSlashSlashComment lexes a // comment.
func lexSlashSlashComment(src *bufio.Reader, tokens chan<- token) lexFn {
	b, _, err := accept(src, func(r rune) bool { return r != '\n' }, []byte("//"))
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b)})
}

// lexHashComment lexes a # comment.
func lexHashComment(src *bufio.Reader, tokens chan<- token) lexFn {
	b, _, err := accept(src, func(r rune) bool { return r != '\n' }, nil)
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b)})
}

// lexSlashStarComment lexes a /* */ comment.
func lexSlashStarComment(src *bufio.Reader, tokens chan<- token) lexFn {
	var pr rune
	depth := 1
	pred := func(r rune) bool {
		if pr == '*' && r == '/' {
			depth--
			if depth <= 0 {
				return false
			}
		} else if pr == '/' && r == '*' {
			depth++
		}
		pr = r
		return true
	}
	b, _, err := accept(src, pred, []byte("/*"))
	src.ReadRune() // Re-read the / that accept unreads.
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b)})
}

// lexNumber lexes a number.
func lexNumber(src *bufio.Reader, tokens chan<- token) lexFn {
	b, r, err := accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, nil)
	if err != nil {
		return lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
	}
	if r == 'x' || r == 'X' {
		b = append(b, 'x')
		b, _, err = accept(src, func(r rune) bool {
			return '0' <= r && r <= '9' || 'a' <= r && r <= 'f' || 'A' <= r && r <= 'F'
		}, b)
		lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
	}
	if r == '.' {
		b = append(b, '.')
		_, _, err = src.ReadRune()
		if err != nil {
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
		}
		b, r, err = accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, b)
		if err != nil {
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
		}
	}
	if r == 'e' || r == 'E' {
		r, _, err = src.ReadRune()
		if err != nil {
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
		}
		if r == '-' || r == '+' {
			r, _, err = src.ReadRune()
			if err != nil {
				return lexsend(err, tokens, token{Kind: badToken, Err: err})
			}
			b = append(b, 'e', byte(r))
		} else {
			b = append(b, 'e')
		}
		b, _, err = accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, b)
	}
	return lexsend(err, tokens, token{Kind: numberToken, Value: string(b)})
}

// lexString lexes a string, which may be monoquote or triquote.
func lexString(src *bufio.Reader, tokens chan<- token) lexFn {
	peek, _ := src.Peek(3)
	if bytes.Equal(peek, []byte{'"', '"', '"'}) {
		return lexTriquote(src, tokens)
	}
	return lexMonoquote(src, tokens)
}

// lexTriquote lexes a triquote string.
func lexTriquote(src *bufio.Reader, tokens chan<- token) lexFn {
	b := make([]byte, 3, 6)
	src.Read(b)
	for {
		r, _, err := src.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			tokens <- token{
				Kind:  badToken,
				Value: string(b),
				Err:   err,
			}
			return nil
		}
		if r == '"' {
			peek, err := src.Peek(2)
			if bytes.Equal(peek, []byte{'"', '"'}) {
				return lexsend(err, tokens, token{Kind: triquoteToken, Value: string(b) + `"""`})
			}
		}
		b = append(b, string(r)...)
	}
}

// lexMonoquote lexes a monoquote string.
func lexMonoquote(src *bufio.Reader, tokens chan<- token) lexFn {
	b := make([]byte, 1, 2)
	src.Read(b)
	for {
		r, _, err := src.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			tokens <- token{
				Kind:  badToken,
				Value: string(b),
				Err:   err,
			}
			return nil
		}
		b = append(b, string(r)...)
		if r == '\\' {
			continue
		}
		if r == '"' {
			return lexsend(err, tokens, token{Kind: stringToken, Value: string(b)})
		}
	}
}
