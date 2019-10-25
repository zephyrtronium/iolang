package iolang

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zephyrtronium/contains"
)

// Slots holds the set of messages to which an object responds.
type Slots = map[string]*Object

// Object is the basic type of Io. Everything is an Object.
type Object struct {
	// Mutex is a lock which must be held when accessing the slots or protos of
	// the object, or the value of the object if it is or may be mutable.
	sync.Mutex

	// Slots is the set of messages to which this object responds.
	Slots Slots
	// Protos are the set of objects to which messages are forwarded, in
	// depth-first order without duplicates, when this object cannot respond.
	Protos []*Object

	// Value is the object's type-specific primitive value.
	Value interface{}
	// Tag is the type indicator of the object.
	Tag Tag

	// protoSet is the set of protos checked during GetSlot.
	protoSet contains.Set
	// protoStack is the stack of protos to check during GetSlot.
	protoStack []*Object

	// TODO: I think the current implementation of UniqueID is wrong, since the
	// GC can move objects. Better would be something like a goroutine serving
	// uintptrs for each new object. Maybe make a new VM method to create an
	// object with a given Value and Tag.
}

// Tag is a type indicator for iolang objects. Tag values must be comparable.
// Tags for different types must not be equal, meaning they must have different
// underlying types or different values otherwise.
type Tag interface {
	// Activate activates an object that has this tag. The self argument is the
	// object which has this tag, target is the object that received the
	// message, and context is the object that actually had the slot.
	Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object
	// CloneValue takes the Value of an existing object and returns the Value
	// of a clone of that object.
	CloneValue(value interface{}) interface{}

	// String returns the name of the type associated with this tag.
	String() string
}

// Activate activates the object.
func (o *Object) Activate(vm *VM, target, locals, context *Object, msg *Message) *Object {
	if o.Tag == nil {
		// Basic object. Check the isActivatable slot.
		ok, proto := o.GetSlot("isActivatable")
		// We can't use vm.AsBool even though it's one of the few situations
		// where we'd want to, because it will attempt to activate the isTrue
		// slot, which is typically a plain object, which will activate this
		// method and recurse infinitely.
		if proto == nil || ok == vm.False || ok == vm.Nil {
			return o
		}
		act, proto := o.GetSlot("activate")
		if proto != nil {
			return act.Activate(vm, target, locals, context, msg)
		}
		return o
	}
	return o.Tag.Activate(vm, o, target, locals, context, msg)
}

// Clone returns a new object with empty slots and this object as its only
// proto. The clone's tag is the same as its parent's, and its primitive value
// is produced by the tag's CloneValue method. Returns nil if the object's tag
// is nil.
func (o *Object) Clone() *Object {
	var v interface{}
	if o.Tag != nil {
		o.Lock()
		v = o.Tag.CloneValue(o.Value)
		o.Unlock()
	}
	return &Object{
		Protos: []*Object{o},
		Value:  v,
		Tag:    o.Tag,
	}
}

// GetSlot checks the object and its ancestors in depth-first order without
// cycles for a slot, returning the slot value and the proto which had it.
// proto is nil if and only if the slot was not found. This method acquires the
// object's lock, as well as the lock of each ancestor in turn.
func (o *Object) GetSlot(slot string) (value, proto *Object) {
	if o == nil {
		return nil, nil
	}
	// An object can have slots checked from multiple coroutines, so we need to
	// hold the lock the entire time we might be modifying its protoStack and
	// protoSet. This means we need to check its local slots outside the main
	// loop, which might make some programs faster anyway.
	o.Lock()
	if o.Slots != nil {
		if r, ok := o.Slots[slot]; ok {
			o.Unlock()
			return r, o
		}
	}
	o.protoSet.Add(o.UniqueID())
	for i := len(o.Protos) - 1; i >= 0; i-- {
		if p := o.Protos[i]; o.protoSet.Add(p.UniqueID()) {
			o.protoStack = append(o.protoStack, p)
		}
	}
	// Recursion is easy, but it can cause deadlocks, since we need to hold
	// each proto's lock to check its respective protos. This stack-based
	// approach is a bit messy, but it allows us to hold each ancestor's lock
	// only while grabbing its (current) protos.
	for len(o.protoStack) > 0 {
		rp := o.protoStack[len(o.protoStack)-1] // grab the top
		rp.Lock()
		if rp.Slots != nil {
			if r, ok := rp.Slots[slot]; ok {
				rp.Unlock()
				o.protoSet.Reset()
				o.protoStack = o.protoStack[:0]
				o.Unlock()
				return r, rp
			}
		}
		o.protoStack = o.protoStack[:len(o.protoStack)-1] // actually pop
		for i := len(rp.Protos) - 1; i >= 0; i-- {
			if p := rp.Protos[i]; o.protoSet.Add(p.UniqueID()) {
				o.protoStack = append(o.protoStack, p)
			}
		}
		rp.Unlock()
	}
	o.protoSet.Reset()
	o.Unlock()
	return nil, nil
}

