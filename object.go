package iolang

import (
	"fmt"
	"os"
	"reflect"
	"strings"
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
	Activate(vm *VM, target, locals, context Interface, msg *Message) Interface
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

// Activate activates the object. If the isActivatable slot is true, and the
// activate slot exists, then this activates that slot; otherwise, it returns
// the object.
func (o *Object) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	ok, proto := GetSlot(o, "isActivatable")
	// We can't use vm.AsBool even though it's one of the few situations where
	// we'd want to, because it will attempt to activate the isTrue slot, which
	// is typically a plain Object, which will activate this method and recurse
	// infinitely.
	if proto == nil || ok == vm.False || ok == vm.Nil {
		return o
	}
	act, proto := GetSlot(o, "activate")
	if proto != nil {
		return act.Activate(vm, target, locals, context, msg)
	}
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
		"":                     vm.NewCFunction(ObjectEvalArg),
		"!=":                   vm.NewCFunction(ObjectNotEqual),
		"<":                    vm.NewCFunction(ObjectLess),
		"<=":                   vm.NewCFunction(ObjectLessOrEqual),
		"==":                   vm.NewCFunction(ObjectEqual),
		">":                    vm.NewCFunction(ObjectGreater),
		">=":                   vm.NewCFunction(ObjectGreaterOrEqual),
		"ancestorWithSlot":     vm.NewCFunction(ObjectAncestorWithSlot),
		"appendProto":          vm.NewCFunction(ObjectAppendProto),
		"asString":             vm.NewCFunction(ObjectAsString),
		"block":                vm.NewCFunction(ObjectBlock),
		"break":                vm.NewCFunction(ObjectBreak),
		"clone":                vm.NewCFunction(ObjectClone),
		"cloneWithoutInit":     vm.NewCFunction(ObjectCloneWithoutInit),
		"compare":              vm.NewCFunction(ObjectCompare),
		"contextWithSlot":      vm.NewCFunction(ObjectContextWithSlot),
		"continue":             vm.NewCFunction(ObjectContinue),
		"do":                   vm.NewCFunction(ObjectDo),
		"doFile":               vm.NewCFunction(ObjectDoFile),
		"doMessage":            vm.NewCFunction(ObjectDoMessage),
		"doString":             vm.NewCFunction(ObjectDoString),
		"evalArgAndReturnNil":  vm.NewCFunction(ObjectEvalArgAndReturnNil),
		"evalArgAndReturnSelf": vm.NewCFunction(ObjectEvalArgAndReturnSelf),
		"for":                  vm.NewCFunction(ObjectFor),
		"foreachSlot":          vm.NewCFunction(ObjectForeachSlot),
		"getLocalSlot":         vm.NewCFunction(ObjectGetLocalSlot),
		"getSlot":              vm.NewCFunction(ObjectGetSlot),
		"hasLocalSlot":         vm.NewCFunction(ObjectHasLocalSlot),
		"if":                   vm.NewCFunction(ObjectIf),
		"isIdenticalTo":        vm.NewCFunction(ObjectIsIdenticalTo),
		"isNil":                vm.False,
		"isTrue":               vm.True,
		"lexicalDo":            vm.NewCFunction(ObjectLexicalDo),
		"loop":                 vm.NewCFunction(ObjectLoop),
		"message":              vm.NewCFunction(ObjectMessage),
		"method":               vm.NewCFunction(ObjectMethod),
		"not":                  vm.Nil,
		"or":                   vm.True,
		"perform":              vm.NewCFunction(ObjectPerform),
		"performWithArgList":   vm.NewCFunction(ObjectPerformWithArgList),
		"prependProto":         vm.NewCFunction(ObjectPrependProto),
		"protos":               vm.NewCFunction(ObjectProtos),
		"removeAllProtos":      vm.NewCFunction(ObjectRemoveAllProtos),
		"removeAllSlots":       vm.NewCFunction(ObjectRemoveAllSlots),
		"removeProto":          vm.NewCFunction(ObjectRemoveProto),
		"removeSlot":           vm.NewCFunction(ObjectRemoveSlot),
		"return":               vm.NewCFunction(ObjectReturn),
		"returnIfNonNil":       vm.NewCFunction(ObjectReturnIfNonNil),
		"setProto":             vm.NewCFunction(ObjectSetProto),
		"setProtos":            vm.NewCFunction(ObjectSetProtos),
		"setSlot":              vm.NewCFunction(ObjectSetSlot),
		"shallowCopy":          vm.NewCFunction(ObjectShallowCopy),
		"slotNames":            vm.NewCFunction(ObjectSlotNames),
		"slotValues":           vm.NewCFunction(ObjectSlotValues),
		"thisContext":          vm.NewCFunction(ObjectThisContext),
		"thisLocalContext":     vm.NewCFunction(ObjectThisLocalContext),
		"thisMessage":          vm.NewCFunction(ObjectThisMessage),
		"try":                  vm.NewCFunction(ObjectTry),
		"type":                 vm.NewString("Object"),
		"uniqueId":             vm.NewCFunction(ObjectUniqueId),
		"updateSlot":           vm.NewCFunction(ObjectUpdateSlot),
		"while":                vm.NewCFunction(ObjectWhile),
	}
	slots["evalArg"] = slots[""]
	slots["ifNil"] = slots["thisContext"]
	slots["ifNilEval"] = slots["thisContext"]
	slots["ifNonNil"] = slots["evalArgAndReturnSelf"]
	slots["ifNonNilEval"] = slots["evalArg"]
	slots["returnIfNonNil"] = slots["return"]
	slots["uniqueHexId"] = slots["uniqueId"]
	vm.BaseObject.Slots = slots
	SetSlot(vm.Core, "Object", vm.BaseObject)
}

