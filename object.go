package iolang

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Interface is the interface which all Io objects satisfy. To satisfy this
// interface, *Object's method set must be embedded, then Clone implemented to
// return a value of the new type and Activate implemented to return the result
// of activating the object.
type Interface interface {
	// Activate produces a result. For most objects, this returns self.
	Activate(vm *VM, target, locals, context Interface, msg *Message) (result Interface, control Stop)
	// Clone creates an object with empty slots and this object as its only
	// proto.
	Clone() Interface

	// The following methods are for direct access to the object properties.
	// They are not synchronized; the object lock must be held to use or modify
	// slots and protos.

	// Lock blocks until acquiring the object's lock.
	Lock()
	// Unlock releases the object's lock.
	Unlock()

	// RawSlots returns the slot map.
	RawSlots() Slots
	// RawSetSlots sets the object's slot map.
	RawSetSlots(slots Slots)
	// Protos returns the object's protos list directly.
	RawProtos() []Interface
	// SetProtos sets the object's protos list.
	RawSetProtos(protos []Interface)
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

	// L is a lock which should be held when accessing slots or protos
	// directly.
	L sync.Mutex
}

// Activate activates the object. If the isActivatable slot is true, and the
// activate slot exists, then this activates that slot; otherwise, it returns
// the object.
func (o *Object) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	ok, proto := vm.GetSlot(o, "isActivatable")
	// We can't use vm.AsBool even though it's one of the few situations where
	// we'd want to, because it will attempt to activate the isTrue slot, which
	// is typically a plain Object, which will activate this method and recurse
	// infinitely.
	if proto == nil || ok == vm.False || ok == vm.Nil {
		return o, NoStop
	}
	act, proto := vm.GetSlot(o, "activate")
	if proto != nil {
		return act.Activate(vm, target, locals, context, msg)
	}
	return o, NoStop
}

// Clone returns a new object with empty slots and this object as its only
// proto.
func (o *Object) Clone() Interface {
	return &Object{Protos: []Interface{o}}
}

// Lock blocks until acquiring the object's lock. This lock is to synchronize
// only slots and protos, not values.
func (o *Object) Lock() {
	o.L.Lock()
}

// Unlock releases the object's lock.
func (o *Object) Unlock() {
	o.L.Unlock()
}

// RawSlots returns the object's slot map directly. It is not synchronized;
// callers must hold the object's lock to use this safely.
func (o *Object) RawSlots() Slots {
	return o.Slots
}

// RawSetSlots sets the object's slot map directly. It is not synchronized;
// callers must hold the object's lock to use this safely.
func (o *Object) RawSetSlots(slots Slots) {
	o.Slots = slots
}

// RawProtos returns the object's protos list directly. It is not synchronized;
// callers must hold the object's lock to use this safely.
func (o *Object) RawProtos() []Interface {
	return o.Protos
}

// RawSetProtos sets the object's protos list directly. It is not synchronized;
// callers must hold the object's lock to use this safely.
func (o *Object) RawSetProtos(protos []Interface) {
	o.Protos = protos
}