// GetLocalSlot checks only the object's own slots for a slot.
func (o *Object) GetLocalSlot(slot string) (value *Object, ok bool) {
	if o == nil {
		return nil, false
	}
	o.Lock()
	if o.Slots == nil {
		o.Unlock()
		return nil, false
	}
	value, ok = o.Slots[slot]
	o.Unlock()
	return value, ok
}

// SetSlot sets a local slot's value.
func (o *Object) SetSlot(slot string, value *Object) {
	o.Lock()
	if o.Slots == nil {
		o.Slots = Slots{}
	}
	o.Slots[slot] = value
	o.Unlock()
}

// SetSlots sets multiple slots more efficiently than using SetSlot for each.
func (o *Object) SetSlots(slots Slots) {
	o.Lock()
	if o.Slots == nil {
		o.Slots = Slots{}
	}
	for slot, value := range slots {
		o.Slots[slot] = value
	}
	o.Unlock()
}

// RemoveSlot removes slots from the object's local slots, if they are present.
func (o *Object) RemoveSlot(slots ...string) {
	o.Lock()
	if o.Slots != nil {
		for _, slot := range slots {
			delete(o.Slots, slot)
		}
	}
	o.Unlock()
}

// IsKindOf evaluates whether the object has kind as any of its ancestors, or
// is itself kind.
func (o *Object) IsKindOf(kind *Object) bool {
	if o == nil {
		return false
	}
	// Unlike in GetSlot, we aren't in the hot path for message passing, so we
	// can behave a bit more simply. In particular, we can use our own set and
	// stack, so we don't have to hold the object's lock the whole time, and we
	// can traverse the graph in any order instead of specifically depth-first.
	protos := []*Object{o}
	set := contains.Set{}
	set.Add(o.UniqueID())
	for len(protos) > 0 {
		proto := protos[len(protos)-1]
		protos = protos[:len(protos)-1]
		if proto == kind {
			return true
		}
		proto.Lock()
		for _, p := range proto.Protos {
			if set.Add(p.UniqueID()) {
				protos = append(protos, p)
			}
		}
		proto.Unlock()
	}
	return false
}

// BasicTag is a special Tag type for basic primitive types which do not have
// special activation and whose clones have values that are shallow copies of
// their parents.
type BasicTag string

// Activate returns self.
func (t BasicTag) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

// CloneValue returns value.
func (t BasicTag) CloneValue(value interface{}) interface{} {
	return value
}

// String returns the receiver.
func (t BasicTag) String() string {
	return string(t)
}

// initObject sets up the "base" object that is the first proto of all other
// built-in types.
func (vm *VM) initObject() {
	vm.BaseObject.Protos = []*Object{vm.Lobby}
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
		"uniqueId":             vm.NewCFunction(ObjectUniqueID, nil),
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
	vm.Core.SetSlot("Object", vm.BaseObject)
}

// ObjectWith creates a new object with the given slots and with the VM's
// Core Object as its proto.
func (vm *VM) ObjectWith(slots Slots) *Object {
	return &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
	}
}

// TypeName gets the name of the type of an object by activating its type slot.
// If there is no such slot, then its tag's name will be returned; if its tag
// is nil, then its name is Object.
func (vm *VM) TypeName(o *Object) string {
	if typ, proto := o.GetSlot("type"); proto != nil {
		return vm.AsString(typ)
	}
	if o.Tag != nil {
		return o.Tag.String()
	}
	return "Object"
}

