package iolang

// An Fn is a statically compiled function which can be executed in the context
// of an Io VM.
type Fn func(vm *VM, self, locals Interface, msg *Message) Interface

// A CFunction is an object representing a statically compiled function,
// probably written in Go for this implementation.
type CFunction struct {
	Object
	Function Fn
	Name     string
}

// NewCFunction creates a new CFunction wrapping f, with the given name used
// as the string representation of the function.
func (vm *VM) NewCFunction(f Fn, name string) *CFunction {
	return &CFunction{
		Object{Slots: vm.DefaultSlots["CFunction"], Protos: []Interface{vm.BaseObject}},
		f,
		name,
	}
}

// Clone creates a clone of the CFunction with the same function and name.
func (f *CFunction) Clone() Interface {
	return &CFunction{
		Object{Slots: Slots{}, Protos: []Interface{f}},
		f.Function,
		f.Name,
	}
}

// Activate calls the wrapped function.
func (f *CFunction) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	return f.Function(vm, target, locals, msg)
}

// String returns the name of the object. This is invoked by the default
// asString method in Io.
func (f *CFunction) String() string {
	return f.Name
}
