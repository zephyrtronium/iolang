// Package testutils provides utilities for testing Io code in Go.
package testutils

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/zephyrtronium/iolang"
)

// testVM is the VM used for all tests.
var testVM *iolang.VM

var testVMInit sync.Once

// VM returns a VM for testing Io. The VM is shared by all tests that
// use this package.
func VM() *iolang.VM {
	testVMInit.Do(ResetVM)
	return testVM
}

// ResetVM reinitializes the VM returned by TestVM. It is not safe to
// call this in parallel tests.
func ResetVM() {
	testVM = iolang.NewVM()
}

// A SourceTestCase is a test case containing Io source code and a predicate to
// check the result.
type SourceTestCase struct {
	// Source is the Io source code to execute.
	Source string
	// Pass is a predicate taking the result of executing Source. If Pass
	// returns false, then the test fails.
	Pass func(result *iolang.Object, control iolang.Stop) bool
}

// TestFunc returns a test function for the test case. This uses TestingVM to
// parse and execute the code.
func (c SourceTestCase) TestFunc(name string) func(*testing.T) {
	return func(t *testing.T) {
		vm := VM()
		msg, err := vm.ParseScanner(strings.NewReader(c.Source), name)
		if err != nil {
			t.Fatalf("could not parse %q: %v", c.Source, err)
		}
		if err := vm.OpShuffle(vm.MessageObject(msg)); err != nil {
			t.Fatalf("could not opshuffle %q: %v", c.Source, err)
		}
		if r, s := vm.DoMessage(msg, vm.Lobby); !c.Pass(r, s) {
			if s == iolang.ExceptionStop && r.Tag() == iolang.ExceptionTag {
				w := strings.Builder{}
				ex := r.Value.(iolang.Exception)
				fmt.Fprintf(&w, "%q produced wrong result; an exception occurred:\n", c.Source)
				for i := len(ex.Stack) - 1; i >= 0; i-- {
					m := ex.Stack[i]
					if m.IsStart() {
						fmt.Fprintf(&w, "\t%s\t%s:%d\n", m.Name(), m.Label, m.Line)
					} else {
						fmt.Fprintf(&w, "\t%s %s\t%s:%d\n", m.Prev.Name(), m.Name(), m.Label, m.Line)
					}
				}
				fmt.Fprint(&w, vm.AsString(r))
				t.Error(w.String())
			} else {
				t.Errorf("%q produced wrong result; got %s@%p (%s)", c.Source, vm.AsString(r), r, s)
			}
		}
	}
}

// PassEqual returns a Pass function for a SourceTestCase that predicates on
// equality. To determine equality, this first checks for equal identities; if
// not, it checks that the result of TestingVM().Compare(want, result) is 0. If
// the Stop is not NoStop, then the predicate returns false.
func PassEqual(want *iolang.Object) func(*iolang.Object, iolang.Stop) bool {
	return PassControl(want, iolang.NoStop)
}

// PassUnequal returns a Pass function for a SourceTestCase that predicates on
// non-equality by checking that the result of testVM.Compare(want, result) is
// not 0.
func PassUnequal(want *iolang.Object) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		vm := VM()
		if control != iolang.NoStop {
			return false
		}
		if want == result {
			return false
		}
		v, obj, stop := vm.Compare(want, result)
		if stop != iolang.NoStop {
			return false
		}
		if obj != nil {
			return false
		}
		return v != 0
	}
}

// PassIdentical returns a Pass function for a SourceTestCase that predicates
// on identity equality, i.e. the result must be exactly the given object. If
// the Stop is not NoStop, then the predicate returns false.
func PassIdentical(want *iolang.Object) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		if control != iolang.NoStop {
			return false
		}
		return want == result
	}
}

// PassControl returns a Pass function for a SourceTestCase that predicates on
// equality with a certain control flow status. The control flow check precedes
// the value check. Equality here has the same semantics as in PassEqual.
func PassControl(want *iolang.Object, stop iolang.Stop) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		vm := VM()
		if control != stop {
			return false
		}
		if want == result {
			return true
		}
		v, obj, stop := vm.Compare(want, result)
		if stop != iolang.NoStop {
			return false
		}
		if obj != nil {
			return false
		}
		return v == 0
	}
}

