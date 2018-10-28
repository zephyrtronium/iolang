package iolang

func (vm *VM) initTrue() {
	vm.True.Protos = []Interface{vm.BaseObject}
	s := vm.NewString("true")
	vm.True.Slots = Slots{
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.True,
		"else":           vm.True,
		"elseif":         vm.True,
		"ifFalse":        vm.True,
		"ifTrue":         vm.NewCFunction(ObjectEvalArgAndReturnSelf),
		"not":            vm.False,
		"or":             vm.True,
		"type":           s,
	}
	vm.True.Slots["then"] = vm.True.Slots["ifTrue"]
	SetSlot(vm.Core, "true", vm.True)
}

func (vm *VM) initFalse() {
	vm.False.Protos = []Interface{vm.BaseObject}
	s := vm.NewString("false")
	vm.False.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.False,
		"else":           vm.NewCFunction(ObjectEvalArgAndReturnNil),
		"elseif":         vm.NewCFunction(ObjectIf),
		"ifFalse":        vm.NewCFunction(ObjectEvalArgAndReturnSelf),
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
	s := vm.NewString("nil")
	vm.Nil.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"clone":          vm.Nil,
		"else":           vm.Nil,
		"elseif":         vm.Nil,
		"ifNil":          vm.NewCFunction(ObjectEvalArgAndReturnSelf),
		"ifNilEval":      vm.NewCFunction(ObjectEvalArg),
		"ifNonNil":       vm.Nil, // TODO: Io calls "Object_thisContext()"
		"ifNonNilEval":   vm.Nil, // TODO: same
		"isNil":          vm.True,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.Nil,
		"type":           s,
	}
	SetSlot(vm.Core, "nil", vm.Nil)
}
