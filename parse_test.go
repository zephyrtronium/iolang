package iolang

import (
	"strings"
	"testing"
)

// TestParseArgs tests that messages are parsed with the correct number of
// arguments.
func TestParseArgs(t *testing.T) {
	cases := map[string]struct {
		text string
		n    int
	}{
		"None":        {"abcd", 0},
		"None()":      {"abcd()", 0},
		"NoneNone":    {"abcd    x", 0},
		"One(x)":      {"abcd(x)", 1},
		"OneSpace(x)": {"abcd    (x)", 1},
		"One(xy)":     {"abcd(x y)", 1},
		"Many": {`abcd(a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a,
				       a, a, a, a, a, a, a, a)`, 64},
		"BlankOne": {"(x)", 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			msg, err := testVM.Parse(strings.NewReader(c.text), "TestParseArgs")
			if msg == nil {
				t.Fatalf("%q parsed to nil (err: %v)", c.text, err)
			}
			if err != nil {
				t.Error(err)
			}
			if msg.ArgCount() != c.n {
				t.Errorf("%q results in wrong number of arguments; want %d, have %d", c.text, c.n, msg.ArgCount())
			}
		})
	}
}

// TestParseErrors tests that certain illegal phrasings result in errors.
func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"BareComma":         "a, b",
		"UnclosedBracket":   "abc(def",
		"UnopenedBracket":   "abc def)",
		"MismatchedBracket": "abc(def}",
		"LexerError":        "`",
		"IncorrectNumber":   "1234e",
		"EmptyFirstArg":     "abc(, e, f)",
		"EmptyMidArg":       "abc(d, , f)",
		"EmptyLastArg":      "abc(d, e, )",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := testVM.Parse(strings.NewReader(text), "TestParseErrors")
			if err == nil {
				t.Errorf("%q failed to cause an error", text)
			}
		})
	}
}