// initObject sets up the "base" object that is the first proto of all other
// built-in types.
func (vm *VM) initObject() {
	vm.BaseObject.Protos = []Interface{vm.Lobby}
	slots := Slots{
		"":                     vm.NewCFunction(ObjectEvalArg, nil),
		"!=":                   vm.NewCFunction(ObjectNotEqual, nil),
		"<":                    vm.NewCFunction(ObjectLess, nil),
		"<=":                   vm.NewCFunction(ObjectLessOrEqual, nil),
		"==":                   vm.NewCFunction(ObjectEqual, nil),
		">":                    vm.NewCFunction(ObjectGreater, nil),
		">=":                   vm.NewCFunction(ObjectGreaterOrEqual, nil),
		"ancestorWithSlot":     vm.NewCFunction(ObjectAncestorWithSlot, nil),
		"appendProto":          vm.NewCFunction(ObjectAppendProto, nil),
		"asGoRepr":             vm.NewCFunction(ObjectAsGoRepr, nil),
		"asString":             vm.NewCFunction(ObjectAsString, nil),
		"asyncSend":            vm.NewCFunction(ObjectAsyncSend, nil), // future.go
		"block":                vm.NewCFunction(ObjectBlock, nil),
		"break":                vm.NewCFunction(ObjectBreak, nil),
		"clone":                vm.NewCFunction(ObjectClone, nil),
		"cloneWithoutInit":     vm.NewCFunction(ObjectCloneWithoutInit, nil),
		"compare":              vm.NewCFunction(ObjectCompare, nil),
		"contextWithSlot":      vm.NewCFunction(ObjectContextWithSlot, nil),
		"continue":             vm.NewCFunction(ObjectContinue, nil),
		"do":                   vm.NewCFunction(ObjectDo, nil),
		"doFile":               vm.NewCFunction(ObjectDoFile, nil),
		"doMessage":            vm.NewCFunction(ObjectDoMessage, nil),
		"doString":             vm.NewCFunction(ObjectDoString, nil),
		"evalArgAndReturnNil":  vm.NewCFunction(ObjectEvalArgAndReturnNil, nil),
		"evalArgAndReturnSelf": vm.NewCFunction(ObjectEvalArgAndReturnSelf, nil),
		"for":                  vm.NewCFunction(ObjectFor, nil),
		"foreachSlot":          vm.NewCFunction(ObjectForeachSlot, nil),
		"futureSend":           vm.NewCFunction(ObjectFutureSend, nil), // future.go
		"getLocalSlot":         vm.NewCFunction(ObjectGetLocalSlot, nil),
		"getSlot":              vm.NewCFunction(ObjectGetSlot, nil),
		"hasLocalSlot":         vm.NewCFunction(ObjectHasLocalSlot, nil),
		"if":                   vm.NewCFunction(ObjectIf, nil),
		"isError":              vm.False,
		"isIdenticalTo":        vm.NewCFunction(ObjectIsIdenticalTo, nil),
		"isKindOf":             vm.NewCFunction(ObjectIsKindOf, nil),
		"isNil":                vm.False,
		"isTrue":               vm.True,
		"lexicalDo":            vm.NewCFunction(ObjectLexicalDo, nil),
		"loop":                 vm.NewCFunction(ObjectLoop, nil),
		"message":              vm.NewCFunction(ObjectMessage, nil),
		"method":               vm.NewCFunction(ObjectMethod, nil),
		"not":                  vm.Nil,
		"or":                   vm.True,
		"perform":              vm.NewCFunction(ObjectPerform, nil),
		"performWithArgList":   vm.NewCFunction(ObjectPerformWithArgList, nil),
		"prependProto":         vm.NewCFunction(ObjectPrependProto, nil),
		"protos":               vm.NewCFunction(ObjectProtos, nil),
		"removeAllProtos":      vm.NewCFunction(ObjectRemoveAllProtos, nil),
		"removeAllSlots":       vm.NewCFunction(ObjectRemoveAllSlots, nil),
		"removeProto":          vm.NewCFunction(ObjectRemoveProto, nil),
		"removeSlot":           vm.NewCFunction(ObjectRemoveSlot, nil),
		"return":               vm.NewCFunction(ObjectReturn, nil),
		"setProto":             vm.NewCFunction(ObjectSetProto, nil),
		"setProtos":            vm.NewCFunction(ObjectSetProtos, nil),
		"setSlot":              vm.NewCFunction(ObjectSetSlot, nil),
		"shallowCopy":          vm.NewCFunction(ObjectShallowCopy, nil),
		"slotNames":            vm.NewCFunction(ObjectSlotNames, nil),
		"slotValues":           vm.NewCFunction(ObjectSlotValues, nil),
		"thisContext":          vm.NewCFunction(ObjectThisContext, nil),
		"thisLocalContext":     vm.NewCFunction(ObjectThisLocalContext, nil),
		"thisMessage":          vm.NewCFunction(ObjectThisMessage, nil),
		"try":                  vm.NewCFunction(ObjectTry, nil),
		"type":                 vm.NewString("Object"),
		"uniqueId":             vm.NewCFunction(ObjectUniqueId, nil),
		"updateSlot":           vm.NewCFunction(ObjectUpdateSlot, nil),
		"wait":                 vm.NewCFunction(ObjectWait, nil),
		"while":                vm.NewCFunction(ObjectWhile, nil),
	}
	slots["evalArg"] = slots[""]
	slots["ifError"] = slots["thisContext"]
	slots["ifNil"] = slots["thisContext"]
	slots["ifNilEval"] = slots["thisContext"]
	slots["ifNonNil"] = slots["evalArgAndReturnSelf"]
	slots["ifNonNilEval"] = slots["evalArg"]
	slots["raiseIfError"] = slots["thisContext"]
	slots["returnIfError"] = slots["thisContext"]
	slots["returnIfNonNil"] = slots["return"]
	slots["uniqueHexId"] = slots["uniqueId"]
	vm.BaseObject.Slots = slots
	vm.SetSlot(vm.Core, "Object", vm.BaseObject)
}