// SimpleActivate activates an object using the identifier message named with
// text and with the given arguments. Any control flow signals sent while
// activating the object are consumed and ignored. (Normally, the value paired
// with a stop is the returned result from any CFunction; this behavior should
// only make a difference if an Exception value is the result, or if the
// control flow is sent from a different coroutine.)
func (vm *VM) SimpleActivate(o, self, locals *Object, text string, args ...*Object) *Object {
	a := make([]*Message, len(args))
	for i, arg := range args {
		a[i] = vm.CachedMessage(arg)
	}
	result := o.Activate(vm, self, locals, self, vm.IdentMessage(text, a...))
	select {
	case <-vm.Control: // do nothing
	default: // do nothing
	}
	return result
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals *Object, msg *Message) *Object {
	clone := target.Clone()
	if init, proto := target.GetSlot("init"); proto != nil {
		init.Activate(vm, clone, locals, proto, vm.IdentMessage("init"))
	}
	return clone
}

// ObjectCloneWithoutInit is an Object method.
//
// cloneWithoutInit creates a new object with empty slots and the cloned object
// as its proto, without checking for an init slot.
func ObjectCloneWithoutInit(vm *VM, target, locals *Object, msg *Message) *Object {
	return target.Clone()
}

// ObjectSetSlot is an Object method.
//
// setSlot sets the value of a slot on this object. It is typically invoked via
// the := operator.
func ObjectSetSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		target.SetSlot(slot, v)
	}
	return vm.Stop(v, stop)
}

// ObjectUpdateSlot is an Object method.
//
// updateSlot raises an exception if the target does not have the given slot,
// and otherwise sets the value of a slot on this object. This is typically
// invoked via the = operator.
func ObjectUpdateSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		_, proto := target.GetSlot(slot)
		if proto == nil {
			return vm.RaiseExceptionf("slot %s not found", slot)
		}
		target.SetSlot(slot, v)
	}
	return vm.Stop(v, stop)
}

// ObjectGetSlot is an Object method.
//
// getSlot gets the value of a slot. The slot is never activated.
func ObjectGetSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v, _ := target.GetSlot(slot)
	return v
}

// ObjectGetLocalSlot is an Object method.
//
// getLocalSlot gets the value of a slot on the receiver, not checking its
// protos. The slot is not activated.
func ObjectGetLocalSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v, _ := target.GetLocalSlot(slot)
	return v
}

// ObjectHasLocalSlot is an Object method.
//
// hasLocalSlot returns whether the object has the given slot name, not
// checking its protos.
func ObjectHasLocalSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	_, ok := target.GetLocalSlot(slot)
	return vm.IoBool(ok)
}

// ObjectSlotNames is an Object method.
//
// slotNames returns a list of the names of the slots on this object.
func ObjectSlotNames(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	names := make([]*Object, 0, len(target.Slots))
	for name := range target.Slots {
		names = append(names, vm.NewString(name))
	}
	target.Unlock()
	return vm.NewList(names...)
}

// ObjectSlotValues is an Object method.
//
// slotValues returns a list of the values of the slots on this obect.
func ObjectSlotValues(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	vals := make([]*Object, 0, len(target.Slots))
	for _, val := range target.Slots {
		vals = append(vals, val)
	}
	target.Unlock()
	return vm.NewList(vals...)
}

// ObjectProtos is an Object method.
//
// protos returns a list of the receiver's protos.
func ObjectProtos(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	v := make([]*Object, len(target.Protos))
	copy(v, target.Protos)
	target.Unlock()
	return vm.NewList(v...)
}

// ObjectEvalArg is an Object method.
//
// evalArg evaluates and returns its argument. It is typically invoked via the
// empty string slot, i.e. parentheses with no preceding message.
func ObjectEvalArg(vm *VM, target, locals *Object, msg *Message) *Object {
	// The original Io implementation has an assertion that there is at least
	// one argument; this will instead return vm.Nil. It wouldn't be difficult
	// to mimic Io's behavior, but ehhh.
	return vm.Stop(msg.EvalArgAt(vm, locals, 0))
}

// ObjectEvalArgAndReturnSelf is an Object method.
//
// evalArgAndReturnSelf evaluates its argument and returns this object.
func ObjectEvalArgAndReturnSelf(vm *VM, target, locals *Object, msg *Message) *Object {
	result, stop := msg.EvalArgAt(vm, locals, 0)
	if stop == NoStop {
		return target
	}
	return vm.Stop(result, stop)
}

// ObjectEvalArgAndReturnNil is an Object method.
//
// evalArgAndReturnNil evaluates its argument and returns nil.
func ObjectEvalArgAndReturnNil(vm *VM, target, locals *Object, msg *Message) *Object {
	result, stop := msg.EvalArgAt(vm, locals, 0)
	if stop == NoStop {
		return vm.Nil
	}
	return vm.Stop(result, stop)
}

