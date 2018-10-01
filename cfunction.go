package iolang

import (
	"runtime"
)

// An Fn is a statically compiled function which can be executed in the context
// of an Io VM.
type Fn func(vm *VM, self, locals Interface, msg *Message) Interface

// A CFunction is an object representing a compiled function.
type CFunction struct {
	Object
	Function Fn
	Name     string
}

// NewCFunction creates a new CFunction wrapping f, with the given name used
// as the string representation of the function.
func (vm *VM) NewCFunction(f Fn, name string) *CFunction {
	return &CFunction{
		Object:   Object{Slots: vm.DefaultSlots["CFunction"], Protos: []Interface{vm.BaseObject}},
		Function: f,
		Name:     name,
	}
}

// NewTypedCFunction creates a new CFunction with a wrapper that recovers from
// failed type assertions and returns an appropriate error instead.
func (vm *VM) NewTypedCFunction(f Fn, name string) *CFunction {
	return &CFunction{
		Object: Object{Slots: vm.DefaultSlots["CFunction"], Protos: []Interface{vm.BaseObject}},
		Function: func(vm *VM, target, locals Interface, msg *Message) (result Interface) {
			defer func() {
				e := recover()
				if te, ok := e.(*runtime.TypeAssertionError); ok {
					result = vm.NewExceptionf("error calling %s: %v", name, te)
				} else if e != nil {
					panic(e)
				}
			}()
			return f(vm, target, locals, msg)
		},
		Name: name,
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
