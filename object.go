package iolang

import (
	"fmt"
	"reflect"
	"sync"
)

// All Io objects satisfy Interface. To satisfy this interface, *Object's
// method set must be embedded and Clone() implemented to return a value of the
// new type.
type Interface interface {
	// Get slots and protos.
	SP() *Object
	// Create an object with empty slots and this object as its only proto.
	Clone() Interface

	isIoObject()
}

// An Actor is an object which activates.
type Actor interface {
	Interface
	Activate(vm *VM, target, locals Interface, msg *Message) Interface
}

// Slots holds the set of messages to which an object responds.
type Slots map[string]Interface

// Object is the basic type of Io. Everything is an Object.
type Object struct {
	// Slots is the set of messages to which this object responds.
	Slots Slots
	// Protos are the set of objects to which messages are forwarded, in
	// depth-first order without duplicates, when this object cannot respond.
	Protos []Interface

	// The lock should be held when accessing slots or protos directly.
	L sync.Mutex
}

// SP returns this object's slots and protos. It primarily serves as a
// polymorphic way to access slots and protos of types embedding *Object.
func (o *Object) SP() *Object {
	return o
}

// Clone returns a new object with empty slots and this object as its only
// proto.
func (o *Object) Clone() Interface {
	return &Object{Slots: Slots{}, Protos: []Interface{o}}
}

// isIoObject is a method to force types to embed Object to satisfy Interface
// for the sake of consistency.
func (*Object) isIoObject() {}

// initObject sets up the "base" object that is the first proto of all other
// built-in types.
func (vm *VM) initObject() {
	vm.BaseObject.Protos = []Interface{vm.Lobby}
	slots := Slots{
		"":           vm.NewCFunction(ObjectEvalArg, "ObjectEvalArg(msg)"),
		"asString":   vm.NewCFunction(ObjectAsString, "ObjectAsString()"),
		"break":      vm.NewCFunction(ObjectBreak, "ObjectBreak(result)"),
		"block":      vm.NewCFunction(ObjectBlock, "ObjectBlock(args..., msg)"),
		"clone":      vm.NewCFunction(ObjectClone, "ObjectClone()"),
		"compare":    vm.NewCFunction(ObjectCompare, "ObjectCompare()"),
		"continue":   vm.NewCFunction(ObjectContinue, "ObjectContinue()"),
		"for":        vm.NewCFunction(ObjectFor, "ObjectFor(ctr, start, stop, [step,] msg)"),
		"getSlot":    vm.NewCFunction(ObjectGetSlot, "ObjectGetSlot(name)"),
		"if":         vm.NewCFunction(ObjectIf, "ObjectIf(cond, onTrue, onFalse)"),
		"isTrue":     vm.True,
		"loop":       vm.NewCFunction(ObjectLoop, "ObjectLoop(msg)"),
		"method":     vm.NewCFunction(ObjectMethod, "ObjectMethod(args..., msg)"),
		"protos":     vm.NewCFunction(ObjectProtos, "ObjectProtos"),
		"return":     vm.NewCFunction(ObjectReturn, "ObjectReturn(result)"),
		"setSlot":    vm.NewCFunction(ObjectSetSlot, "ObjectSetSlot(name, value)"),
		"slotNames":  vm.NewCFunction(ObjectSlotNames, "ObjectSlotNames()"),
		"type":       vm.NewString("Object"),
		"updateSlot": vm.NewCFunction(ObjectUpdateSlot, "ObjectUpdateSlot(name, value)"),
		"while":      vm.NewCFunction(ObjectWhile, "ObjectWhile(cond, msg)"),
	}
	vm.BaseObject.Slots = slots
	SetSlot(vm.Core, "Object", vm.BaseObject)

	slots["returnIfNonNil"] = slots["return"]
}

// ObjectWith creates a new object with the given slots and with the VM's
// BaseObject as its proto.
func (vm *VM) ObjectWith(slots Slots) *Object {
	return &Object{Slots: slots, Protos: []Interface{vm.BaseObject}}
}

