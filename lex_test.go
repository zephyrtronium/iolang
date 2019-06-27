package iolang

import (
	"testing"
	"bufio"
	"strings"
)

// String returns the name of a token kind.
func (t tokenKind) String() string {
	switch t {
	case badToken: return "badToken"
	case semiToken: return "semiToken"
	case identToken: return "identToken"
	case openToken: return "openToken"
	case closeToken: return "closeToken"
	case commaToken: return "commaToken"
	case numberToken: return "numberToken"
	case hexToken: return "hexToken"
	case stringToken: return "stringToken"
	case triquoteToken: return "triquoteToken"
	case commentToken: return "commentToken"
	}
	panic("invalid tokenKind")
}

// TestLexSingles tests that individual tokens have the correct kinds and
// values.
func TestLexSingles(t *testing.T) {
	cases := map[string]struct {
		text string
		kind tokenKind
		val  string
	}{
		"Semi-;": {";", semiToken, ";"},
		"Semi-\\n": {"\n", semiToken, "\n"},
		"Ident-alpha": {"abcd", identToken, "abcd"},
		"Ident-alnum": {"a123", identToken, "a123"},
		"Ident-sym": {"_._", identToken, "_._"},
		"Ident-dot": {".", identToken, "."},
		"Ident-op!": {"!", identToken, "!"},
		"Ident-op!!": {"!!", identToken, "!!"},
		"Ident-op$$": {"$$", identToken, "$$"},
		"Ident-op%%": {"%%", identToken, "%%"},
		"Ident-op&&": {"&&", identToken, "&&"},
		"Ident-op''": {"''", identToken, "''"},
		"Ident-op**": {"**", identToken, "**"},
		"Ident-op++": {"++", identToken, "++"},
		"Ident-op--": {"--", identToken, "--"},
		"Ident-op/!": {"/!", identToken, "/!"},
		"Ident-op::": {"::", identToken, "::"},
		"Ident-op<<": {"<<", identToken, "<<"},
		"Ident-op==": {"==", identToken, "=="},
		"Ident-op>>": {">>", identToken, ">>"},
		"Ident-op??": {"??", identToken, "??"},
		"Ident-op@@": {"@@", identToken, "@@"},
		`Ident-op\\`: {`\\`, identToken, `\\`},
		"Ident-op^^": {"^^", identToken, "^^"},
		"Ident-op||": {"||", identToken, "||"},
		"Ident-op~~": {"~~", identToken, "~~"},
		"Open-(": {"(", openToken, "("},
		"Open-[": {"[", openToken, "["},
		"Open-{": {"{", openToken, "{"},
		"Close-)": {")", closeToken, ")"},
		"Close-]": {"]", closeToken, "]"},
		"Close-}": {"}", closeToken, "}"},
		"Comma": {",", commaToken, ","},
		"Number-num": {"1234", numberToken, "1234"},
		"Number-num.": {"1234.", numberToken, "1234."},
		"Number-num.num": {"1234.567", numberToken, "1234.567"},
		"Number-.num": {".567", numberToken, ".567"},
		"Number-numE": {"1234e9", numberToken, "1234e9"},
		"Number-num.E": {"1234.e9", numberToken, "1234.e9"},
		"Number-num.numE": {"1234.567e9", numberToken, "1234.567e9"},
		"Number-numEp": {"1234e+9", numberToken, "1234e+9"},
		"Number-numEm": {"1234e-9", numberToken, "1234e-9"},
		"Hex-0xdigits": {"0x1234", hexToken, "0x1234"},
		"Hex-0xabcdef": {"0xabcdef", hexToken, "0xabcdef"},
		"Monoquote-plain": {`"abcd"`, stringToken, `"abcd"`},
		"Monoquote-backslash": {`"a\bcd"`, stringToken, `"a\bcd"`},
		"Monoquote-backslash-quote": {`"a\"bcd"`, stringToken, `"a\"bcd"`},
		"Triquote-plain": {`"""abcd"""`, triquoteToken, `"""abcd"""`},
		"Triquote-backslash": {`"""a\bcd"""`, triquoteToken, `"""a\bcd"""`},
		"Triquote-backslash-quote": {`"""a\"bcd"""`, triquoteToken, `"""a\"bcd"""`},
		"Triquote-backslash-end": {`"""abcd\"""`, triquoteToken, `"""abcd\"""`},
		"Triquote-newline": {"\"\"\"\n\"\"\"", triquoteToken, "\"\"\"\n\"\"\""},
		"Comment-//": {"//", commentToken, "//"},
		"Comment-//comment": {"// comment goes here", commentToken, "// comment goes here"},
		"Comment-/////": {"/////", commentToken, "/////"},
		"Comment-/////comment": {"///// comment goes here", commentToken, "///// comment goes here"},
		"Comment-/**/": {"/**/", commentToken, "/**/"},
		"Comment-/*comment*/": {"/* comment */", commentToken, "/* comment */"},
		"Comment-/**comment**/": {"/** comment **/", commentToken, "/** comment **/"},
		"Comment-/*recursive*/": {"/*/* comment */*/", commentToken, "/*/* comment */*/"},
		"Comment-/*newline*/": {"/* \n */", commentToken, "/* \n */"},
		"Comment-#": {"#", commentToken, "#"},
		"Comment-#comment": {"# comment goes here", commentToken, "# comment goes here"},
		"Error-`": {"`", badToken, "`"},
		"Error-unclosed-string": {`"abcd`, badToken, `"abcd`},
		"Error-unclosed-triquote": {`"""abcd""`, badToken, `"""abcd""`},
		"Space": {"   abcd   ", identToken, "abcd"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			ch := make(chan token, 100) // large buffer so failures complete
			lex(bufio.NewReader(strings.NewReader(c.text)), ch)
			tok, ok := <-ch
			if !ok {
				t.Fatal("no token lexed")
			}
			if tok.Kind != c.kind {
				t.Errorf("%q lexed as wrong kind: wanted %v, got %v", c.text, c.kind, tok.Kind)
			}
			if tok.Value != c.val {
				t.Errorf("%q lexed with wrong text: wanted %q, got %q", c.text, c.val, tok.Value)
			}
			tok, ok = <-ch
			if ok {
				t.Errorf("lexed extra token %v", tok)
			}
		})
	}
}