// ObjectDo is an Object method.
//
// do evaluates its message in the context of the receiver.
func ObjectDo(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, target, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	return target
}

// ObjectLexicalDo is an Object method.
//
// lexicalDo appends the lexical context to the receiver's protos, evaluates
// the message in the context of the receiver, then removes the added proto.
func ObjectLexicalDo(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	n := len(target.Protos)
	target.Protos = append(target.Protos, locals)
	target.Unlock()
	result, stop := msg.EvalArgAt(vm, target, 0)
	target.Lock()
	copy(target.Protos[n:], target.Protos[n+1:])
	target.Protos = target.Protos[:len(target.Protos)-1]
	target.Unlock()
	if stop != NoStop {
		return vm.Stop(result, stop)
	}
	return target
}

// ObjectAsString is an Object method.
//
// asString creates a string representation of an object.
func ObjectAsString(vm *VM, target, locals *Object, msg *Message) *Object {
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
	if stringer, ok := target.Value.(fmt.Stringer); ok {
		return vm.NewString(stringer.String())
	}
	return vm.NewString(fmt.Sprintf("%s_%p", vm.TypeName(target), target))
}

// ObjectAsGoRepr is an Object method.
//
// asGoRepr returns a string containing a Go-syntax representation of the
// object's value.
func ObjectAsGoRepr(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(fmt.Sprintf("%#v", target.Value))
}

// Compare uses the compare method of x to compare to y. The result should be a
// *Number holding -1, 0, or 1, if the compare method is proper and no
// exception occurs. Any Stop will be returned.
func (vm *VM) Compare(x, y *Object) (*Object, Stop) {
	cmp, proto := x.GetSlot("compare")
	if proto == nil {
		// No compare method.
		return vm.NewNumber(float64(PtrCompare(x, y))), NoStop
	}
	return vm.Status(cmp.Activate(vm, x, x, proto, vm.IdentMessage("compare", vm.CachedMessage(y))))
}

// ObjectCompare is an Object method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal. The default order is the order of the
// numeric values of the objects' addresses.
func ObjectCompare(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	return vm.NewNumber(float64(PtrCompare(target, v)))
}

// PtrCompare returns a compare value for the pointers of two objects. It
// panics if the value is not a real object.
func PtrCompare(x, y *Object) int {
	a := x.UniqueID()
	b := y.UniqueID()
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
func ObjectLess(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n < 0)
	}
	return vm.IoBool(PtrCompare(target, c) < 0)
}

// ObjectLessOrEqual is an Object method.
//
// x <=(y) returns true if the result of x compare(y) is not 1.
func ObjectLessOrEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n <= 0)
	}
	return vm.IoBool(PtrCompare(target, c) <= 0)
}

// ObjectEqual is an Object method.
//
// x ==(y) returns true if the result of x compare(y) is 0.
func ObjectEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n == 0)
	}
	return vm.IoBool(PtrCompare(target, c) == 0)
}

// ObjectNotEqual is an Object method.
//
// x !=(y) returns true if the result of x compare(y) is not 0.
func ObjectNotEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n != 0)
	}
	return vm.IoBool(PtrCompare(target, c) != 0)
}

// ObjectGreaterOrEqual is an Object method.
//
// x >=(y) returns true if the result of x compare(y) is not -1.
func ObjectGreaterOrEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n >= 0)
	}
	return vm.IoBool(PtrCompare(target, c) >= 0)
}

// ObjectGreater is an Object method.
//
// x >(y) returns true if the result of x compare(y) is 1.
func ObjectGreater(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if n, ok := c.Value.(float64); ok {
		return vm.IoBool(n > 0)
	}
	return vm.IoBool(PtrCompare(target, c) > 0)
}

// ObjectTry is an Object method.
//
// try executes its message, returning any exception that occurs or nil if none
// does. Any other control flow (continue, break, return) is passed normally.
func ObjectTry(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	switch stop {
	case NoStop: // do nothing
	case ContinueStop, BreakStop, ReturnStop:
		return vm.Stop(r, stop)
	case ExceptionStop:
		return r
	default:
		panic(fmt.Sprintf("try: invalid stop status %v", stop))
	}
	return vm.Nil
}