// ObjectWith creates a new object with the given slots and with the VM's
// Core Object as its proto.
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
	return getSlotRecurse(o, slot, map[*Object]struct{}{})
}

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

// SimpleActivate activates an object using the identifier message named with
// text and with the given arguments. This does not propagate exceptions.
func (vm *VM) SimpleActivate(o, self, locals Interface, text string, args ...Interface) Interface {
	a := make([]*Message, len(args))
	for i, arg := range args {
		a[i] = vm.CachedMessage(arg)
	}
	result, _ := CheckStop(o.Activate(vm, self, locals, self, &Message{Object: *vm.CoreInstance("Message"), Text: text, Args: a}), ExceptionStop)
	return result
}

// ObjectClone is an Object method.
//
// clone creates a new object with empty slots and the cloned object as its
// proto.
func ObjectClone(vm *VM, target, locals Interface, msg *Message) Interface {
	clone := target.Clone()
	if init, proto := GetSlot(target, "init"); proto != nil {
		r, ok := CheckStop(init.Activate(vm, clone, locals, proto, vm.IdentMessage("init")), LoopStops)
		if !ok {
			return r
		}
	}
	return clone
}

// ObjectCloneWithoutInit is an Object method.
//
// cloneWithoutInit creates a new object with empty slots and the cloned object
// as its proto, without checking for an init slot.
func ObjectCloneWithoutInit(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.Clone()
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

// ObjectGetLocalSlot is an Object method.
//
// getLocalSlot gets the value of a slot on the receiver, not checking its
// protos. The slot is not activated.
func ObjectGetLocalSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	o := target.SP()
	o.L.Lock()
	v := o.Slots[slot.String()]
	o.L.Unlock()
	if v != nil {
		return v
	}
	return vm.Nil
}

// ObjectHasLocalSlot is an Object method.
//
// hasLocalSlot returns whether the object has the given slot name, not
// checking its protos.
func ObjectHasLocalSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	o := target.SP()
	o.L.Lock()
	_, ok := o.Slots[s.String()]
	o.L.Unlock()
	return vm.IoBool(ok)
}

// ObjectSlotNames is an Object method.
//
// slotNames returns a list of the names of the slots on this object.
func ObjectSlotNames(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	names := make([]Interface, 0, len(o.Slots))
	for name := range o.Slots {
		names = append(names, vm.NewString(name))
	}
	o.L.Unlock()
	return vm.NewList(names...)
}

// ObjectSlotValues is an Object method.
//
// slotValues returns a list of the values of the slots on this obect.
func ObjectSlotValues(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	vals := make([]Interface, 0, len(o.Slots))
	for _, val := range o.Slots {
		vals = append(vals, val)
	}
	o.L.Unlock()
	return vm.NewList(vals...)
}

// ObjectProtos is an Object method.
//
// protos returns a list of the receiver's protos.
func ObjectProtos(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	v := make([]Interface, len(o.Protos))
	copy(v, o.Protos)
	o.L.Unlock()
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
	r, _ := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	return r
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
	r, ok := CheckStop(msg.EvalArgAt(vm, target, 0), LoopStops)
	if !ok {
		return r
	}
	return target
}

