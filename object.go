package iolang

import (
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"strings"
	"sync"
)

// Any Io object. To satisfy this interface, *Object's method set must be
// embedded and Clone() implemented to return a value of the new type.
type Interface interface {
	// Get slots and protos.
	SP() *Object
	// Create an object with empty slots and this object as its only proto.
	Clone() Interface

	isIoObject()
}

// An object which activates. Probably just CFunctions and Blocks.
type Actor interface {
	Interface
	Activate(vm *VM, target, locals Interface, msg *Message) Interface
}

type Slots map[string]Interface

type Object struct {
	Slots  Slots
	Protos []Interface

	// The lock should be held when accessing slots or protos directly.
	L sync.Mutex
}

func (o *Object) SP() *Object {
	return o
}

func (o *Object) Clone() Interface {
	return &Object{Slots: Slots{}, Protos: []Interface{o}}
}

func (*Object) isIoObject() {}

// This sets up the "base" object that is the first proto of all other
// built-in types, not the slots of "default" objects (which have empty slots).
func (vm *VM) initObject() {
	vm.BaseObject.Protos = []Interface{vm.Lobby}
	slots := Slots{
		"":           vm.NewCFunction(ObjectEvalArg, "ObjectEvalArg(msg)"),
		"Lobby":      vm.Lobby,
		"Object":     vm.BaseObject,
		"asString":   vm.NewCFunction(ObjectAsString, "ObjectAsString()"),
		"break":      vm.NewCFunction(ObjectBreak, "ObjectBreak(result)"),
		"clone":      vm.NewCFunction(ObjectClone, "ObjectClone()"),
		"continue":   vm.NewCFunction(ObjectContinue, "ObjectContinue()"),
		"false":      vm.False,
		"for":        vm.NewCFunction(ObjectFor, "ObjectFor(ctr, start, stop, [step,] msg)"),
		"getSlot":    vm.NewCFunction(ObjectGetSlot, "ObjectGetSlot(name)"),
		"if":         vm.NewCFunction(ObjectIf, "ObjectIf(cond, onTrue, onFalse)"),
		"isTrue":     vm.True,
		"loop":       vm.NewCFunction(ObjectLoop, "ObjectLoop(msg)"),
		"nil":        vm.Nil,
		"return":     vm.NewCFunction(ObjectReturn, "ObjectReturn(result)"),
		"setSlot":    vm.NewCFunction(ObjectSetSlot, "ObjectSetSlot(name, value)"),
		"true":       vm.True,
		"type":       vm.NewString("Object"),
		"updateSlot": vm.NewCFunction(ObjectUpdateSlot, "ObjectUpdateSlot(name, value)"),
		"while":      vm.NewCFunction(ObjectWhile, "ObjectWhile(cond, msg)"),
	}
	vm.BaseObject.Slots = slots

	slots["returnIfNonNil"] = slots["return"]
}

func (vm *VM) ObjectWith(slots Slots) *Object {
	return &Object{Slots: slots, Protos: []Interface{vm.BaseObject}}
}

// Get a slot, checking protos in depth-first order without duplicates. The
// proto is the object which actually had the slot. If the slot is not found,
// both returned values will be nil.
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

// Set a slot's value.
func SetSlot(o Interface, slot string, value Interface) {
	obj := o.SP()
	obj.L.Lock()
	defer obj.L.Unlock()
	if obj.Slots == nil {
		obj.Slots = Slots{}
	}
	obj.Slots[slot] = value
}

// Get the name of the type of an object.
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

// Activate with a simple identifier message with the given text and with the
// given arguments.
func (vm *VM) SimpleActivate(o Actor, self, locals Interface, text string, args ...Interface) Interface {
	a := make([]*Message, len(args))
	for i, arg := range args {
		// Since we are setting memos, we don't have to set anything else
		// because the evaluator will see the memo first. If that behavior
		// changes, this must be changed with it.
		a[i] = &Message{Memo: arg}
	}
	result := o.Activate(vm, self, locals, &Message{Symbol: Symbol{Kind: IdentSym, Text: text}, Args: a})
	if stop, ok := result.(Stop); ok {
		return stop.Result
	}
	return result
}

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

func ObjectSetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v := msg.EvalArgAt(vm, locals, 1)
	if IsIoError(v) {
		return v
	}
	SetSlot(target, slot.Value, v)
	return v
}

func ObjectUpdateSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v := msg.EvalArgAt(vm, locals, 1)
	if IsIoError(v) {
		return v
	}
	_, proto := GetSlot(target, slot.Value)
	if proto == nil {
		return vm.NewExceptionf("slot %s not found", slot.Value)
	}
	SetSlot(proto, slot.Value, v)
	return v
}

func ObjectGetSlot(vm *VM, target, locals Interface, msg *Message) Interface {
	slot, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	v, _ := GetSlot(target, slot.Value)
	return v
}

func ObjectSlotNames(vm *VM, target, locals Interface, msg *Message) Interface {
	slots := target.SP().Slots
	names := make([]string, len(slots))
	i := 0
	for name := range slots {
		names[i] = name
		i++
	}
	return vm.NewString(strings.Join(names, ", "))
}

func ObjectEvalArg(vm *VM, target, locals Interface, msg *Message) Interface {
	// The original Io implementation has an assertion that there is at least
	// one argument; this will instead return vm.Nil. It wouldn't be difficult
	// to mimic Io's behavior, but ehhh.
	return msg.ArgAt(0).Eval(vm, locals)
}

func ObjectEvalArgAndReturnSelf(vm *VM, target, locals Interface, msg *Message) Interface {
	if result := msg.ArgAt(0).Eval(vm, locals); IsIoError(result) {
		return result
	}
	return target
}

func ObjectEvalArgAndReturnNil(vm *VM, target, locals Interface, msg *Message) Interface {
	if result := msg.ArgAt(0).Eval(vm, locals); IsIoError(result) {
		return result
	}
	return vm.Nil
}

func ObjectAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	if stringer, ok := target.(fmt.Stringer); ok {
		return vm.NewString(stringer.String())
	}
	return vm.NewString(fmt.Sprintf("%T_%p", target, target))
}