// ObjectAncestorWithSlot is an Object method.
//
// ancestorWithSlot returns the proto which owns the given slot.
func ObjectAncestorWithSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	protos := make([]*Object, len(target.Protos))
	copy(protos, target.Protos)
	target.Unlock()
	for _, p := range protos {
		// TODO: this finds the slot on target if target is in its own protos
		_, proto := p.GetSlot(slot)
		if proto != nil {
			return proto
		}
	}
	return vm.Nil
}

// ObjectAppendProto is an Object method.
//
// appendProto adds an object as a proto to the object.
func ObjectAppendProto(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	target.Lock()
	target.Protos = append(target.Protos, v)
	target.Unlock()
	return target
}

// ObjectContextWithSlot is an Object method.
//
// contextWithSlot returns the first of the receiver or its protos which
// contains the given slot.
func ObjectContextWithSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	_, proto := target.GetSlot(slot)
	return proto
}

// ObjectDoFile is an Object method.
//
// doFile executes the file at the given path in the context of the receiver.
func ObjectDoFile(vm *VM, target, locals *Object, msg *Message) *Object {
	nm, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	f, err := os.Open(nm)
	if err != nil {
		return vm.IoError(err)
	}
	m, err := vm.Parse(f, f.Name())
	if err != nil {
		return vm.IoError(err)
	}
	return vm.Stop(vm.DoMessage(m, target))
}

// ObjectDoMessage is an Object method.
//
// doMessage sends the message to the receiver.
func ObjectDoMessage(vm *VM, target, locals *Object, msg *Message) *Object {
	m, exc, stop := msg.MessageArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	ctxt := target
	if msg.ArgCount() > 1 {
		ctxt, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(ctxt, stop)
		}
	}
	return vm.Stop(m.Send(vm, target, ctxt))
}

// ObjectDoString is an Object method.
//
// doString executes the string in the context of the receiver.
func ObjectDoString(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	src := strings.NewReader(s)
	label := "doString"
	if msg.ArgCount() > 1 {
		l, exc, stop := msg.StringArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		label = l
	}
	m, err := vm.Parse(src, label)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.Stop(m.Eval(vm, target))
}

// ObjectForeachSlot is a Object method.
//
// foreachSlot performs a loop on each slot of an object.
func ObjectForeachSlot(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseExceptionf("foreachSlot requires 2 or 3 args")
	}
	// To be safe in a parallel world, we need to make a copy of the target's
	// slots while holding its lock.
	target.Lock()
	slots := make(Slots, len(target.Slots))
	for k, v := range target.Slots {
		slots[k] = v
	}
	target.Unlock()
	var control Stop
	for k, v := range slots {
		locals.SetSlot(vn, v)
		if hkn {
			locals.SetSlot(kn, vm.NewString(k))
		}
		result, control = ev.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
	return result
}

// ObjectIsIdenticalTo is an Object method.
//
// isIdenticalTo returns whether the object is the same as the argument.
func ObjectIsIdenticalTo(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	return vm.IoBool(target == r)
}

// ObjectIsKindOf is an Object method.
//
// isKindOf returns whether the object is the argument or has the argument
// among its protos.
func ObjectIsKindOf(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	return vm.IoBool(target.IsKindOf(r))
}

// ObjectMessage is an Object method.
//
// message returns the argument message.
func ObjectMessage(vm *VM, target, locals *Object, msg *Message) *Object {
	if msg.ArgCount() > 0 {
		return vm.MessageObject(msg.ArgAt(0))
	}
	return vm.Nil
}

// ObjectPerform is an Object method.
//
// perform executes the method named by the first argument using the remaining
// argument messages as arguments to the method.
func ObjectPerform(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	r.Lock()
	switch a := r.Value.(type) {
	case Sequence:
		// String name, arguments are messages.
		name := a.String()
		r.Unlock()
		m := vm.IdentMessage(name, msg.Args[1:]...)
		for i, arg := range m.Args {
			m.Args[i] = arg.DeepCopy()
		}
		return vm.Stop(vm.Perform(target, locals, m))
	case *Message:
		// Message argument, which provides both the name and the args.
		r.Unlock()
		if msg.ArgCount() > 1 {
			return vm.RaiseExceptionf("perform takes a single argument when using a Message as an argument")
		}
		return vm.Stop(vm.Perform(target, locals, a))
	}
	return vm.RaiseExceptionf("argument 0 to perform must be Sequence or Message, not %s", vm.TypeName(r))
}