// ObjectWith creates a new object with the given slots and with the VM's
// Core Object as its proto.
func (vm *VM) ObjectWith(slots Slots) *Object {
	return &Object{Slots: slots, Protos: []Interface{vm.BaseObject}}
}

// TypeName gets the name of the type of an object by activating its type slot.
// If there is no such slot, the Go type name will be returned.
func (vm *VM) TypeName(o Interface) string {
	if typ, proto := vm.GetSlot(o, "type"); proto != nil {
		return vm.AsString(typ)
	}
	return fmt.Sprintf("%T", o)
}

// SimpleActivate activates an object using the identifier message named with
// text and with the given arguments. This does not propagate control flow.
func (vm *VM) SimpleActivate(o, self, locals Interface, text string, args ...Interface) Interface {
	a := make([]*Message, len(args))
	for i, arg := range args {
		a[i] = vm.CachedMessage(arg)
	}
	result, _ := o.Activate(vm, self, locals, self, vm.IdentMessage(text, a...))
	return result
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	clone := target.Clone()
	if init, proto := vm.GetSlot(target, "init"); proto != nil {
		r, s := init.Activate(vm, clone, locals, proto, vm.IdentMessage("init"))
		if s != NoStop {
			return r, s
		}
	}
	return clone, NoStop
}

// ObjectCloneWithoutInit is an Object method.
//
// cloneWithoutInit creates a new object with empty slots and the cloned object
// as its proto, without checking for an init slot.
func ObjectCloneWithoutInit(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target.Clone(), NoStop
}

// ObjectSetSlot is an Object method.
//
// setSlot sets the value of a slot on this object. It is typically invoked via
// the := operator.
func ObjectSetSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	slot, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		vm.SetSlot(target, slot.String(), v)
	}
	return v, stop
}

// ObjectUpdateSlot is an Object method.
//
// updateSlot raises an exception if the target does not have the given slot,
// and otherwise sets the value of a slot on this object. This is typically
// invoked via the = operator.
func ObjectUpdateSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	slot, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		s := slot.String()
		_, proto := vm.GetSlot(target, s)
		if proto == nil {
			return vm.RaiseExceptionf("slot %s not found", s)
		}
		vm.SetSlot(target, s, v)
	}
	return v, stop
}

// ObjectGetSlot is an Object method.
//
// getSlot gets the value of a slot. The slot is never activated.
func ObjectGetSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	slot, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, _ := vm.GetSlot(target, slot.String())
	if v != nil {
		return v, NoStop
	}
	return vm.Nil, NoStop
}

