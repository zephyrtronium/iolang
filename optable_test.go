package iolang

import (
	"strings"
	"testing"
)

// TestLazyOptable tests that a new OperatorTable is created whenever
// one is needed but does not exist.
func TestLazyOptable(t *testing.T) {
	vm := TestingVM()
	cases := []string{"operators", "assignOperators"}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			t.Run("Remove", func(t *testing.T) {
				vm.Operators.RemoveSlot(c)
				vm.Parse(strings.NewReader("Lobby"), "TestLazyOptable")
				r, proto := vm.Operators.GetSlot(c)
				if proto == nil {
					t.Fatalf("OperatorTable missing %s after parsing", c)
				}
				if _, ok := r.Value.(map[string]*Object); !ok {
					t.Fatalf("OperatorTable %s has wrong type; want Map, have %v", c, vm.TypeName(r))
				}
			})
			t.Run("Change", func(t *testing.T) {
				vm.Operators.SetSlot(c, vm.Nil)
				vm.Parse(strings.NewReader("Lobby"), "TestLazyOptable")
				r, proto := vm.Operators.GetSlot(c)
				if proto == nil {
					t.Fatalf("OperatorTable missing %s after parsing", c)
				}
				if _, ok := r.Value.(map[string]*Object); !ok {
					t.Fatalf("OperatorTable %s has wrong type; want Map, have %v", c, vm.TypeName(r))
				}
			})
		})
	}
}

// Diff returns nil if m has the same text as other, both or neither have a
// memo, both have the same number of arguments, their respective arguments are
// recursively equal, and their Next messages are recursively equal. Otherwise,
// the first message belonging to other that differs from m is returned. Panics
// if other is nil.
func (m *Message) Diff(other *Message) *Message {
	if m == nil && other != nil {
		return other
	}
	if m.Text != other.Text {
		return other
	}
	if m.Memo == nil && other.Memo != nil || m.Memo != nil && other.Memo == nil {
		return other
	}
	if len(m.Args) != len(other.Args) {
		return other
	}
	for i, arg := range m.Args {
		r := arg.Diff(other.Args[i])
		if r != nil {
			return r
		}
	}
	if m.Next == nil {
		if other.Next != nil {
			return other.Next
		}
		return nil
	}
	return m.Next.Diff(other.Next)
}

// TestOptableShuffle tests that operator precedence shuffling produces the
// correct message chains using the default OperatorTable.
func TestOptableShuffle(t *testing.T) {
	vm := TestingVM()
	cases := map[string]string{
		"x+y":      "x +(y)",
		"x+y+z":    "x +(y) +(z)",
		"x+y-z":    "x +(y) -(z)",
		"x*y+z":    "x *(y) +(z)",
		"x+y*z":    "x +(y *(z))",
		"x*y+z*w":  "x *(y) +(z *(w))",
		"x**y*z+w": "x **(y) *(z) +(w)",
		"x*y**z+w": "x *(y **(z)) +(w)",

		"x := y":        `setSlot("x", y)`,
		"x = y":         `updateSlot("x", y)`,
		"x ::= y":       `newSlot("x", y)`,
		"x := y+z":      `setSlot("x", y +(z))`,
		"x := ?x":       `setSlot("x", ?(x))`,
		"x := return x": `setSlot("x", return(x))`,
		"return x := y": `return(setSlot("x", y))`,
		"x := y := z":   `setSlot("x", setSlot("y", z))`,
		"x y := z":      `x setSlot("y", z)`,

		"__noShuffling__ x+y":           "__noShuffling__ x + y",
		"__noShuffling__ x+y+z":         "__noShuffling__ x + y + z",
		"__noShuffling__ x := y":        "__noShuffling__ x := y",
		"__noShuffling__ x := y := z":   "__noShuffling__ x := y := z",
		"__noShuffling__ return x":      "__noShuffling__ return x",
		"__noShuffling__ x := return x": "__noShuffling__ x := return x",
	}
	for c, s := range cases {
		t.Run(c, func(t *testing.T) {
			a, err := vm.Parse(strings.NewReader(c), "TestOptableShuffle")
			if err != nil {
				t.Fatalf("error parsing %q: %v", c, err)
			}
			b, err := vm.ParseUnshuffled(strings.NewReader(s), "TestOptableShuffle")
			if err != nil {
				t.Fatalf("error parsing unshuffled %q: %v", s, err)
			}
			if d := b.Diff(a); d != nil {
				t.Errorf("parses of %q and unshuffled %q differ with %#v", c, s, d)
			}
		})
	}
}

// TestOptableErrors tests that invalid operator expressions produce errors when
// shuffled.
func TestOptableErrors(t *testing.T) {
	vm := TestingVM()
	cases := map[string]string{
		"AssignStart":    ":= x",
		"AssignOnly":     ":=",
		"AssignArgCount": "x := (y, z)",
		"AssignNothing":  "x :=",
		"AssignToCall":   "x(y) := z", // controversial
		"BadAssignOp":    "x <>< y",
		"BadOp":          "x $ y",
	}
	ops, _ := vm.Operators.GetSlot("operators")
	asgn, _ := vm.Operators.GetSlot("assignOperators")
	ops.Value.(map[string]*Object)["$"] = vm.Nil
	asgn.Value.(map[string]*Object)["<><"] = vm.Nil
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := vm.Parse(strings.NewReader(c), "TestOptableErrors")
			if err == nil {
				t.Errorf("%q failed to cause a parsing error", c)
			}
		})
	}
	vm.initOpTable()
}

// TODO: tests on changing the operator table
