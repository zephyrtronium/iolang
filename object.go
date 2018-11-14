package iolang

import (
	"fmt"
	"reflect"
	"sync"
)

// Interface is the interface which all Io objects satisfy. To satisfy this
// interface, *Object's method set must be embedded, then Clone() implemented
// to return a value of the new type and Activate() implemented to return the
// result of the object.
type Interface interface {
	// Get slots and protos.
	SP() *Object
	// Produce a result. For most objects, this returns self.
	Activate(vm *VM, target, locals Interface, msg *Message) Interface
	// Create an object with empty slots and this object as its only proto.
	Clone() Interface

	isIoObject()
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

// Activate returns the object.
func (o *Object) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
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
		"":           vm.NewCFunction(ObjectEvalArg),
		"!=":         vm.NewCFunction(ObjectNotEqual),
		"<":          vm.NewCFunction(ObjectLess),
		"<=":         vm.NewCFunction(ObjectLessOrEqual),
		"==":         vm.NewCFunction(ObjectEqual),
		">":          vm.NewCFunction(ObjectGreater),
		">=":         vm.NewCFunction(ObjectGreaterOrEqual),
		"asString":   vm.NewCFunction(ObjectAsString),
		"break":      vm.NewCFunction(ObjectBreak),
		"block":      vm.NewCFunction(ObjectBlock),
		"clone":      vm.NewCFunction(ObjectClone),
		"compare":    vm.NewCFunction(ObjectCompare),
		"continue":   vm.NewCFunction(ObjectContinue),
		"do":         vm.NewCFunction(ObjectDo),
		"for":        vm.NewCFunction(ObjectFor),
		"getSlot":    vm.NewCFunction(ObjectGetSlot),
		"if":         vm.NewCFunction(ObjectIf),
		"isTrue":     vm.True,
		"lexicalDo":  vm.NewCFunction(ObjectLexicalDo),
		"loop":       vm.NewCFunction(ObjectLoop),
		"method":     vm.NewCFunction(ObjectMethod),
		"protos":     vm.NewCFunction(ObjectProtos),
		"return":     vm.NewCFunction(ObjectReturn),
		"setSlot":    vm.NewCFunction(ObjectSetSlot),
		"slotNames":  vm.NewCFunction(ObjectSlotNames),
		"type":       vm.NewString("Object"),
		"updateSlot": vm.NewCFunction(ObjectUpdateSlot),
		"while":      vm.NewCFunction(ObjectWhile),
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

// TypeName gets the name of the type of an object by activating its type slot.
// If there is no such slot, the Go type name will be returned.
func (vm *VM) TypeName(o Interface) string {
	if typ, proto := GetSlot(o, "type"); proto != nil {
		return vm.AsString(typ)
	}
	return fmt.Sprintf("%T", o)
}

// SimpleActivate activates an Actor using the identifier message named with
// text and with the given arguments. This does not propagate exceptions.
func (vm *VM) SimpleActivate(o, self, locals Interface, text string, args ...Interface) Interface {
	a := make([]*Message, len(args))
	for i, arg := range args {
		// Since we are setting memos, we don't have to set anything else
		// because the evaluator will see the memo first. If that behavior
		// changes, this must be changed with it.
		a[i] = &Message{Memo: arg}
	}
	// TODO: should this use CheckStop to propagate exceptions?
	result, _ := CheckStop(o.Activate(vm, self, locals, &Message{Text: text, Args: a}), ExceptionStop)
	return result
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals Interface, msg *Message) Interface {
	clone := target.Clone()
	if init, proto := GetSlot(target, "init"); proto != nil {
		r, ok := CheckStop(init.Activate(vm, clone, locals, vm.IdentMessage("init")), LoopStops)
		if !ok {
			return r
		}
	}
	return clone
}

// ObjectSetSlot is an Object method.
//
// setSlot sets the value of a slot on this object. It is typically invoked via
// the := operator.
func ObjectSetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if ok {
		SetSlot(target, slot.String(), v)
	}
	return v
}