// ObjectLexicalDo is an Object method.
//
// lexicalDo appends the lexical context to the receiver's protos, evaluates
// the message in the context of the receiver, then removes the added proto.
func ObjectLexicalDo(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	n := len(o.Protos)
	o.Protos = append(o.Protos, locals)
	result, ok := CheckStop(msg.EvalArgAt(vm, target, 0), LoopStops)
	copy(o.Protos[n:], o.Protos[n+1:])
	o.Protos = o.Protos[:len(o.Protos)-1]
	if !ok {
		return result
	}
	return target
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
	r, _ := CheckStop(cmp.Activate(vm, x, x, proto, vm.IdentMessage("compare", arg)), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
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
	x, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return x
	}
	c, ok := CheckStop(vm.Compare(target, x), LoopStops)
	if !ok {
		return x
	}
	if n, ok := c.(*Number); ok {
		return vm.IoBool(n.Value > 0)
	}
	return vm.IoBool(ptrCompare(target, c) > 0)
}

// ObjectTry is an Object method.
//
// try executes its message, returning any exception that occurs or nil if none
// does.
func ObjectTry(vm *VM, target, locals Interface, msg *Message) Interface {
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return r.(Stop).Result
	}
	return vm.Nil
}

// ObjectAncestorWithSlot is an Object method.
//
// ancestorWithSlot returns the proto which owns the given slot.
func ObjectAncestorWithSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	ss := s.String()
	for _, p := range target.SP().Protos {
		_, proto := GetSlot(p, ss)
		if proto != nil {
			return proto
		}
	}
	return vm.Nil
}

// ObjectAppendProto is an Object method.
//
// appendProto adds an object as a proto to the object.
func ObjectAppendProto(vm *VM, target, locals Interface, msg *Message) Interface {
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return v
	}
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	o.Protos = append(o.Protos, v)
	return target
}

// ObjectContextWithSlot is an Object method.
//
// contextWithSlot returns the first of the receiver or its protos which
// contains the given slot.
func ObjectContextWithSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	_, proto := GetSlot(target, s.String())
	if proto != nil {
		return proto
	}
	return vm.Nil
}

// ObjectDoFile is an Object method.
//
// doFile executes the file at the given path in the context of the receiver.
func ObjectDoFile(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	r, _ := CheckStop(vm.DoMessage(m, target), ReturnStop)
	return r
}

// ObjectDoMessage is an Object method.
//
// doMessage sends the message to the receiver.
func ObjectDoMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	m, ok := r.(*Message)
	if !ok {
		return vm.RaiseException("argument 0 to doMessage must be Message, not " + vm.TypeName(r))
	}
	ctxt := target
	if msg.ArgCount() > 1 {
		ctxt, ok = CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
		if !ok {
			return ctxt
		}
	}
	return m.Send(vm, target, ctxt)
}

// ObjectDoString is an Object method.
//
// doString executes the string in the context of the receiver.
func ObjectDoString(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	src := strings.NewReader(s.String())
	label := "doString"
	if msg.ArgCount() > 1 {
		l, stop := msg.StringArgAt(vm, locals, 1)
		if stop != nil {
			return stop
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
func ObjectForeachSlot(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseException("foreach requires 2 or 3 args")
	}
	for k, v := range target.SP().Slots {
		SetSlot(locals, vn, v)
		if hkn {
			SetSlot(locals, kn, vm.NewString(k))
		}
		result = ev.Eval(vm, locals)
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
	return result
}

// ObjectIsIdenticalTo is an Object method.
//
// isIdenticalTo returns whether the object is the same as the argument.
func ObjectIsIdenticalTo(vm *VM, target, locals Interface, msg *Message) Interface {
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	return vm.IoBool(ptrCompare(target, r) == 0)
}

// ObjectMessage is an Object method.
//
// message returns the argument message.
func ObjectMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	if msg.ArgCount() > 0 {
		return msg.ArgAt(0)
	}
	return vm.Nil
}

// ObjectPerform is an Object method.
//
// perform executes the method named by the first argument using the remaining
// argument messages as arguments to the method.
func ObjectPerform(vm *VM, target, locals Interface, msg *Message) Interface {
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	switch a := r.(type) {
	case *Sequence:
		// String name, arguments are messages.
		name := a.String()
		m := vm.IdentMessage(name, msg.Args[1:]...)
		for i, arg := range m.Args {
			m.Args[i] = arg.DeepCopy()
		}
		r, _ := CheckStop(vm.Perform(target, locals, m), ReturnStop)
		return r
	case *Message:
		// Message argument, which provides both the name and the args.
		if msg.ArgCount() > 1 {
			return vm.RaiseException("perform takes a single argument when using a Message as an argument")
		}
		r, _ = CheckStop(vm.Perform(target, locals, a), ReturnStop)
		return r
	}
	return vm.RaiseException("argument 0 to perform must be Sequence or Message, not " + vm.TypeName(r))
}

// ObjectPerformWithArgList is an Object method.
//
// performWithArgList activates the given method with arguments given in the
// second argument as a list.
func ObjectPerformWithArgList(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	l, stop := msg.ListArgAt(vm, locals, 1)
	if stop != nil {
		return stop
	}
	name := s.String()
	m := vm.IdentMessage(name)
	for _, arg := range l.Value {
		m.Args = append(m.Args, vm.CachedMessage(arg))
	}
	slot, proto := GetSlot(target, name)
	if proto != nil {
		r, _ := CheckStop(slot.Activate(vm, target, locals, proto, m), ReturnStop)
		return r
	}
	forward, fp := GetSlot(target, "forward")
	if fp != nil {
		r, _ := CheckStop(forward.Activate(vm, target, locals, fp, m), ReturnStop)
		return r
	}
	return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), name)
}

