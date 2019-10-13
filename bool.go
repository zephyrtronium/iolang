package iolang

func (vm *VM) initTrue() {
	vm.True.Protos = []*Object{vm.BaseObject}
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
	vm.Core.SetSlot("true", vm.True)
}

func (vm *VM) initFalse() {
	vm.False.Protos = []*Object{vm.BaseObject}
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
	vm.Core.SetSlot("false", vm.False)
}

func (vm *VM) initNil() {
	vm.Nil.Protos = []*Object{vm.BaseObject}
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
		"returnIfNonNil": vm.Nil,
		"then":           vm.Nil,
		"type":           s,
	}
	vm.Core.SetSlot("nil", vm.Nil)
}