// ObjectUpdateSlot is an Object method.
//
// updateSlot sets the value of a slot on the proto which has it. If no object
// has the target slot, an exception is raised. This is typically invoked via
// the = operator.
func ObjectUpdateSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if ok {
		_, proto := GetSlot(target, slot.String())
		if proto == nil {
			return vm.RaiseExceptionf("slot %s not found", slot.String())
		}
		SetSlot(proto, slot.String(), v)
	}
	return v
}

// ObjectGetSlot is an Object method.
//
// getSlot gets the value of a slot. The slot is never activated.
func ObjectGetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v, _ := GetSlot(target, slot.String())
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

// ObjectDo is an Object method.
//
// do evaluates its message in the context of the receiver.
func ObjectDo(vm *VM, target, locals Interface, msg *Message) Interface {
	return msg.EvalArgAt(vm, target, 0)
}

// ObjectLexicalDo is an Object method.
//
// lexicalDo appends the lexical context to the receiver's protos, evaluates
// the message in the context of the receiver, then removes the added proto.
func ObjectLexicalDo(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	n := len(o.Protos)
	o.Protos = append(o.Protos, locals)
	result := msg.EvalArgAt(vm, target, 0)
	copy(o.Protos[n:], o.Protos[n+1:])
	o.Protos = o.Protos[:len(o.Protos)-1]
	return result
}

// ObjectAsString is an Object method.
//
// asString creates a string representation of an object.
func ObjectAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	// Non-standard, but if the object is one of the important basic objects,
	// return a more informative name.
	switch target {
	case vm.BaseObject:
		return vm.NewString(fmt.Sprintf("Core Object_%p", target))
	case vm.Lobby:
		return vm.NewString(fmt.Sprintf("Lobby_%p", target))
	case vm.Core:
		return vm.NewString(fmt.Sprintf("Core_%p", target))
	}
	if stringer, ok := target.(fmt.Stringer); ok {
		return vm.NewString(stringer.String())
	}
	return vm.NewString(fmt.Sprintf("%T_%p", target, target))
}

// Compare uses the compare method of x to compare to y. The result should be a
// *Number holding -1, 0, or 1, if the compare method is proper and no
// exception occurs. If an exception is raised, its Stop will be returned.
func (vm *VM) Compare(x, y Interface) Interface {
	cmp, proto := GetSlot(x, "compare")
	if proto == nil {
		// No compare method.
		return vm.NewNumber(float64(ptrCompare(x, y)))
	}
	arg := &Message{Memo: y}
	r, _ := CheckStop(cmp.Activate(vm, x, x, vm.IdentMessage("compare", arg)), ReturnStop)
	return r
}

// ObjectCompare is an Object method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal. The default order is the order of the
// numeric values of the objects' addresses.
func ObjectCompare(vm *VM, target, locals Interface, msg *Message) Interface {
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return v
	}
	return vm.NewNumber(float64(ptrCompare(target, v)))
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

// ObjectLess is an Object method.
//
// x <(y) returns true if the result of x compare(y) is -1.
func ObjectLess(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value < 0)
	}
	return vm.IoBool(ptrCompare(target, c) < 0)
}

// ObjectLessOrEqual is an Object method.
//
// x <=(y) returns true if the result of x compare(y) is not 1.
func ObjectLessOrEqual(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value <= 0)
	}
	return vm.IoBool(ptrCompare(target, c) <= 0)
}

// ObjectEqual is an Object method.
//
// x ==(y) returns true if the result of x compare(y) is 0.
func ObjectEqual(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value == 0)
	}
	return vm.IoBool(ptrCompare(target, c) == 0)
}

// ObjectNotEqual is an Object method.
//
// x !=(y) returns true if the result of x compare(y) is not 0.
func ObjectNotEqual(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value != 0)
	}
	return vm.IoBool(ptrCompare(target, c) != 0)
}

// ObjectGreaterOrEqual is an Object method.
//
// x >=(y) returns true if the result of x compare(y) is not -1.
func ObjectGreaterOrEqual(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value >= 0)
	}
	return vm.IoBool(ptrCompare(target, c) >= 0)
}

// ObjectGreater is an Object method.
//
// x >(y) returns true if the result of x compare(y) is 1.
func ObjectGreater(vm *VM, target, locals Interface, msg *Message) Interface {
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), ReturnStop)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value > 0)
	}
	return vm.IoBool(ptrCompare(target, c) > 0)
}
