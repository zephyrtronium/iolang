package internal

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zephyrtronium/contains"
)

// Object is the basic type of Io. Everything is an Object.
//
// Always use NewObject, ObjectWith, or a type-specific constructor to obtain
// new objects. Creating objects directly will result in arbitrary failures.
type Object struct {
	// slots is the set of messages to which this object responds.
	slots actualSlots
	// protos is the head of this object's protos list. If protos.p is nil,
	// then the object has no protos.
	protos protoLink

	// Mutex is a lock which must be held when accessing the value of the
	// object if it is or may be mutable.
	sync.Mutex
	// Value is the object's type-specific primitive value.
	Value interface{}
	// tag is the type indicator of the object.
	tag Tag

	// id is the object's unique ID.
	id uintptr
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
	if o.Tag() == nil {
		// Basic object. Check the isActivatable slot.
		ok, proto := vm.GetSlot(o, "isActivatable")
		// We can't use vm.AsBool even though it's one of the few situations
		// where we'd want to, because it will attempt to activate the
		// asBoolean slot, which is typically a plain object, which will
		// activate this method and recurse infinitely.
		if proto == nil || ok == vm.False || ok == vm.Nil {
			return o
		}
		act, proto := vm.GetSlot(o, "activate")
		if proto != nil {
			return act.Activate(vm, target, locals, context, msg)
		}
		return o
	}
	return o.Tag().Activate(vm, o, target, locals, context, msg)
}

// Clone returns a new object with empty slots and this object as its only
// proto. The clone's tag is the same as its parent's, and its primitive value
// is produced by the tag's CloneValue method. Returns nil if the object's tag
// is nil.
func (o *Object) Clone() *Object {
	var v interface{}
	if o.Tag() != nil {
		o.Lock()
		v = o.Tag().CloneValue(o.Value)
		o.Unlock()
	}
	return &Object{
		protos: protoLink{p: o},
		Value:  v,
		tag:    o.Tag(),
		id:     nextObject(),
	}
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
		for _, p := range proto.Protos() {
			if set.Add(p.UniqueID()) {
				protos = append(protos, p)
			}
		}
	}
	return false
}

// Tag returns the object's type indicator.
func (o *Object) Tag() Tag {
	return o.tag
}

