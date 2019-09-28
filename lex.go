package iolang

import (
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

	Line, Col int
}

type tokenKind int

const (
	badToken tokenKind = iota

	semiToken     // semicolon and newline
	identToken    // identifier
	openToken     // open bracket: (, [, {
	closeToken    // close bracket: ), ], }
	commaToken    // comma
	numberToken   // number
	hexToken      // hexadecimal number
	stringToken   // "string"
	triquoteToken // """string"""
	commentToken  // //, #, or /* */
)

const (
	lexerSpace = " \r\f\t\v"
	lexerOp    = "!$%&'*+-/:<=>?@\\^|~"
)

// lexFn is a lexer state function. Each lexFn lexes a token, sends it on the
// supplied channel, and returns the next lexFn to use.
type lexFn func(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int)

// lex converts a source into a stream of tokens.
func lex(src io.RuneScanner, tokens chan<- token) {
	state := eatSpace
	line, col := 1, 1
	for state != nil {
		state, line, col = state(src, tokens, line, col)
	}
	close(tokens)
}

// accept appends the next run of characters in src which satisfy the predicate
// to b. Returns b after appending, the first rune which did not satisfy the
// predicate, and any error that occurred. If there was no such error, the
// last rune is unread.
func accept(src io.RuneScanner, predicate func(rune) bool, b []byte) ([]byte, rune, error) {
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
func eatSpace(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	eaten, r, err := accept(src, func(r rune) bool { return strings.ContainsRune(lexerSpace, r) }, nil)
	col += len(eaten)
	if err != nil {
		if err != io.EOF {
			tokens <- token{
				Kind:  badToken,
				Value: string(r),
				Err:   err,
			}
		}
		return nil, line, col
	}
	switch {
	case r == ';', r == '\n':
		src.ReadRune()
		tokens <- token{
			Kind:  semiToken,
			Value: string(r),
			Line:  line,
			Col:   col,
		}
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		return eatSpace, line, col
	case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', r == '_', r >= 0x80:
		return lexIdent, line, col
	case strings.ContainsRune("!$%&'*+-/:<=>?@\\^|~", r):
		return lexOp, line, col
	case strings.ContainsRune("([{", r):
		src.ReadRune()
		tokens <- token{
			Kind:  openToken,
			Value: string(r),
			Line:  line,
			Col:   col,
		}
		col++
		return eatSpace, line, col
	case strings.ContainsRune(")]}", r):
		src.ReadRune()
		tokens <- token{
			Kind:  closeToken,
			Value: string(r),
			Line:  line,
			Col:   col,
		}
		col++
		return eatSpace, line, col
	case r == ',':
		src.ReadRune()
		tokens <- token{
			Kind:  commaToken,
			Value: ",",
			Line:  line,
			Col:   col,
		}
		col++
		return eatSpace, line, col
	case '0' <= r && r <= '9':
		return lexNumber, line, col
	case r == '.':
		return lexDot, line, col
	case r == '"':
		return lexString, line, col
	case r == '#':
		return lexHashComment, line, col
	}
	tokens <- token{
		Kind:  badToken,
		Value: string(r),
		Err:   fmt.Errorf("lexer encountered invalid character %q", r),
		Line:  line,
		Col:   col,
	}
	return nil, line, col
}

// lexIdent lexes an identifier, which consists of a-z, A-Z, 0-9, _, ., and all
// runes greater than 0x80.
func lexIdent(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	b, _, err := accept(src, func(r rune) bool {
		return 'a' <= r && r <= 'z' ||
			'A' <= r && r <= 'Z' ||
			'0' <= r && r <= '9' ||
			r == '_' || r == '.' || r >= 0x80
	}, nil)
	ncol := col + len(b)
	return lexsend(err, tokens, token{Kind: identToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexOp lexes an operator, which consists of !$%&'*+-/:<=>?@\^|~
func lexOp(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	var pr1, pr2 rune
	// This predicate tracks whether a comment starts.
	pred := func(r rune) bool {
		if pr1 == '/' && (pr2 == '/' || pr2 == '*') {
			return false
		}
		pr1, pr2 = pr2, r
		return strings.ContainsRune("!$%&'*+-/:<=>?@\\^|~", r)
	}
	b, _, err := accept(src, pred, nil)
	switch string(b) {
	case "//":
		return lexSlashSlashComment, line, col
	case "/*":
		return lexSlashStarComment, line, col
	}
	ncol := col + len(b)
	return lexsend(err, tokens, token{Kind: identToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexSlashSlashComment lexes a // comment.
func lexSlashSlashComment(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	b, _, err := accept(src, func(r rune) bool { return r != '\n' }, []byte("//"))
	ncol := col + len(b)
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexHashComment lexes a # comment.
func lexHashComment(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	b, _, err := accept(src, func(r rune) bool { return r != '\n' }, nil)
	ncol := col + len(b)
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexSlashStarComment lexes a /* */ comment.
func lexSlashStarComment(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	depth := 1
	nline := line
	ncol := col
	var pr1, pr2 rune
	// This predicate tracks /* */ comments recursively.
	pred := func(r rune) bool {
		if pr1 == '*' && pr2 == '/' {
			depth--
			if depth <= 0 {
				return false
			}
		} else if pr1 == '/' && pr2 == '*' {
			depth++
		} else if r == '\n' {
			nline++
			ncol = 0
		}
		pr1, pr2 = pr2, r
		ncol++
		return true
	}
	b, _, err := accept(src, pred, []byte("/*"))
	if !bytes.HasSuffix(b, []byte("*/")) {
		err = io.ErrUnexpectedEOF
	}
	return lexsend(err, tokens, token{Kind: commentToken, Value: string(b), Line: line, Col: col}), nline, ncol
}

// lexNumber lexes a number.
func lexNumber(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	b, r, err := accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, nil)
	ncol := col + len(b)
	if err != nil {
		return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
	}
	prelen := len(b)
	if r == 'x' || r == 'X' {
		if len(b) != 1 || b[0] != '0' {
			tokens <- token{Kind: badToken, Value: string(b), Err: fmt.Errorf("invalid numeric literal %s%c", b, r), Line: line, Col: col}
			return eatSpace, line, ncol
		}
		b = append(b, byte(r))
		_, _, err = src.ReadRune()
		if err != nil {
			return lexsend(err, tokens, token{Kind: badToken, Err: err, Line: line, Col: col}), line, ncol
		}
		b, r, err = accept(src, func(r rune) bool {
			return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
		}, b)
		ncol += len(b) - prelen
		return lexsend(err, tokens, token{Kind: hexToken, Value: string(b), Line: line, Col: col}), line, ncol
	}
	if r == '.' {
		b = append(b, '.')
		_, _, err = src.ReadRune()
		if err != nil {
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
		}
		b, r, err = accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, b)
		ncol += len(b) - prelen
		if err != nil {
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
		}
		prelen = len(b)
	}
	if r == 'e' || r == 'E' {
		src.ReadRune() // read previously unread 'e'
		b = append(b, byte(r))
		r, _, err = src.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
		}
		if r == '-' || r == '+' {
			b = append(b, byte(r))
		} else {
			src.UnreadRune()
		}
		b, _, err = accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, b)
		ncol += len(b) - prelen
	}
	return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexDot lexes a token starting with a '.' rune, which may be a number or an
// identifier.
func lexDot(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	src.ReadRune() // Re-read the '.'.
	b, r, err := accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, []byte{'.'})
	if len(b) == 1 {
		// No digits follow the '.', so this is an identifier.
		b, _, err := accept(src, func(r rune) bool {
			return 'a' <= r && r <= 'z' ||
				'A' <= r && r <= 'Z' ||
				'0' <= r && r <= '9' ||
				r == '_' || r == '.' || r >= 0x80
		}, b)
		ncol := col + len(b)
		return lexsend(err, tokens, token{Kind: identToken, Value: string(b), Line: line, Col: col}), line, ncol
	}
	ncol := col + len(b)
	if err != nil {
		return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
	}
	if r == 'e' || r == 'E' {
		prelen := len(b)
		src.ReadRune() // Read the previously unread 'e'.
		b = append(b, byte(r))
		r, _, err = src.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
		}
		if r == '-' || r == '+' {
			b = append(b, byte(r))
		} else {
			src.UnreadRune()
		}
		b, _, err = accept(src, func(r rune) bool { return '0' <= r && r <= '9' }, b)
		ncol += len(b) - prelen
	}
	return lexsend(err, tokens, token{Kind: numberToken, Value: string(b), Line: line, Col: col}), line, ncol
}

// lexString lexes a string, which may be monoquote or triquote.
func lexString(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	src.ReadRune() // Re-read the previously unread '"'.
	b := []byte{'"'}
	ncol := col + 1
	ps := false
	// We try to lex a monoquote string at first, but if we start off with """,
	// then we switch to triquote.
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
				Line:  line,
				Col:   col,
			}
			return nil, line, ncol
		}
		ncol++
		b = append(b, string(r)...)
		if r == '\\' {
			ps = !ps
		} else if r == '"' && !ps {
			if len(b) == 2 {
				// The token so far is `""`. Check for a triquote string.
				r, _, err = src.ReadRune()
				if err != nil {
					return lexsend(err, tokens, token{Kind: stringToken, Value: string(b), Line: line, Col: col}), line, ncol
				}
				if r == '"' {
					return lexTriquote, line, col
				}
				src.UnreadRune()
			}
			return lexsend(err, tokens, token{Kind: stringToken, Value: string(b), Line: line, Col: col}), line, ncol
		} else {
			ps = false
		}
	}
}

// lexTriquote lexes a triquote string.
func lexTriquote(src io.RuneScanner, tokens chan<- token, line, col int) (lexFn, int, int) {
	b := []byte{'"', '"', '"'}
	nline := line
	ncol := col + 3
	q := 0
	for {
		r, _, err := src.ReadRune()
		ncol++
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			tokens <- token{
				Kind:  badToken,
				Value: string(b),
				Err:   err,
				Line:  line,
				Col:   col,
			}
			return nil, line, ncol
		}
		b = append(b, string(r)...)
		if r == '"' {
			if q == 2 {
				return lexsend(err, tokens, token{Kind: triquoteToken, Value: string(b), Line: line, Col: col}), nline, ncol
			}
			q++
		} else {
			if r == '\n' {
				nline++
				ncol = 1
			}
			q = 0
		}
	}
}
