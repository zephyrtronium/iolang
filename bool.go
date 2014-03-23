package iolang

func (vm *VM) initTrue() {
	vm.True.Protos = []Interface{vm.BaseObject}
	s := vm.NewString("true")
	vm.True.Slots = Slots{
		"asSimpleString": s,
		"asString":       s,
		"else":           vm.True,
		"elseif":         vm.True,
		"ifFalse":        vm.True,
		"ifTrue":         vm.NewCFunction(ObjectEvalArgAndReturnSelf, "ObjectEvalArgAndReturnSelf(msg)"),
		"not":            vm.False,
		"or":             vm.True,
		"type":           s,
	}
}

func (vm *VM) initFalse() {
	vm.False.Protos = []Interface{vm.BaseObject}
	s := vm.NewString("false")
	vm.False.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"else":           vm.NewCFunction(ObjectEvalArgAndReturnNil, "ObjectEvalArgAndReturnNil(msg)"),
		"elseif":         vm.NewCFunction(ObjectIf, "ObjectIf(cond, onTrue, onFalse)"),
		"ifFalse":        vm.NewCFunction(ObjectEvalArgAndReturnSelf, "ObjectEvalArgAndReturnSelf(msg)"),
		"ifTrue":         vm.False,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.False,
		"type":           s,
	}
}

func (vm *VM) initNil() {
	vm.Nil.Protos = []Interface{vm.BaseObject}
	s := vm.NewString("nil")
	vm.Nil.Slots = Slots{
		"and":            vm.False,
		"asSimpleString": s,
		"asString":       s,
		"else":           vm.Nil,
		"elseif":         vm.Nil,
		"ifNil":          vm.NewCFunction(ObjectEvalArgAndReturnSelf, "ObjectEvalArgAndReturnSelf(msg)"),
		"ifNilEval":      vm.NewCFunction(ObjectEvalArg, "ObjectEvalArg(msg)"),
		"ifNonNil":       vm.Nil, // TODO: Io calls "Object_thisContext()"
		"ifNonNilEval":   vm.Nil, // TODO: same
		"isNil":          vm.True,
		"isTrue":         vm.False,
		"not":            vm.True,
		"then":           vm.Nil,
		"type":           s,
	}
}