// ObjectPrependProto is an Object method.
//
// prependProto adds a new proto as the first in the object's protos.
func ObjectPrependProto(vm *VM, target, locals Interface, msg *Message) Interface {
	p, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return p
	}
	o := target.SP()
	o.L.Lock()
	o.Protos = append(o.Protos, p)
	copy(o.Protos[1:], o.Protos)
	o.Protos[0] = p
	o.L.Unlock()
	return target
}

// ObjectRemoveAllProtos is an Object method.
//
// removeAllProtos removes all protos from the object.
func ObjectRemoveAllProtos(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	for i := range o.Protos {
		o.Protos[i] = nil
	}
	o.Protos = []Interface{}
	o.L.Unlock()
	return target
}

// ObjectRemoveAllSlots is an Object method.
//
// removeAllSlots removes all slots from the object.
func ObjectRemoveAllSlots(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	o.Slots = Slots{}
	o.L.Unlock()
	return target
}

// ObjectRemoveProto is an Object method.
//
// removeProto removes the given object from the object's protos.
func ObjectRemoveProto(vm *VM, target, locals Interface, msg *Message) Interface {
	p, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return p
	}
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	n := make([]Interface, len(o.Protos))
	j := 0
	for _, proto := range o.Protos {
		if ptrCompare(proto, p) != 0 {
			n[j] = proto
			j++
		}
	}
	o.Protos = n
	return target
}

// ObjectRemoveSlot is an Object method.
//
// removeSlot removes the given slot from the object.
func ObjectRemoveSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	delete(o.Slots, s.String())
	return target
}

// ObjectReturnIfNonNil is an Object method.
//
// returnIfNonNil executes a return if the receiver is not nil.
func ObjectReturnIfNonNil(vm *VM, target, locals Interface, msg *Message) Interface {
	if ptrCompare(target, vm.Nil) != 0 {
		return Stop{Status: ReturnStop, Result: target}
	}
	return target
}

// ObjectSetProto is an Object method.
//
// setProto sets the object's proto list to have only the given object.
func ObjectSetProto(vm *VM, target, locals Interface, msg *Message) Interface {
	p, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return p
	}
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	o.Protos = append(o.Protos[:0], p)
	return target
}

// ObjectSetProtos is an Object method.
//
// setProtos sets the object's protos to the objects in the given list.
func ObjectSetProtos(vm *VM, target, locals Interface, msg *Message) Interface {
	l, stop := msg.ListArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	o.Protos = append(o.Protos[:0], l.Value...)
	return target
}

// ObjectShallowCopy is an Object method.
//
// shallowCopy creates a new object with the receiver's slots and protos.
func ObjectShallowCopy(vm *VM, target, locals Interface, msg *Message) Interface {
	o, ok := target.(*Object)
	if !ok {
		return vm.RaiseException("shallowCopy cannot be used on primitives")
	}
	o.L.Lock()
	defer o.L.Unlock()
	n := &Object{Slots: make(Slots, len(o.Slots)), Protos: make([]Interface, len(o.Protos))}
	// The shallow copy in Io doesn't actually copy the protos...
	copy(n.Protos, o.Protos)
	for slot, value := range o.Slots {
		n.Slots[slot] = value
	}
	return n
}

// ObjectThisContext is an Object method.
//
// thisContext returns the current slot context, which is the receiver.
func ObjectThisContext(vm *VM, target, locals Interface, msg *Message) Interface {
	return target
}

// ObjectThisLocalContext is an Object method.
//
// thisLocalContext returns the current locals object.
func ObjectThisLocalContext(vm *VM, target, locals Interface, msg *Message) Interface {
	return locals
}

// ObjectThisMessage is an Object method.
//
// thisMessage returns the message which activated this method, which is likely
// to be thisMessage.
func ObjectThisMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	return msg
}

// ObjectUniqueId is an Object method.
//
// uniqueId returns a string representation of the object's address.
func ObjectUniqueId(vm *VM, target, locals Interface, msg *Message) Interface {
	u := reflect.ValueOf(target).Pointer()
	return vm.NewString(fmt.Sprintf("%#x", u))
}
