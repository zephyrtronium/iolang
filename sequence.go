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
		Object{Slots: vm.DefaultSlots["String"], Protos: []Interface{vm.BaseObject}},
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
