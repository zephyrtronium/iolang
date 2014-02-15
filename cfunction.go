package iolang

// A function which can be executed in the context of an Io VM.
type Fn func(vm *VM, self, locals Interface, msg *Message) Interface

// An object representing a statically compiled function, probably written
// in Go for this implementation.
type CFunction struct {
	Object
	Function Fn
	Name     string
}

// Create a new CFunction wrapping f.
func (vm *VM) NewCFunction(f Fn, name string) *CFunction {
	return &CFunction{
		Object{Slots: vm.DefaultSlots["CFunction"], Protos: []Interface{vm.BaseObject}},
		f,
		name,
	}
}

func (f *CFunction) Clone() Interface {
	return &CFunction{
		Object{Slots: Slots{}, Protos: []Interface{f}},
		f.Function,
		f.Name,
	}
}

// Call the wrapped function.
func (f *CFunction) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	return f.Function(vm, target, locals, msg)
}

func (f *CFunction) String() string {
	return f.Name
}