// GetSlot finds the value in a slot, checking protos in depth-first order
// without duplicates. The proto is the object which actually had the slot. If
// the slot is not found, both returned values will be nil.
//
// Note: This function differentiates between Go's nil and Io's nil. The
// result when called with the former is always nil, nil.
func GetSlot(o Interface, slot string) (value, proto Interface) {
	if o == nil {
		return nil, nil
	}
	return getSlotRecurse(o, slot, make(map[*Object]struct{}, len(o.SP().Protos)+1))
}

// var Debugvm *VM
// var _ = Debugvm

func getSlotRecurse(o Interface, slot string, checked map[*Object]struct{}) (Interface, Interface) {
	obj := o.SP()
	if obj.Slots != nil {
		obj.L.Lock()
		// This will not unlock until the entire recursive search finishes.
		// Do we care? Behavior is possibly more predictable this way, but
		// performance may suffer.
		defer obj.L.Unlock()
		// spew.Dump(obj)
		// fmt.Println(ObjectSlotNames(Debugvm, obj, nil, nil))
		if s, ok := obj.Slots[slot]; ok {
			return s, o
		}
		checked[obj] = struct{}{}
		for _, proto := range obj.Protos {
			p := proto.SP()
			if _, skip := checked[p]; skip {
				continue
			}
			if s, pp := getSlotRecurse(p, slot, checked); pp != nil {
				return s, pp
			}
		}
	}
	return nil, nil
}

// SetSlot sets a slot's value on the given Interface, as if using the :=
// operator.
func SetSlot(o Interface, slot string, value Interface) {
	obj := o.SP()
	obj.L.Lock()
	defer obj.L.Unlock()
	if obj.Slots == nil {
		obj.Slots = Slots{}
	}
	obj.Slots[slot] = value
}

// MutableMethod locks an object, so that methods on mutable objects can
// synchronize. The returned value is the unlock function, so that methods can
// call this like:
//
//    defer MutableMethod(target)()
func MutableMethod(o Interface) func() {
	sp := o.SP()
	sp.L.Lock()
	return sp.L.Unlock
}

// TypeName gets the name of the type of an object by activating its type slot.
// If there is no such slot, the Go type name will be returned.
func (vm *VM) TypeName(o Interface) string {
	if typ, proto := GetSlot(o, "type"); proto != nil {
		switch tt := typ.(type) {
		case *String:
			return tt.Value
		case Actor:
			// TODO: provide a Call
			name := vm.SimpleActivate(tt, o, nil, "type")
			if s, ok := name.(*String); ok {
				return s.Value
			}
		}
	}
	return fmt.Sprintf("%T", o)
}

// SimpleActivate activates an Actor using the identifier message named with
// text and with the given arguments.
func (vm *VM) SimpleActivate(o Actor, self, locals Interface, text string, args ...Interface) Interface {
	a := make([]*Message, len(args))
	for i, arg := range args {
		// Since we are setting memos, we don't have to set anything else
		// because the evaluator will see the memo first. If that behavior
		// changes, this must be changed with it.
		a[i] = &Message{Memo: arg}
	}
	// TODO: should this use CheckStop to propagate exceptions?
	result := o.Activate(vm, self, locals, &Message{Symbol: Symbol{Kind: IdentSym, Text: text}, Args: a})
	if stop, ok := result.(Stop); ok {
		return stop.Result
	}
	return result
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals Interface, msg *Message) Interface {
	clone := target.Clone()
	if init, proto := GetSlot(target, "init"); proto != nil {
		if a, ok := init.(Actor); ok {
			if result := vm.SimpleActivate(a, clone, locals, "init"); IsIoError(result) {
				return result
			}
		}
	}
	return clone
}

// ObjectSetSlot is an Object method.
//
// setSlot sets the value of a slot on this object. It is typically invoked via
// the := operator.
func ObjectSetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if ok {
		SetSlot(target, slot.Value, v)
	}
	return v
}

