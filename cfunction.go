package iolang

import (
	"path"
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
	Type     reflect.Type
	Name     string
}

// NewCFunction creates a new CFunction wrapping f.
func (vm *VM) NewCFunction(f Fn) *CFunction {
	u := reflect.ValueOf(f).Pointer()
	return &CFunction{
		Object:   *vm.CoreInstance("CFunction"),
		Function: f,
		Name:     path.Base(runtime.FuncForPC(u).Name()),
	}
}

// NewTypedCFunction creates a new CFunction that raises an exception when
// called on a target of a type different from that of the exemplar.
func (vm *VM) NewTypedCFunction(f Fn, exemplar Interface) *CFunction {
	u := reflect.ValueOf(f).Pointer()
	name := path.Base(runtime.FuncForPC(u).Name())
	typ := reflect.TypeOf(exemplar)
	return &CFunction{
		Object: *vm.CoreInstance("CFunction"),
		Function: func(vm *VM, target, locals Interface, msg *Message) (result Interface) {
			if ttyp := reflect.TypeOf(target); ttyp != typ {
				return vm.RaiseExceptionf("receiver of %s must be %v, not %v", name, typ, ttyp)
			}
			return f(vm, target, locals, msg)
		},
		Type: typ,
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
	// NOTE: We can't use vm.NewString yet because initSequence has to wait
	// until after this. Use initCFunction2 instead.
	var exemplar *CFunction
	slots := Slots{}
	SetSlot(vm.Core, "CFunction", vm.ObjectWith(slots))
	// Now we can create CFunctions.
	slots["=="] = vm.NewTypedCFunction(CFunctionEqual, exemplar)
	slots["id"] = vm.NewTypedCFunction(CFunctionID, exemplar)
	slots["performOn"] = vm.NewTypedCFunction(CFunctionPerformOn, exemplar)
	slots["typeName"] = vm.NewTypedCFunction(CFunctionTypeName, exemplar)
	slots["uniqueName"] = vm.NewTypedCFunction(CFunctionUniqueName, exemplar)
}

func (vm *VM) initCFunction2() {
	slots := vm.Core.Slots["CFunction"].SP().Slots
	slots["type"] = vm.NewString("CFunction")
}

// CFunctionEqual is a CFunction method.
//
// == returns whether the two CFunctions hold the same internal function.
func CFunctionEqual(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*CFunction)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	other, ok := r.(*CFunction)
	if !ok {
		return vm.RaiseException("argument 0 to == must be CFunction, not " + vm.TypeName(r))
	}
	return vm.IoBool(reflect.ValueOf(f.Function).Pointer() == reflect.ValueOf(other.Function).Pointer())
}

// CFunctionID is a CFunction method.
//
// id returns a unique number for the function invoked by the CFunction.
func CFunctionID(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*CFunction)
	u := reflect.ValueOf(f.Function).Pointer()
	return vm.NewNumber(float64(u))
}

// CFunctionPerformOn is a CFunction method.
//
// performOn activates the CFunction using the supplied settings.
func CFunctionPerformOn(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*CFunction)
	nt, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return nt
	}
	nl := locals
	nm := msg
	if msg.ArgCount() > 1 {
		nl, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
		if !ok {
			return nl
		}
		if msg.ArgCount() > 2 {
			r, ok := CheckStop(msg.EvalArgAt(vm, locals, 2), LoopStops)
			if !ok {
				return nm
			}
			if nm, ok = r.(*Message); !ok {
				return vm.RaiseException("argument 2 to performOn must be Message, not " + vm.TypeName(r))
			}
		}
	}
	return f.Activate(vm, nt, nl, nm)
}

// CFunctionTypeName is a CFunction method.
//
// typeName returns the name of the type to which the CFunction is assigned.
func CFunctionTypeName(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*CFunction)
	if f.Type == nil {
		return vm.Nil
	}
	return vm.NewString(f.Type.String())
}

// CFunctionUniqueName is a CFunction method.
//
// uniqueName returns the name of the function.
func CFunctionUniqueName(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(target.(*CFunction).Name)
}
