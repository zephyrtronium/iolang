package iolang

import (
	"testing"
)

// CheckSlots is a testing helper to check whether an object has exactly the
// slots we expect.
func CheckSlots(t *testing.T, obj Interface, slots []string) {
	t.Helper()
	o := obj.SP()
	o.L.Lock()
	defer o.L.Unlock()
	checked := make(map[string]bool, len(slots))
	for _, name := range slots {
		checked[name] = true
		t.Run("Have_" + name, func(t *testing.T) {
			slot, ok := o.Slots[name]
			if !ok {
				t.Fatal("no slot", name)
			}
			if slot == nil {
				t.Fatal("slot", name, "is nil")
			}
		})
	}
	for name := range o.Slots {
		t.Run("Want_" + name, func(t *testing.T) {
			if !checked[name] {
				t.Fatal("unexpected slot", name)
			}
		})
	}
}

// CheckObjectIsProto is a testing helper to check that an object has exactly
// one proto, which is Core Object. obj must come from testVM.
func CheckObjectIsProto(t *testing.T, obj Interface) {
	t.Helper()
	o := obj.SP()
	o.L.Lock()
	defer o.L.Unlock()
	switch len(o.Protos) {
	case 0:
		t.Fatal("no protos")
	case 1: // do nothing
	default:
		t.Error("incorrect number of protos: expected 1, have", len(o.Protos))
	}
	if p := o.Protos[0]; p != testVM.BaseObject {
		t.Errorf("wrong proto: expected %T@%p, have %T@%p", testVM.BaseObject, testVM.BaseObject, p, p)
	}
}