// ObjectUpdateSlot is an Object method.
//
// updateSlot sets the value of a slot on the proto which has it. If no object
// has the target slot, an exception is raised. This is typically invoked via
// the = operator.
func ObjectUpdateSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if ok {
		_, proto := GetSlot(target, slot.Value)
		if proto == nil {
			return vm.NewExceptionf("slot %s not found", slot.Value)
		}
		SetSlot(proto, slot.Value, v)
	}
	return v
}

// ObjectGetSlot is an Object method.
//
// getSlot gets the value of a slot. The slot is never activated.
func ObjectGetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v, _ := GetSlot(target, slot.Value)
	return v
}

// ObjectSlotNames is an Object method.
//
// slotNames returns a list of the names of the slots on this object.
func ObjectSlotNames(vm *VM, target, locals Interface, msg *Message) Interface {
	slots := target.SP().Slots
	names := make([]Interface, 0, len(slots))
	for name := range slots {
		names = append(names, vm.NewString(name))
	}
	return vm.NewList(names...)
}

// ObjectProtos is an Object method.
//
// protos returns a list of the receiver's protos.
func ObjectProtos(vm *VM, target, locals Interface, msg *Message) Interface {
	protos := target.SP().Protos
	v := make([]Interface, len(protos))
	copy(v, protos)
	return vm.NewList(v...)
}

// ObjectEvalArg is an Object method.
//
// evalArg evaluates and returns its argument. It is typically invoked via the
// empty string slot, i.e. parentheses with no preceding message.
func ObjectEvalArg(vm *VM, target, locals Interface, msg *Message) Interface {
	// The original Io implementation has an assertion that there is at least
	// one argument; this will instead return vm.Nil. It wouldn't be difficult
	// to mimic Io's behavior, but ehhh.
	return msg.EvalArgAt(vm, locals, 0)
}

// ObjectEvalArgAndReturnSelf is an Object method.
//
// evalArgAndReturnSelf evaluates its argument and returns this object.
func ObjectEvalArgAndReturnSelf(vm *VM, target, locals Interface, msg *Message) Interface {
	result, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if ok {
		return target
	}
	return result
}

// ObjectEvalArgAndReturnNil is an Object method.
//
// evalArgAndReturnNil evaluates its argument and returns nil.
func ObjectEvalArgAndReturnNil(vm *VM, target, locals Interface, msg *Message) Interface {
	result, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if ok {
		return vm.Nil
	}
	return result
}

// ObjectAsString is an Object method.
//
// asString creates a string representation of an object.
func ObjectAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	if stringer, ok := target.(fmt.Stringer); ok {
		return vm.NewString(stringer.String())
	}
	return vm.NewString(fmt.Sprintf("%T_%p", target, target))
}

// Compare uses the compare method of x to compare to y. The result should be a
// *Number holding -1, 0, or 1, if the compare method is proper and no
// exception occurs. If an exception is raised, its Stop will be returned. In
// the event that the compare slot exists but is not an Actor, it will be
// returned.
func (vm *VM) Compare(x, y Interface) Interface {
	cmp, proto := GetSlot(x, "compare")
	if proto == nil {
		// No compare method.
		return vm.NewNumber(float64(ptrCompare(x, y)))
	}
	if a, ok := cmp.(Actor); ok {
		arg := &Message{Memo: y}
		r, _ := CheckStop(a.Activate(vm, x, x, vm.IdentMessage("compare", arg)), ReturnStop)
		return r
	}
	// If the compare slot isn't an actor, there isn't really much to do except
	// return it, in case someone does `theirObject compare := 1` to benchmark
	// their sorting algorithm or something.
	return cmp
}

// ObjectCompare is an Object method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal. The default order is the order of the
// numeric values of the objects' addresses.
func ObjectCompare(vm *VM, target, locals Interface, msg *Message) Interface {
	if v, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops); !ok {
		return v
	} else {
		return vm.NewNumber(float64(ptrCompare(target, v)))
	}
}

// ptrCompare returns a compare value for the pointers of two objects. It
// panics if the value is not a real object.
func ptrCompare(x, y Interface) int {
	a := reflect.ValueOf(x.SP()).Pointer()
	b := reflect.ValueOf(y.SP()).Pointer()
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
