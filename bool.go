package iolang

func (vm *VM) initTrue() {
	vm.True.Protos = []Interface{vm.BaseObject}
	object := vm.BaseObject.Slots
	s := vm.NewString("true")
	vm.True.Slots = Slots{
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.True,
		"else":           vm.True,
		"elseif":         vm.True,
		"ifFalse":        vm.True,
		"ifTrue":         object["evalArgAndReturnSelf"],
		"not":            vm.False,
		"or":             vm.True,
		"then":           object["evalArgAndReturnNil"],
		"type":           s,
	}
	SetSlot(vm.Core, "true", vm.True)
}

func (vm *VM) initFalse() {
	vm.False.Protos = []Interface{vm.BaseObject}
	object := vm.BaseObject.Slots
	s := vm.NewString("false")
	vm.False.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.False,
		"else":           object["evalArgAndReturnNil"],
		"elseif":         object["if"],
		"ifFalse":        object["evalArgAndReturnSelf"],
		"ifTrue":         vm.False,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.False,
		"type":           s,
	}
	SetSlot(vm.Core, "false", vm.False)
}

func (vm *VM) initNil() {
	vm.Nil.Protos = []Interface{vm.BaseObject}
	object := vm.BaseObject.Slots
	s := vm.NewString("nil")
	vm.Nil.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"catch":          vm.Nil,
		"clone":          vm.Nil,
		"else":           vm.Nil,
		"elseif":         vm.Nil,
		"ifNil":          object["evalArgAndReturnSelf"],
		"ifNilEval":      object["evalArg"],
		"ifNonNil":       object["thisContext"],
		"ifNonNilEval":   object["thisContext"],
		"isNil":          vm.True,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.Nil,
		"type":           s,
	}
	SetSlot(vm.Core, "nil", vm.Nil)
}
