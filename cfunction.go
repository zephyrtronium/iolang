package iolang

import (
	"path"
	"reflect"
	"runtime"
)

// An Fn is a statically compiled function which can be executed in the context
// of an Io VM.
type Fn func(vm *VM, target, locals *Object, msg *Message) *Object

// A CFunction is object value representing a compiled function.
type CFunction struct {
	Function Fn
	Type     Tag
	Name     string
}

// tagCFunction is the tag type for CFunctions.
type tagCFunction struct{}

func (tagCFunction) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self.Value.(CFunction).Function(vm, target, locals, msg)
}

func (tagCFunction) CloneValue(value interface{}) interface{} {
	return value
}

func (tagCFunction) String() string {
	return "CFunction"
}

// CFunctionTag is the tag for CFunctions. Activate calls the wrapped function.
// CloneValue returns the same function.
var CFunctionTag Tag = tagCFunction{}

// NewCFunction creates a new CFunction object wrapping f. If kind is not nil,
// then the CFunction will raise an exception when called on a target with a
// different tag.
func (vm *VM) NewCFunction(f Fn, kind Tag) *Object {
	u := reflect.ValueOf(f).Pointer()
	name := path.Base(runtime.FuncForPC(u).Name())
	if kind != nil {
		return &Object{
			Protos: vm.CoreProto("CFunction"),
			Value: CFunction{
				Function: func(vm *VM, target, locals *Object, msg *Message) *Object {
					if target.Tag != kind {
						return vm.RaiseExceptionf("receiver of %s must be %v, not %v", name, kind, target.Tag)
					}
					return f(vm, target, locals, msg)
				},
				Type: kind,
				Name: name,
			},
			Tag: CFunctionTag,
		}
	}
	return &Object{
		Protos: vm.CoreProto("CFunction"),
		Value: CFunction{
			Function: f,
			Name:     path.Base(runtime.FuncForPC(u).Name()),
		},
		Tag: CFunctionTag,
	}
}

// String returns the name of the object.
func (f CFunction) String() string {
	return f.Name
}

func (vm *VM) initCFunction() {
	// We can't use NewCFunction yet because the proto doesn't exist. We also
	// want Core CFunction to be a CFunction, but one that won't panic if it's
	// used.
	slots := Slots{}
	proto := &Object{
		Protos: []*Object{vm.BaseObject},
		Value:  CFunction{Function: ObjectThisContext, Name: "ObjectThisContext"},
		Tag:    CFunctionTag,
	}
	vm.Core.SetSlot("CFunction", proto)
	// Now we can create CFunctions.
	slots["=="] = vm.NewCFunction(CFunctionEqual, CFunctionTag)
	slots["asString"] = vm.NewCFunction(CFunctionAsString, CFunctionTag)
	slots["asSimpleString"] = slots["asString"]
	slots["id"] = vm.NewCFunction(CFunctionID, CFunctionTag)
	slots["name"] = slots["asString"]
	slots["performOn"] = vm.NewCFunction(CFunctionPerformOn, CFunctionTag)
	slots["typeName"] = vm.NewCFunction(CFunctionTypeName, CFunctionTag)
	slots["uniqueName"] = vm.NewCFunction(CFunctionUniqueName, CFunctionTag)
}

// CFunctionAsString is a CFunction method.
//
// asString returns a string representation of the object.
func CFunctionAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(target.Value.(CFunction).Name)
}

// CFunctionEqual is a CFunction method.
//
// == returns whether the two CFunctions hold the same internal function.
func CFunctionEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	f := target.Value.(CFunction)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	other, ok := r.Value.(CFunction)
	if !ok {
		return vm.RaiseExceptionf("argument 0 to == must be CFunction, not %s", vm.TypeName(r))
	}
	return vm.IoBool(reflect.ValueOf(f.Function).Pointer() == reflect.ValueOf(other.Function).Pointer())
}

// CFunctionID is a CFunction method.
//
// id returns a unique number for the function invoked by the CFunction.
func CFunctionID(vm *VM, target, locals *Object, msg *Message) *Object {
	f := target.Value.(CFunction)
	u := reflect.ValueOf(f.Function).Pointer()
	return vm.NewNumber(float64(u))
}

// CFunctionPerformOn is a CFunction method.
//
// performOn activates the CFunction using the supplied settings.
func CFunctionPerformOn(vm *VM, target, locals *Object, msg *Message) *Object {
	f := target.Value.(CFunction)
	nt, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(nt, stop)
	}
	nl := locals
	nm := msg
	if msg.ArgCount() > 1 {
		nl, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(nl, stop)
		}
		if msg.ArgCount() > 2 {
			var err *Object
			nm, err, stop = msg.MessageArgAt(vm, locals, 2)
			if stop != NoStop {
				return vm.Stop(err, stop)
			}
		}
	}
	// The original implementation allows one to supply a slotContext, but it
	// is never used.
	return f.Function(vm, nt, nl, nm)
}

// CFunctionTypeName is a CFunction method.
//
// typeName returns the name of the type to which the CFunction is assigned.
func CFunctionTypeName(vm *VM, target, locals *Object, msg *Message) *Object {
	f := target.Value.(CFunction)
	if f.Type == nil {
		return vm.Nil
	}
	return vm.NewString(f.Type.String())
}

// CFunctionUniqueName is a CFunction method.
//
// uniqueName returns the name of the function.
func CFunctionUniqueName(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(target.Value.(CFunction).Name)
}