// ObjectGetLocalSlot is an Object method.
//
// getLocalSlot gets the value of a slot on the receiver, not checking its
// protos. The slot is not activated.
func ObjectGetLocalSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	slot, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, _ := vm.GetLocalSlot(target, slot.String())
	if v != nil {
		return v, NoStop
	}
	return vm.Nil, NoStop
}

// ObjectHasLocalSlot is an Object method.
//
// hasLocalSlot returns whether the object has the given slot name, not
// checking its protos.
func ObjectHasLocalSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	_, ok := vm.GetLocalSlot(target, s.String())
	return vm.IoBool(ok), NoStop
}

// ObjectSlotNames is an Object method.
//
// slotNames returns a list of the names of the slots on this object.
func ObjectSlotNames(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	slots := target.RawSlots()
	names := make([]Interface, 0, len(slots))
	for name := range slots {
		names = append(names, vm.NewString(name))
	}
	target.Unlock()
	return vm.NewList(names...), NoStop
}

// ObjectSlotValues is an Object method.
//
// slotValues returns a list of the values of the slots on this obect.
func ObjectSlotValues(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	slots := target.RawSlots()
	vals := make([]Interface, 0, len(slots))
	for _, val := range slots {
		vals = append(vals, val)
	}
	target.Unlock()
	return vm.NewList(vals...), NoStop
}

// ObjectProtos is an Object method.
//
// protos returns a list of the receiver's protos.
func ObjectProtos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	protos := target.RawProtos()
	v := make([]Interface, len(protos))
	copy(v, protos)
	target.Unlock()
	return vm.NewList(v...), NoStop
}

// ObjectEvalArg is an Object method.
//
// evalArg evaluates and returns its argument. It is typically invoked via the
// empty string slot, i.e. parentheses with no preceding message.
func ObjectEvalArg(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// The original Io implementation has an assertion that there is at least
	// one argument; this will instead return vm.Nil. It wouldn't be difficult
	// to mimic Io's behavior, but ehhh.
	return msg.EvalArgAt(vm, locals, 0)
}

// ObjectEvalArgAndReturnSelf is an Object method.
//
// evalArgAndReturnSelf evaluates its argument and returns this object.
func ObjectEvalArgAndReturnSelf(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	result, stop := msg.EvalArgAt(vm, locals, 0)
	if stop == NoStop {
		return target, NoStop
	}
	return result, stop
}

// ObjectEvalArgAndReturnNil is an Object method.
//
// evalArgAndReturnNil evaluates its argument and returns nil.
func ObjectEvalArgAndReturnNil(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	result, stop := msg.EvalArgAt(vm, locals, 0)
	if stop == NoStop {
		return vm.Nil, NoStop
	}
	return result, stop
}

// ObjectDo is an Object method.
//
// do evaluates its message in the context of the receiver.
func ObjectDo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, stop := msg.EvalArgAt(vm, target, 0)
	if stop != NoStop {
		return r, stop
	}
	return target, NoStop
}

// ObjectLexicalDo is an Object method.
//
// lexicalDo appends the lexical context to the receiver's protos, evaluates
// the message in the context of the receiver, then removes the added proto.
func ObjectLexicalDo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	protos := target.RawProtos()
	n := len(protos)
	protos = append(protos, locals)
	target.RawSetProtos(protos)
	target.Unlock()
	result, stop := msg.EvalArgAt(vm, target, 0)
	target.Lock()
	protos = target.RawProtos()
	copy(protos[n:], protos[n+1:])
	target.RawSetProtos(protos[:len(protos)-1])
	target.Unlock()
	if stop != NoStop {
		return result, stop
	}
	return target, NoStop
}

// ObjectAsString is an Object method.
//
// asString creates a string representation of an object.
func ObjectAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// Non-standard, but if the object is one of the important basic objects,
	// return a more informative name.
	switch target {
	case vm.BaseObject:
		return vm.NewString(fmt.Sprintf("Core Object_%p", target)), NoStop
	case vm.Lobby:
		return vm.NewString(fmt.Sprintf("Lobby_%p", target)), NoStop
	case vm.Core:
		return vm.NewString(fmt.Sprintf("Core_%p", target)), NoStop
	}
	if stringer, ok := target.(fmt.Stringer); ok {
		return vm.NewString(stringer.String()), NoStop
	}
	return vm.NewString(fmt.Sprintf("%T_%p", target, target)), NoStop
}