// UniqueID returns the object's unique ID.
func (o *Object) UniqueID() uintptr {
	return o.id
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

// objcounter is the global counter for object IDs. All accesses to this must
// be atomic.
var objcounter uintptr

// nextObject increments the object counter and returns its value as a unique
// ID for a new object.
func nextObject() uintptr {
	return atomic.AddUintptr(&objcounter, 1)
}

// initObject sets up the "base" object that is the first proto of all other
// built-in types.
func (vm *VM) initObject() {
	vm.BaseObject.SetProtos(vm.Lobby)
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
		"block":                vm.NewCFunction(ObjectBlock, nil), // block.go
		"break":                vm.NewCFunction(ObjectBreak, nil), // control.go
		"clone":                vm.NewCFunction(ObjectClone, nil),
		"cloneWithoutInit":     vm.NewCFunction(ObjectCloneWithoutInit, nil),
		"compare":              vm.NewCFunction(ObjectCompare, nil),
		"contextWithSlot":      vm.NewCFunction(ObjectContextWithSlot, nil),
		"continue":             vm.NewCFunction(ObjectContinue, nil), // control.go
		"do":                   vm.NewCFunction(ObjectDo, nil),
		"doFile":               vm.NewCFunction(ObjectDoFile, nil),
		"doMessage":            vm.NewCFunction(ObjectDoMessage, nil),
		"doString":             vm.NewCFunction(ObjectDoString, nil),
		"evalArgAndReturnNil":  vm.NewCFunction(ObjectEvalArgAndReturnNil, nil),
		"evalArgAndReturnSelf": vm.NewCFunction(ObjectEvalArgAndReturnSelf, nil),
		"for":                  vm.NewCFunction(ObjectFor, nil), // control.go
		"foreachSlot":          vm.NewCFunction(ObjectForeachSlot, nil),
		"getLocalSlot":         vm.NewCFunction(ObjectGetLocalSlot, nil),
		"getSlot":              vm.NewCFunction(ObjectGetSlot, nil),
		"hasLocalSlot":         vm.NewCFunction(ObjectHasLocalSlot, nil),
		"if":                   vm.NewCFunction(ObjectIf, nil), // control.go
		"isError":              vm.False,
		"isIdenticalTo":        vm.NewCFunction(ObjectIsIdenticalTo, nil),
		"isKindOf":             vm.NewCFunction(ObjectIsKindOf, nil),
		"isNil":                vm.False,
		"isTrue":               vm.True,
		"lexicalDo":            vm.NewCFunction(ObjectLexicalDo, nil),
		"loop":                 vm.NewCFunction(ObjectLoop, nil), // control.go
		"message":              vm.NewCFunction(ObjectMessage, nil),
		"method":               vm.NewCFunction(ObjectMethod, nil), // block.go
		"not":                  vm.Nil,
		"or":                   vm.True,
		"perform":              vm.NewCFunction(ObjectPerform, nil),
		"performWithArgList":   vm.NewCFunction(ObjectPerformWithArgList, nil),
		"prependProto":         vm.NewCFunction(ObjectPrependProto, nil),
		"print":                vm.NewCFunction(ObjectPrint, nil),
		"protos":               vm.NewCFunction(ObjectProtos, nil),
		"removeAllProtos":      vm.NewCFunction(ObjectRemoveAllProtos, nil),
		"removeAllSlots":       vm.NewCFunction(ObjectRemoveAllSlots, nil),
		"removeProto":          vm.NewCFunction(ObjectRemoveProto, nil),
		"removeSlot":           vm.NewCFunction(ObjectRemoveSlot, nil),
		"return":               vm.NewCFunction(ObjectReturn, nil), // control.go
		"setProto":             vm.NewCFunction(ObjectSetProto, nil),
		"setProtos":            vm.NewCFunction(ObjectSetProtos, nil),
		"setSlot":              vm.NewCFunction(ObjectSetSlot, nil),
		"shallowCopy":          vm.NewCFunction(ObjectShallowCopy, nil),
		"slotNames":            vm.NewCFunction(ObjectSlotNames, nil),
		"slotValues":           vm.NewCFunction(ObjectSlotValues, nil),
		"stopStatus":           vm.NewCFunction(ObjectStopStatus, nil),
		"thisContext":          vm.NewCFunction(ObjectThisContext, nil),
		"thisLocalContext":     vm.NewCFunction(ObjectThisLocalContext, nil),
		"thisMessage":          vm.NewCFunction(ObjectThisMessage, nil),
		"try":                  vm.NewCFunction(ObjectTry, nil),
		"type":                 vm.NewString("Object"),
		"uniqueId":             vm.NewCFunction(ObjectUniqueID, nil),
		"updateSlot":           vm.NewCFunction(ObjectUpdateSlot, nil),
		"wait":                 vm.NewCFunction(ObjectWait, nil),
		"while":                vm.NewCFunction(ObjectWhile, nil), // control.go
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
	vm.SetSlots(vm.BaseObject, slots)
	vm.SetSlot(vm.Core, "Object", vm.BaseObject)
}

// ObjectWith creates a new object with the given slots, protos, value, and
// tag.
func (vm *VM) ObjectWith(slots Slots, protos []*Object, value interface{}, tag Tag) *Object {
	if protos == nil {
		protos = []*Object{}
	}
	r := &Object{
		Value: value,
		tag:   tag,
		id:    nextObject(),
	}
	r.SetProtos(protos...)
	vm.definitelyNewSlots(r, slots)
	return r
}

// NewObject creates a new object with the given slots and with the VM's
// Core Object as its proto.
func (vm *VM) NewObject(slots Slots) *Object {
	r := &Object{
		protos: protoLink{p: vm.BaseObject},
		id:     nextObject(),
	}
	vm.definitelyNewSlots(r, slots)
	return r
}

// TypeName gets the name of the type of an object by activating its type slot.
// If there is no such slot, then its tag's name will be returned; if its tag
// is nil, then its name is Object.
func (vm *VM) TypeName(o *Object) string {
	if typ, proto := vm.GetSlot(o, "type"); proto != nil {
		return vm.AsString(typ)
	}
	if o.Tag() != nil {
		return o.Tag().String()
	}
	return "Object"
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals *Object, msg *Message) *Object {
	clone := target.Clone()
	if init, proto := vm.GetSlot(target, "init"); proto != nil {
		// By calling Activate directly, any control flow it sends remains on
		// vm.Control. We don't have to call vm.Stop.
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
	sy := vm.SetSlotSync(target, slot)
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		sy.Set(v)
	}
	sy.Unlock()
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
	_, proto := vm.GetSlot(target, slot)
	if proto == nil {
		return vm.RaiseExceptionf("slot %s not found", slot)
	}
	sy := vm.SetSlotSync(target, slot)
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop == NoStop {
		sy.Set(v)
	}
	sy.Unlock()
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
	v, _ := vm.GetSlot(target, slot)
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
	v, _ := vm.GetLocalSlot(target, slot)
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
	_, ok := vm.GetLocalSlot(target, slot)
	return vm.IoBool(ok)
}

// ObjectSlotNames is an Object method.
//
// slotNames returns a list of the names of the slots on this object.
func ObjectSlotNames(vm *VM, target, locals *Object, msg *Message) *Object {
	v := []*Object{}
	vm.ForeachSlot(target, func(key string, value SyncSlot) bool {
		if value.Valid() {
			v = append(v, vm.NewString(key))
		}
		return true
	})
	return vm.NewList(v...)
}

// ObjectSlotValues is an Object method.
//
// slotValues returns a list of the values of the slots on this obect.
func ObjectSlotValues(vm *VM, target, locals *Object, msg *Message) *Object {
	v := []*Object{}
	vm.ForeachSlot(target, func(key string, value SyncSlot) bool {
		value.Lock()
		if value.Valid() {
			v = append(v, value.Load())
		}
		value.Unlock()
		return true
	})
	return vm.NewList(v...)
}

// ObjectProtos is an Object method.
//
// protos returns a list of the receiver's protos.
func ObjectProtos(vm *VM, target, locals *Object, msg *Message) *Object {
	v := target.Protos()
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
	target.AppendProto(locals)
	result, stop := msg.EvalArgAt(vm, target, 0)
	target.RemoveProto(locals)
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

// Compare uses the compare method of x to compare to y. If the result is a
// Number, then its value is converted to int and returned as c along with nil
// for obj and NoStop for stop. Otherwise, c is 0, and obj and stop are the
// results of evaluation.
func (vm *VM) Compare(x, y *Object) (c int, obj *Object, stop Stop) {
	cmp, proto := vm.GetSlot(x, "compare")
	if proto == nil {
		// No compare method.
		return PtrCompare(x, y), nil, NoStop
	}
	obj, stop = vm.Status(cmp.Activate(vm, x, x, proto, vm.IdentMessage("compare", vm.CachedMessage(y))))
	if stop != NoStop {
		vm.Stop(obj, stop)
		return
	}
	obj.Lock()
	if obj.Tag() != NumberTag {
		obj.Unlock()
		return
	}
	c = int(obj.Value.(float64))
	obj.Unlock()
	return c, nil, NoStop
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
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) < 0)
	}
	return vm.IoBool(c < 0)
}

