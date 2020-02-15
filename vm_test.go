package iolang

import (
	"reflect"
	"testing"
)

// TestNewVM tests that NewVM creates an object.
func TestNewVM(t *testing.T) {
	// We can use testVM to test NewVM.
	if TestingVM() == nil {
		t.Fatal("testVM is nil")
	}
}

// TestNewVMAttrs tests that a new VM has the attributes we expect.
func TestNewVMAttrs(t *testing.T) {
	vm := TestingVM()
	attrs := []string{
		"Lobby", "Core", "Addons",
		"BaseObject", "True", "False", "Nil", "Operators",
		"Sched", "Control", "Coro",
	}
	v := reflect.ValueOf(vm).Elem()
	for _, attr := range attrs {
		t.Run("Attr"+attr, func(t *testing.T) {
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
		if vm.StartTime.IsZero() {
			t.Fatal("VM attribute StartTime is zero")
		}
	})
}

// TestLobbySlots tests that a new VM Lobby has the slots we expect.
func TestLobbySlots(t *testing.T) {
	vm := TestingVM()
	slots := []string{"Lobby", "Protos"}
	CheckSlots(t, vm.Lobby, slots)
}

// TestLobbyProtos tests that a new VM Lobby has the protos we expect.
func TestLobbyProtos(t *testing.T) {
	vm := TestingVM()
	// Lobby's proto is a generic object that has Core and Addon slots and Core
	// and Addons as protos. Check that this is all correct.
	protos := vm.Lobby.Protos()
	switch len(protos) {
	case 0:
		t.Fatal("Lobby has no protos")
	case 1: // do nothing
	default:
		t.Error("Lobby has too many protos: expected 1, have", len(protos))
	}
	p := protos[0]
	slots := []string{"Core", "Addons"}
	CheckSlots(t, p, slots)
	opro := p.Protos()
	switch len(opro) {
	case 0, 1:
		t.Fatal("Lobby proto has too few protos")
	case 2: // do nothing
	default:
		t.Error("Lobby proto has too many protos: expected 2, have", len(opro))
	}
	if opro[0] != vm.Core {
		t.Errorf("Lobby proto has wrong proto: expected %T@%p (Core), have %T@%p", vm.Core, vm.Core, opro, opro)
	}
	if opro[1] != vm.Addons {
		t.Errorf("Lobby proto has wrong proto: expected %T@%p (Addons), have %T@%p", vm.Addons, vm.Addons, opro, opro)
	}
}

// TestCoreSlots tests that a new VM Core has the slots we expect.
func TestCoreSlots(t *testing.T) {
	slots := []string{
		"Addon",
		"AddonLoader",
		"Block",
		"Break",
		"CFunction",
		// "CLI",
		"Call",
		"Collector",
		// "Compiler",
		"Continue",
		"Coroutine",
		"Date",
		"Debugger",
		"Directory",
		"DirectoryCollector",
		// "DummyLine",
		"Duration",
		"Eol",
		"Error",
		"Exception",
		"File",
		"FileCollector",
		"Future",
		"ImmutableSequence",
		// "Importer",
		"List",
		"Locals",
		"Map",
		"Message",
		"Normal",
		// "Notifier",
		"Number",
		"Object",
		"OperatorTable",
		"Path",
		// "Profiler",
		"Return",
		"RunnerMixIn",
		// "Sandbox",
		"Scheduler",
		"Sequence",
		"SerializationStream",
		"String",
		"System",
		"TestRunner",
		"TestSuite",
		"UnitTest",
		"Vector",
		"false",
		"nil",
		"tildeExpandsTo",
		"true",
		"vector",
	}
	CheckSlots(t, TestingVM().Core, slots)
}

// TestCoreProtos checks that a new VM Core is an Object type.
func TestCoreProtos(t *testing.T) {
	CheckObjectIsProto(t, TestingVM().Core)
}

// TestAddonsSlots checks that a new VM Addons has empty slots.
func TestAddonsSlots(t *testing.T) {
	CheckSlots(t, TestingVM().Addons, nil)
}

// TestAddonsProtos checks that a new VM Addons is an Object type.
func TestAddonsProtos(t *testing.T) {
	CheckObjectIsProto(t, TestingVM().Addons)
}