// ObjectAsGoRepr is an Object method.
//
// asGoRepr returns a string containing a Go-syntax representation of the
// object.
func ObjectAsGoRepr(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(fmt.Sprintf("%#v", target)), NoStop
}

// Compare uses the compare method of x to compare to y. The result should be a
// *Number holding -1, 0, or 1, if the compare method is proper and no
// exception occurs. Any Stop will be returned.
func (vm *VM) Compare(x, y Interface) (Interface, Stop) {
	cmp, proto := vm.GetSlot(x, "compare")
	if proto == nil {
		// No compare method.
		return vm.NewNumber(float64(PtrCompare(x, y))), NoStop
	}
	return cmp.Activate(vm, x, x, proto, vm.IdentMessage("compare", vm.CachedMessage(y)))
}

// ObjectCompare is an Object method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal. The default order is the order of the
// numeric values of the objects' addresses.
func ObjectCompare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	return vm.NewNumber(float64(PtrCompare(target, v))), NoStop
}

// PtrCompare returns a compare value for the pointers of two objects. It
// panics if the value is not a real object.
func PtrCompare(x, y Interface) int {
	a := reflect.ValueOf(x).Pointer()
	b := reflect.ValueOf(y).Pointer()
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
func ObjectLess(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value < 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) < 0), NoStop
}

// ObjectLessOrEqual is an Object method.
//
// x <=(y) returns true if the result of x compare(y) is not 1.
func ObjectLessOrEqual(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value <= 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) <= 0), NoStop
}

// ObjectEqual is an Object method.
//
// x ==(y) returns true if the result of x compare(y) is 0.
func ObjectEqual(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value == 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) == 0), NoStop
}

// ObjectNotEqual is an Object method.
//
// x !=(y) returns true if the result of x compare(y) is not 0.
func ObjectNotEqual(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value != 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) != 0), NoStop
}

// ObjectGreaterOrEqual is an Object method.
//
// x >=(y) returns true if the result of x compare(y) is not -1.
func ObjectGreaterOrEqual(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value >= 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) >= 0), NoStop
}

// ObjectGreater is an Object method.
//
// x >(y) returns true if the result of x compare(y) is 1.
func ObjectGreater(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return c, stop
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value > 0), NoStop
	}
	return vm.IoBool(PtrCompare(target, c) > 0), NoStop
}

// ObjectTry is an Object method.
//
// try executes its message, returning any exception that occurs or nil if none
// does. Any other control flow (continue, break, return) is passed normally.
func ObjectTry(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	switch stop {
	case NoStop: // do nothing
	case ContinueStop, BreakStop, ReturnStop:
		return r, stop
	case ExceptionStop:
		return r, NoStop
	default:
		panic(fmt.Sprintf("try: invalid stop status %#v", stop))
	}
	return vm.Nil, NoStop
}

// ObjectAncestorWithSlot is an Object method.
//
// ancestorWithSlot returns the proto which owns the given slot.
func ObjectAncestorWithSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ss := s.String()
	target.Lock()
	opro := target.RawProtos()
	protos := make([]Interface, len(opro))
	copy(protos, opro)
	target.Unlock()
	for _, p := range protos {
		// TODO: this finds the slot on target if target is in its own protos
		_, proto := vm.GetSlot(p, ss)
		if proto != nil {
			return proto, NoStop
		}
	}
	return vm.Nil, NoStop
}

// ObjectAppendProto is an Object method.
//
// appendProto adds an object as a proto to the object.
func ObjectAppendProto(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	target.Lock()
	target.RawSetProtos(append(target.RawProtos(), v))
	target.Unlock()
	return target, NoStop
}