// ObjectPerformWithArgList is an Object method.
//
// performWithArgList activates the given method with arguments given in the
// second argument as a list.
func ObjectPerformWithArgList(vm *VM, target, locals *Object, msg *Message) *Object {
	name, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	l, obj, stop := msg.ListArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	m := vm.IdentMessage(name)
	obj.Lock()
	for _, arg := range l {
		m.Args = append(m.Args, vm.CachedMessage(arg))
	}
	obj.Unlock()
	return vm.Stop(vm.Perform(target, locals, m))
}

// ObjectPrependProto is an Object method.
//
// prependProto adds a new proto as the first in the object's protos.
func ObjectPrependProto(vm *VM, target, locals *Object, msg *Message) *Object {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(p, stop)
	}
	target.Lock()
	protos := append(target.Protos, p)
	copy(protos[1:], protos)
	protos[0] = p
	target.Protos = protos
	target.Unlock()
	return target
}

// ObjectRemoveAllProtos is an Object method.
//
// removeAllProtos removes all protos from the object.
func ObjectRemoveAllProtos(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	target.Protos = []*Object{}
	target.Unlock()
	return target
}

// ObjectRemoveAllSlots is an Object method.
//
// removeAllSlots removes all slots from the object.
func ObjectRemoveAllSlots(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	target.Slots = Slots{}
	target.Unlock()
	return target
}

// ObjectRemoveProto is an Object method.
//
// removeProto removes the given object from the object's protos.
func ObjectRemoveProto(vm *VM, target, locals *Object, msg *Message) *Object {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(p, stop)
	}
	target.Lock()
	n := make([]*Object, 0, len(target.Protos))
	for _, proto := range target.Protos {
		if proto != p {
			n = append(n, proto)
		}
	}
	target.Protos = n
	target.Unlock()
	return target
}

// ObjectRemoveSlot is an Object method.
//
// removeSlot removes the given slot from the object.
func ObjectRemoveSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.RemoveSlot(slot)
	return target
}

// ObjectSetProto is an Object method.
//
// setProto sets the object's proto list to have only the given object.
func ObjectSetProto(vm *VM, target, locals *Object, msg *Message) *Object {
	p, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(p, stop)
	}
	target.Lock()
	target.Protos = append(target.Protos[:0], p)
	target.Unlock()
	return target
}

// ObjectSetProtos is an Object method.
//
// setProtos sets the object's protos to the objects in the given list.
func ObjectSetProtos(vm *VM, target, locals *Object, msg *Message) *Object {
	l, obj, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	target.Lock()
	obj.Lock()
	target.Protos = append(target.Protos[:0], l...)
	target.Unlock()
	obj.Unlock()
	return target
}

// ObjectShallowCopy is an Object method.
//
// shallowCopy creates a new object with the receiver's slots and protos.
func ObjectShallowCopy(vm *VM, target, locals *Object, msg *Message) *Object {
	if target.Tag != nil {
		return vm.RaiseExceptionf("shallowCopy cannot be used on primitives")
	}
	target.Lock()
	defer target.Unlock()
	n := &Object{Slots: make(Slots, len(target.Slots)), Protos: make([]*Object, len(target.Protos))}
	// The shallow copy in Io doesn't actually copy the protos...
	copy(n.Protos, target.Protos)
	for slot, value := range target.Slots {
		n.Slots[slot] = value
	}
	return n
}

// ObjectThisContext is an Object method.
//
// thisContext returns the current slot context, which is the receiver.
func ObjectThisContext(vm *VM, target, locals *Object, msg *Message) *Object {
	return target
}

// ObjectThisLocalContext is an Object method.
//
// thisLocalContext returns the current locals object.
func ObjectThisLocalContext(vm *VM, target, locals *Object, msg *Message) *Object {
	return locals
}

// ObjectThisMessage is an Object method.
//
// thisMessage returns the message which activated this method, which is likely
// to be thisMessage.
func ObjectThisMessage(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.MessageObject(msg)
}

// ObjectUniqueID is an Object method.
//
// uniqueId returns a string representation of the object's address.
func ObjectUniqueID(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(fmt.Sprintf("%#x", target.UniqueID()))
}

// ObjectWait is an Object method.
//
// wait pauses execution in the current coroutine for the given number of
// seconds. The coroutine must be re-scheduled after waking up, which can
// result in the actual wait time being longer by an unpredictable amount.
func ObjectWait(vm *VM, target, locals *Object, msg *Message) *Object {
	v, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	time.Sleep(time.Duration(v * float64(time.Second)))
	return vm.Nil
}