// ObjectLessOrEqual is an Object method.
//
// x <=(y) returns true if the result of x compare(y) is not 1.
func ObjectLessOrEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) <= 0)
	}
	return vm.IoBool(c <= 0)
}

// ObjectEqual is an Object method.
//
// x ==(y) returns true if the result of x compare(y) is 0.
func ObjectEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) == 0)
	}
	return vm.IoBool(c == 0)
}

// ObjectNotEqual is an Object method.
//
// x !=(y) returns true if the result of x compare(y) is not 0.
func ObjectNotEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) != 0)
	}
	return vm.IoBool(c != 0)
}

// ObjectGreaterOrEqual is an Object method.
//
// x >=(y) returns true if the result of x compare(y) is not -1.
func ObjectGreaterOrEqual(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) >= 0)
	}
	return vm.IoBool(c >= 0)
}

// ObjectGreater is an Object method.
//
// x >(y) returns true if the result of x compare(y) is 1.
func ObjectGreater(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	c, obj, stop := vm.Compare(target, x)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if obj != nil {
		return vm.IoBool(PtrCompare(target, obj) > 0)
	}
	return vm.IoBool(c > 0)
}

// ObjectTry is an Object method.
//
// try executes its message, returning any exception that occurs or nil if none
// does. Any other control flow (continue, break, return) is passed normally.
func ObjectTry(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	switch stop {
	case NoStop: // do nothing
	case ContinueStop, BreakStop, ReturnStop, ExitStop:
		return vm.Stop(r, stop)
	case ExceptionStop:
		return r
	default:
		panic(fmt.Errorf("iolang: invalid Stop: %w", stop.Err()))
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
	r := vm.Nil
	target.ForeachProto(func(proto *Object) bool {
		_, proto = vm.GetSlot(proto, slot)
		if proto != nil {
			r = proto
			return false
		}
		return true
	})
	return r
}

// ObjectAppendProto is an Object method.
//
// appendProto adds an object as a proto to the object.
func ObjectAppendProto(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	target.AppendProto(v)
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
	_, proto := vm.GetSlot(target, slot)
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
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseExceptionf("foreachSlot requires 2 or 3 args")
	}
	vm.ForeachSlot(target, func(key string, value SyncSlot) bool {
		v := value.Snap()
		if v == nil {
			return true
		}
		vm.SetSlot(locals, vn, v)
		if hkn {
			vm.SetSlot(locals, kn, vm.NewString(key))
		}
		var control Stop
		result, control = ev.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return false
		case ReturnStop, ExceptionStop, ExitStop:
			vm.Stop(result, control)
			return false
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
		return true
	})
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
	defer r.Unlock()
	switch a := r.Value.(type) {
	case Sequence:
		// String name, arguments are messages.
		name := a.String()
		m := vm.IdentMessage(name, msg.Args[1:]...)
		for i, arg := range m.Args {
			m.Args[i] = arg.DeepCopy()
		}
		return vm.Stop(vm.Perform(target, locals, m))
	case *Message:
		// Message argument, which provides both the name and the args.
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
	target.PrependProto(p)
	return target
}

// ObjectPrint is an Object method.
//
// print converts the receiver to a string and prints it.
func ObjectPrint(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := vm.Perform(target, locals, vm.IdentMessage("asString"))
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	r.Lock()
	defer r.Unlock()
	if r.Tag() == SequenceTag {
		fmt.Print(r.Value)
	} else {
		fmt.Printf("%v_%p", r.Tag(), r.Value)
	}
	return target
}

// ObjectRemoveAllProtos is an Object method.
//
// removeAllProtos removes all protos from the object.
func ObjectRemoveAllProtos(vm *VM, target, locals *Object, msg *Message) *Object {
	target.SetProtos()
	return target
}

// ObjectRemoveAllSlots is an Object method.
//
// removeAllSlots removes all slots from the object.
func ObjectRemoveAllSlots(vm *VM, target, locals *Object, msg *Message) *Object {
	vm.RemoveAllSlots(target)
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
	target.RemoveProto(p)
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
	vm.RemoveSlot(target, slot)
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
	target.SetProtos(p)
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
	target.SetProtos(l...)
	return target
}

// ObjectShallowCopy is an Object method.
//
// shallowCopy creates a new object with the receiver's slots and protos.
func ObjectShallowCopy(vm *VM, target, locals *Object, msg *Message) *Object {
	if target.Tag() != nil {
		return vm.RaiseExceptionf("shallowCopy cannot be used on primitives")
	}
	// The shallow copy in Io doesn't actually copy the protos...
	protos := target.Protos()
	r := vm.ObjectWith(nil, protos, nil, nil)
	vm.ForeachSlot(target, func(key string, value SyncSlot) bool {
		value.Lock()
		if value.Valid() {
			vm.SetSlot(r, key, value.Load())
		}
		value.Unlock()
		return true
	})
	return r
}

// ObjectStopStatus is an Object method.
//
// stopStatus returns the object associated with the control flow status
// returned when evaluating the argument message. Exceptions continue
// propagating.
func ObjectStopStatus(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	switch stop {
	case NoStop:
		r, _ = vm.GetLocalSlot(vm.Core, "Normal")
	case ContinueStop:
		r, _ = vm.GetLocalSlot(vm.Core, "Continue")
	case BreakStop:
		r, _ = vm.GetLocalSlot(vm.Core, "Break")
	case ReturnStop:
		r, _ = vm.GetLocalSlot(vm.Core, "Return")
	case ExceptionStop, ExitStop:
		return vm.Stop(r, stop)
	default:
		panic(fmt.Errorf("iolang: invalid Stop: %w", stop.Err()))
	}
	return r
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