// ObjectContextWithSlot is an Object method.
//
// contextWithSlot returns the first of the receiver or its protos which
// contains the given slot.
func ObjectContextWithSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	_, proto := vm.GetSlot(target, s.String())
	if proto != nil {
		return proto, NoStop
	}
	return vm.Nil, NoStop
}

// ObjectDoFile is an Object method.
//
// doFile executes the file at the given path in the context of the receiver.
func ObjectDoFile(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	f, ferr := os.Open(s.String())
	if ferr != nil {
		return vm.IoError(ferr)
	}
	m, err := vm.Parse(f, f.Name())
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return vm.DoMessage(m, target)
}

// ObjectDoMessage is an Object method.
//
// doMessage sends the message to the receiver.
func ObjectDoMessage(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m, err, stop := msg.MessageArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ctxt := target
	if msg.ArgCount() > 1 {
		ctxt, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return ctxt, stop
		}
	}
	return m.Send(vm, target, ctxt)
}

// ObjectDoString is an Object method.
//
// doString executes the string in the context of the receiver.
func ObjectDoString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	src := strings.NewReader(s.String())
	label := "doString"
	if msg.ArgCount() > 1 {
		l, err, stop := msg.StringArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		label = l.String()
	}
	m, err := vm.Parse(src, label)
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(m); err != nil {
		return err.Raise()
	}
	return m.Eval(vm, target)
}

// ObjectForeachSlot is a Object method.
//
// foreachSlot performs a loop on each slot of an object.
func ObjectForeachSlot(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseException("foreachSlot requires 2 or 3 args")
	}
	// To be safe in a parallel world, we need to make a copy of the target's
	// slots while holding its lock.
	target.Lock()
	ts := target.RawSlots()
	slots := make(Slots, len(ts))
	for k, v := range ts {
		slots[k] = v
	}
	target.Unlock()
	for k, v := range slots {
		vm.SetSlot(locals, vn, v)
		if hkn {
			vm.SetSlot(locals, kn, vm.NewString(k))
		}
		result, control = ev.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, control
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
	return result, NoStop
}

// ObjectIsIdenticalTo is an Object method.
//
// isIdenticalTo returns whether the object is the same as the argument.
func ObjectIsIdenticalTo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	return vm.IoBool(target == r), NoStop
}

// ObjectIsKindOf is an Object method.
//
// isKindOf returns whether the object is the argument or has the argument
// among its protos.
func ObjectIsKindOf(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	return vm.IoBool(vm.IsKindOf(target, r)), NoStop
}

// ObjectMessage is an Object method.
//
// message returns the argument message.
func ObjectMessage(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	if msg.ArgCount() > 0 {
		return msg.ArgAt(0), NoStop
	}
	return vm.Nil, NoStop
}

// ObjectPerform is an Object method.
//
// perform executes the method named by the first argument using the remaining
// argument messages as arguments to the method.
func ObjectPerform(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	switch a := r.(type) {
	case *Sequence:
		// String name, arguments are messages.
		name := a.String()
		m := vm.IdentMessage(name, msg.Args[1:]...)
		for i, arg := range m.Args {
			m.Args[i] = arg.DeepCopy()
		}
		return vm.Perform(target, locals, m)
	case *Message:
		// Message argument, which provides both the name and the args.
		if msg.ArgCount() > 1 {
			return vm.RaiseException("perform takes a single argument when using a Message as an argument")
		}
		return vm.Perform(target, locals, a)
	}
	return vm.RaiseException("argument 0 to perform must be Sequence or Message, not " + vm.TypeName(r))
}

// ObjectPerformWithArgList is an Object method.
//
// performWithArgList activates the given method with arguments given in the
// second argument as a list.
func ObjectPerformWithArgList(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	l, err, stop := msg.ListArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	name := s.String()
	m := vm.IdentMessage(name)
	for _, arg := range l.Value {
		m.Args = append(m.Args, vm.CachedMessage(arg))
	}
	return vm.Perform(target, locals, m)
}

