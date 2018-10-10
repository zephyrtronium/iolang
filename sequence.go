package iolang

import "fmt"

type Sequence interface {
	Interface
	// Sequence methods?
}

type String struct {
	Object
	Value string
}

func (vm *VM) NewString(value string) *String {
	if s, ok := vm.StringMemo[value]; ok {
		return s
	}
	return &String{
		*vm.CoreInstance("String"),
		value,
	}
}

func (s *String) Clone() Interface {
	return &String{
		Object{Slots: Slots{}, Protos: []Interface{s}},
		s.Value,
	}
}

func (s *String) String() string {
	return fmt.Sprintf("%q", s.Value)
}

func (vm *VM) initSequence() {
	// We can't use vm.NewString yet!!
	// Just create slots for String for now since sequence types don't actually
	// exist yet.
	// Io does have a Core String proto, which is an object having no protos
	// and only a single slot, "type", set to "ImmutableSequence". Not sure
	// what's going on with it.
	slots := Slots{}
	SetSlot(vm.Core, "String", vm.ObjectWith(slots))
	// Now we can use vm.NewString.
	slots["type"] = vm.NewString("ImmutableSequence")
}
