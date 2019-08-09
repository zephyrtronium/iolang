package iolang

import (
	"path"
	"reflect"
	"runtime"
)

// An Fn is a statically compiled function which can be executed in the context
// of an Io VM.
type Fn func(vm *VM, self, locals Interface, msg *Message) (result Interface, control Stop)

// A CFunction is an Io object representing a compiled function.
type CFunction struct {
	Object
	Function Fn
	Type     reflect.Type
	Name     string
}

// NewCFunction creates a new CFunction wrapping f. If kind is not nil, then
// the CFunction will raise an exception when called on a target of a different
// concrete type than that of kind.
func (vm *VM) NewCFunction(f Fn, kind Interface) *CFunction {
	u := reflect.ValueOf(f).Pointer()
	name := path.Base(runtime.FuncForPC(u).Name())
	if kind != nil {
		typ := reflect.TypeOf(kind)
		return &CFunction{
			Object: Object{Protos: vm.CoreProto("CFunction")},
			Function: func(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
				if ttyp := reflect.TypeOf(target); ttyp != typ {
					return vm.RaiseExceptionf("receiver of %s must be %v, not %v", name, typ, ttyp)
				}
				return f(vm, target, locals, msg)
			},
			Type: typ,
			Name: name,
		}
	}
	return &CFunction{
		Object:   Object{Protos: vm.CoreProto("CFunction")},
		Function: f,
		Name:     path.Base(runtime.FuncForPC(u).Name()),
	}
}

// Clone creates a clone of the CFunction with the same function and name.
func (f *CFunction) Clone() Interface {
	return &CFunction{
		Object:   Object{Protos: []Interface{f}},
		Function: f.Function,
		Name:     f.Name,
		Type:     f.Type,
	}
}

// Activate calls the wrapped function.
func (f *CFunction) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return f.Function(vm, target, locals, msg)
}

// String returns the name of the object.
func (f *CFunction) String() string {
	return f.Name
}

func (vm *VM) initCFunction() {
	// We can't use NewCFunction yet because the proto doesn't exist. We also
	// want Core CFunction to be a CFunction, but one that won't panic if it's
	// used. Therefore, our kind that is normally just used for its reflected
	// type can also be a fake-ish thisContext for the Core slot.
	slots := Slots{}
	kind := &CFunction{
		Object:   Object{Slots: slots, Protos: []Interface{vm.BaseObject}},
		Function: ObjectThisContext,
	}
	vm.SetSlot(vm.Core, "CFunction", kind)
	// Now we can create CFunctions.
	slots["=="] = vm.NewCFunction(CFunctionEqual, kind)
	slots["asString"] = vm.NewCFunction(CFunctionAsString, kind)
	slots["asSimpleString"] = slots["asString"]
	slots["id"] = vm.NewCFunction(CFunctionID, kind)
	slots["name"] = slots["asString"]
	slots["performOn"] = vm.NewCFunction(CFunctionPerformOn, kind)
	slots["typeName"] = vm.NewCFunction(CFunctionTypeName, kind)
	slots["uniqueName"] = vm.NewCFunction(CFunctionUniqueName, kind)
}

// CFunctionAsString is a CFunction method.
//
// asString returns a string representation of the object.
func CFunctionAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(target.(*CFunction).Name), NoStop
}

// CFunctionEqual is a CFunction method.
//
// == returns whether the two CFunctions hold the same internal function.
func CFunctionEqual(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*CFunction)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	other, ok := r.(*CFunction)
	if !ok {
		return vm.RaiseException("argument 0 to == must be CFunction, not " + vm.TypeName(r))
	}
	return vm.IoBool(reflect.ValueOf(f.Function).Pointer() == reflect.ValueOf(other.Function).Pointer()), NoStop
}

// CFunctionID is a CFunction method.
//
// id returns a unique number for the function invoked by the CFunction.
func CFunctionID(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*CFunction)
	u := reflect.ValueOf(f.Function).Pointer()
	return vm.NewNumber(float64(u)), NoStop
}

// CFunctionPerformOn is a CFunction method.
//
// performOn activates the CFunction using the supplied settings.
func CFunctionPerformOn(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*CFunction)
	nt, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return nt, stop
	}
	nl := locals
	nm := msg
	if msg.ArgCount() > 1 {
		nl, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return nl, stop
		}
		if msg.ArgCount() > 2 {
			var err Interface
			nm, err, stop = msg.MessageArgAt(vm, locals, 2)
			if stop != NoStop {
				return err, stop
			}
		}
	}
	// The original implementation allows one to supply a slotContext, but it
	// is never used.
	return f.Activate(vm, nt, nl, nil, nm)
}

// CFunctionTypeName is a CFunction method.
//
// typeName returns the name of the type to which the CFunction is assigned.
func CFunctionTypeName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*CFunction)
	if f.Type == nil {
		return vm.Nil, NoStop
	}
	return vm.NewString(f.Type.String()), NoStop
}

// CFunctionUniqueName is a CFunction method.
//
// uniqueName returns the name of the function.
func CFunctionUniqueName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(target.(*CFunction).Name), NoStop
}
