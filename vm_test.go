package iolang

import (
	"reflect"
	"testing"
)

// testVM is the VM used for all tests.
var testVM *VM

func init() {
	testVM = NewVM()
}

// TestNewVM tests that NewVM creates an object.
func TestNewVM(t *testing.T) {
	// We can use testVM to test NewVM.
	if testVM == nil {
		t.Fatal("testVM is nil")
	}
}

// TestNewVMAttrs tests that a new VM has the attributes we expect.
func TestNewVMAttrs(t *testing.T) {
	attrs := []string{
		"Lobby", "Core", "Addons",
		"BaseObject", "True", "False", "Nil", "Operators",
		"Sched", "Stop",
		"NumberMemo",
	}
	v := reflect.ValueOf(testVM).Elem()
	for _, attr := range attrs {
		t.Run("Attr" + attr, func(t *testing.T) {
			e := v.FieldByName(attr)
			if !e.IsValid() {
				t.Fatal("no VM attribute", attr)
			}
			if e.IsNil() {
				t.Fatal("VM attribute", attr, "is nil")
			}
		})
	}
	t.Run("AttrStartTime", func(t *testing.T) {
		if testVM.StartTime.IsZero() {
			t.Fatal("VM attribute StartTime is zero")
		}
	})
}

// TestLobbySlots tests that a new VM Lobby has the slots we expect.
func TestLobbySlots(t *testing.T) {
	slots := []string{"Lobby", "Protos"}
	CheckSlots(t, testVM.Lobby, slots)
}

// TestLobbyProtos tests that a new VM Lobby has the protos we expect.
func TestLobbyProtos(t *testing.T) {
	// Lobby's proto is a generic object that has Core and Addon slots and Core
	// and Addons as protos. Check that this is all correct.
	switch len(testVM.Lobby.Protos) {
	case 0:
		t.Fatal("Lobby has no protos")
	case 1: // do nothing
	default:
		t.Error("Lobby has too many protos: expected 1, have", len(testVM.Lobby.Protos))
	}
	p := testVM.Lobby.Protos[0]
	slots := []string{"Core", "Addons"}
	CheckSlots(t, p, slots)
	o := p.SP()
	o.L.Lock()
	defer o.L.Unlock()
	switch len(o.Protos) {
	case 0, 1:
		t.Fatal("Lobby proto has too few protos")
	case 2: // do nothing
	default:
		t.Error("Lobby proto has too many protos: expected 2, have", len(o.Protos))
	}
	if o.Protos[0] != testVM.Core {
		t.Errorf("Lobby proto has wrong proto: expected %T@%p (Core), have %T@%p", testVM.Core, testVM.Core, o.Protos[0], o.Protos[0])
	}
	if o.Protos[1] != testVM.Addons {
		t.Errorf("Lobby proto has wrong proto: expected %T@%p (Addons), have %T@%p", testVM.Addons, testVM.Addons, o.Protos[1], o.Protos[1])
	}
}
