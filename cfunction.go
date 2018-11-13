package iolang

import (
	"reflect"
	"runtime"
)

// An Fn is a statically compiled function which can be executed in the context
// of an Io VM.
type Fn func(vm *VM, self, locals Interface, msg *Message) Interface

// A CFunction is an Io object representing a compiled function.
type CFunction struct {
	Object
	Function Fn
	Name     string
	Type     reflect.Type
}

// NewCFunction creates a new CFunction wrapping f.
func (vm *VM) NewCFunction(f Fn) *CFunction {
	u := reflect.ValueOf(f).Pointer()
	return &CFunction{
		Object:   *vm.CoreInstance("CFunction"),
		Function: f,
		Name:     runtime.FuncForPC(u).Name(),
	}
}

// NewTypedCFunction creates a new CFunction that raises an exception when
// called on a target of a type different from that of the exemplar.
func (vm *VM) NewTypedCFunction(f Fn, exemplar Interface) *CFunction {
	u := reflect.ValueOf(f).Pointer()
	name := runtime.FuncForPC(u).Name()
	typ := reflect.TypeOf(exemplar)
	return &CFunction{
		Object: *vm.CoreInstance("CFunction"),
		Function: func(vm *VM, target, locals Interface, msg *Message) (result Interface) {
			if ttyp := reflect.TypeOf(target); ttyp != typ {
				return vm.RaiseExceptionf("receiver of %s must be %v, not %v", name, typ, ttyp)
			}
			return f(vm, target, locals, msg)
		},
		Name: name,
	}
}

// Clone creates a clone of the CFunction with the same function and name.
func (f *CFunction) Clone() Interface {
	return &CFunction{
		Object:   Object{Slots: Slots{}, Protos: []Interface{f}},
		Function: f.Function,
		Name:     f.Name,
		Type:     f.Type,
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

func (vm *VM) initCFunction() {
	// TODO: CFunction slots
	// NOTE: We can't use vm.NewString yet because initSequence has to wait
	// until after this. Use initCFunction2 instead.
	slots := Slots{}
	SetSlot(vm.Core, "CFunction", vm.ObjectWith(slots))
}

func (vm *VM) initCFunction2() {
	slots := vm.Core.Slots["CFunction"].SP().Slots
	slots["type"] = vm.NewString("CFunction")
}