// ObjectPrependProto is an Object method.
//
// prependProto adds a new proto as the first in the object's protos.
func ObjectPrependProto(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return p, stop
	}
	target.Lock()
	protos := append(target.RawProtos(), p)
	copy(protos[1:], protos)
	protos[0] = p
	target.Unlock()
	return target, NoStop
}

// ObjectRemoveAllProtos is an Object method.
//
// removeAllProtos removes all protos from the object.
func ObjectRemoveAllProtos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	target.RawSetProtos([]Interface{})
	target.Unlock()
	return target, NoStop
}

// ObjectRemoveAllSlots is an Object method.
//
// removeAllSlots removes all slots from the object.
func ObjectRemoveAllSlots(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.Lock()
	target.RawSetSlots(Slots{})
	target.Unlock()
	return target, NoStop
}

// ObjectRemoveProto is an Object method.
//
// removeProto removes the given object from the object's protos.
func ObjectRemoveProto(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return p, stop
	}
	target.Lock()
	protos := target.RawProtos()
	n := make([]Interface, 0, len(protos))
	for _, proto := range protos {
		if proto != p {
			n = append(n, proto)
		}
	}
	target.RawSetProtos(n)
	target.Unlock()
	return target, NoStop
}

// ObjectRemoveSlot is an Object method.
//
// removeSlot removes the given slot from the object.
func ObjectRemoveSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	vm.RemoveSlot(target, s.String())
	return target, NoStop
}

// ObjectSetProto is an Object method.
//
// setProto sets the object's proto list to have only the given object.
func ObjectSetProto(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return p, stop
	}
	target.Lock()
	target.RawSetProtos(append(target.RawProtos()[:0], p))
	target.Unlock()
	return target, NoStop
}

// ObjectSetProtos is an Object method.
//
// setProtos sets the object's protos to the objects in the given list.
func ObjectSetProtos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l, err, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	target.Lock()
	target.RawSetProtos(append(target.RawProtos()[:0], l.Value...))
	target.Unlock()
	return target, NoStop
}

// ObjectShallowCopy is an Object method.
//
// shallowCopy creates a new object with the receiver's slots and protos.
func ObjectShallowCopy(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	o, ok := target.(*Object)
	if !ok {
		return vm.RaiseException("shallowCopy cannot be used on primitives")
	}
	o.Lock()
	defer o.Unlock()
	n := &Object{Slots: make(Slots, len(o.Slots)), Protos: make([]Interface, len(o.Protos))}
	// The shallow copy in Io doesn't actually copy the protos...
	copy(n.Protos, o.Protos)
	for slot, value := range o.Slots {
		n.Slots[slot] = value
	}
	return n, NoStop
}

// ObjectThisContext is an Object method.
//
// thisContext returns the current slot context, which is the receiver.
func ObjectThisContext(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target, NoStop
}

// ObjectThisLocalContext is an Object method.
//
// thisLocalContext returns the current locals object.
func ObjectThisLocalContext(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return locals, NoStop
}

// ObjectThisMessage is an Object method.
//
// thisMessage returns the message which activated this method, which is likely
// to be thisMessage.
func ObjectThisMessage(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return msg, NoStop
}

// ObjectUniqueId is an Object method.
//
// uniqueId returns a string representation of the object's address.
func ObjectUniqueId(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	u := reflect.ValueOf(target).Pointer()
	return vm.NewString(fmt.Sprintf("%#x", u)), NoStop
}

// ObjectWait is an Object method.
//
// wait pauses execution in the current coroutine for the given number of
// seconds. The coroutine must be re-scheduled after waking up, which can
// result in the actual wait time being longer by an unpredictable amount.
func ObjectWait(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	time.Sleep(time.Duration(v.Value * float64(time.Second)))
	return vm.Nil, NoStop
}
