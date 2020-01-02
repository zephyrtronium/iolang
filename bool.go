package iolang

func (vm *VM) initTrue() {
	vm.True.proto = vm.BaseObject
	s := vm.NewString("true")
	slots := Slots{
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.True,
		"else":           vm.True,
		"elseif":         vm.True,
		"ifFalse":        vm.True,
		"not":            vm.False,
		"or":             vm.True,
		"type":           s,
	}
	vm.SetSlots(vm.True, slots)
	vm.SetSlot(vm.Core, "true", vm.True)
}

func (vm *VM) initFalse() {
	vm.False.proto = vm.BaseObject
	s := vm.NewString("false")
	slots := Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.False,
		"ifTrue":         vm.False,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.False,
		"type":           s,
	}
	vm.SetSlots(vm.False, slots)
	vm.SetSlot(vm.Core, "false", vm.False)
}

func (vm *VM) initNil() {
	vm.Nil.proto = vm.BaseObject
	s := vm.NewString("nil")
	slots := Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"catch":          vm.Nil,
		"clone":          vm.Nil,
		"else":           vm.Nil,
		"elseif":         vm.Nil,
		"isNil":          vm.True,
		"isTrue":         vm.False,
		"not":            vm.True,
		"returnIfNonNil": vm.Nil,
		"then":           vm.Nil,
		"type":           s,
	}
	vm.SetSlots(vm.Nil, slots)
	vm.SetSlot(vm.Core, "nil", vm.Nil)
}