// PassTag returns a Pass function for a SourceTestCase that predicates on
// equality of the Tag of the result. If the Stop is not NoStop, then the
// predicate returns false.
func PassTag(want iolang.Tag) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		if control != iolang.NoStop {
			return false
		}
		return result.Tag() == want
	}
}

// PassFailure returns a Pass function for a SourceTestCase that returns true
// iff the result is a raised exception.
func PassFailure() func(*iolang.Object, iolang.Stop) bool {
	// This doesn't need to be a function returning a function, but it's nice to
	// stay consistent with the other predicate generators.
	return func(result *iolang.Object, control iolang.Stop) bool {
		return control == iolang.ExceptionStop
	}
}

// PassSuccess returns a Pass function for a SourceTestCase that returns true
// iff the control flow status is NoStop.
func PassSuccess() func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		return control == iolang.NoStop
	}
}

// PassLocalSlots returns a Pass function for a SourceTestCase that returns
// true iff the result locally has all of the slots in want and none of the
// slots in exclude. If the Stop is not NoStop, then the predicate returns
// false.
func PassLocalSlots(want, exclude []string) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		vm := VM()
		if control != iolang.NoStop {
			return false
		}
		for _, slot := range want {
			if _, ok := vm.GetLocalSlot(result, slot); !ok {
				return false
			}
		}
		for _, slot := range exclude {
			if _, ok := vm.GetLocalSlot(result, slot); ok {
				return false
			}
		}
		return true
	}
}

// PassEqualSlots returns a Pass function for a SourceTestCase that returns
// true iff the result has exactly the same slots as want and the slots' values
// compare equal. If the Stop is not NoStop, then the predicate returns false.
func PassEqualSlots(want iolang.Slots) func(*iolang.Object, iolang.Stop) bool {
	return func(result *iolang.Object, control iolang.Stop) bool {
		vm := VM()
		if control != iolang.NoStop {
			return false
		}
		slots := vm.GetAllSlots(result)
		for slot := range slots {
			if _, ok := want[slot]; !ok {
				return false
			}
		}
		for slot, value := range want {
			x, ok := slots[slot]
			if !ok {
				return false
			}
			v, obj, stop := vm.Compare(x, value)
			if stop != iolang.NoStop {
				return false
			}
			if obj != nil {
				return false
			}
			return v == 0
		}
		return true
	}
}

// CheckSlots is a testing helper to check whether an object has exactly the
// listed slots.
func CheckSlots(t *testing.T, obj *iolang.Object, slots []string) {
	t.Helper()
	checked := make(map[string]bool, len(slots))
	on := VM().GetAllSlots(obj)
	for _, name := range slots {
		checked[name] = true
		t.Run("Have_"+name, func(t *testing.T) {
			slot, ok := on[name]
			if !ok {
				t.Error("no slot", name)
			}
			if slot == nil {
				t.Error("slot", name, "is nil")
			}
		})
	}
	for name := range on {
		t.Run("Want_"+name, func(t *testing.T) {
			if !checked[name] {
				t.Error("unexpected slot", name)
			}
		})
	}
}

// CheckNewSlots is a testing helper to check whether an object has the given
// slots, and possibly others.
func CheckNewSlots(t *testing.T, obj *iolang.Object, slots []string) {
	t.Helper()
	on := VM().GetAllSlots(obj)
	for _, name := range slots {
		t.Run("Have_"+name, func(t *testing.T) {
			slot, ok := on[name]
			if !ok {
				t.Error("no slot", name)
			}
			if slot == nil {
				t.Error("slot", name, "is nil")
			}
		})
	}
}

// CheckObjectIsProto is a testing helper to check that an object has exactly
// one proto, which is Core Object. obj must come from the test VM.
func CheckObjectIsProto(t *testing.T, obj *iolang.Object) {
	t.Helper()
	protos := obj.Protos()
	switch len(protos) {
	case 0:
		t.Error("no protos")
		return
	case 1: // do nothing
	default:
		t.Error("incorrect number of protos: expected 1, have", len(protos))
	}
	vm := VM()
	if p := protos[0]; p != vm.BaseObject {
		t.Errorf("wrong proto: expected %T@%p, have %T@%p", vm.BaseObject, vm.BaseObject, p, p)
	}
}
