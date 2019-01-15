package iolang

import (
	"fmt"
)

// A Map is an associative array with string keys.
type Map struct {
	Object
	Value map[string]Interface
}

// NewMap creates a new Map object with the given value, which may be nil.
func (vm *VM) NewMap(value map[string]Interface) *Map {
	m := Map{
		Object: *vm.CoreInstance("Map"),
		Value:  value,
	}
	if m.Value == nil {
		m.Value = map[string]Interface{}
	}
	return &m
}

// Activate returns the map.
func (m *Map) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return m
}

// Clone creates a clone of the map.
func (m *Map) Clone() Interface {
	n := make(map[string]Interface, len(m.Value))
	for k, v := range m.Value {
		n[k] = v
	}
	return &Map{
		Object: Object{Slots: Slots{}, Protos: []Interface{m}},
		Value:  n,
	}
}

func (vm *VM) initMap() {
	var exemplar *Map
	slots := Slots{
		"at":            vm.NewTypedCFunction(MapAt, exemplar),
		"atIfAbsentPut": vm.NewTypedCFunction(MapAtIfAbsentPut, exemplar),
		"atPut":         vm.NewTypedCFunction(MapAtPut, exemplar),
		"empty":         vm.NewTypedCFunction(MapEmpty, exemplar),
		"foreach":       vm.NewTypedCFunction(MapForeach, exemplar),
		"hasKey":        vm.NewTypedCFunction(MapHasKey, exemplar),
		"keys":          vm.NewTypedCFunction(MapKeys, exemplar),
		"removeAt":      vm.NewTypedCFunction(MapRemoveAt, exemplar),
		"size":          vm.NewTypedCFunction(MapSize, exemplar),
		"type":          vm.NewString("Map"),
		"values":        vm.NewTypedCFunction(MapValues, exemplar),
	}
	SetSlot(vm.Core, "Map", &Map{Object: *vm.ObjectWith(slots), Value: map[string]Interface{}})
}

// MapAt is a Map method.
//
// at returns the value at the given key, or the default value if it is
// missing.
func MapAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	k, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v, ok := m.Value[k.String()]
	if ok {
		return v
	}
	return msg.EvalArgAt(vm, locals, 1)
}

// MapAtIfAbsentPut is a Map method.
//
// atIfAbsentPut sets the given key if it is not already in the map and returns
// the value at the key if it is.
func MapAtIfAbsentPut(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	kk, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	k := kk.String()
	v, ok := m.Value[k]
	if ok {
		return v
	}
	v, ok = CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return v
	}
	m.Value[k] = v
	return v
}

// MapAtPut is a Map method.
//
// atPut sets the value of the given string key.
func MapAtPut(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	k, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return v
	}
	m.Value[k.String()] = v
	return target
}

// MapEmpty is a Map method.
//
// empty removes all items from the map.
func MapEmpty(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	m.Value = map[string]Interface{}
	return target
}

// MapForeach is a Map method.
//
// foreach performs a loop on each key of the map in random order, setting
// key and value variables, with the key variable being optional.
func MapForeach(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseException("foreach requires 2 or 3 args")
	}
	m := target.(*Map)
	for k, v := range m.Value {
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

// MapHasKey is a Map method.
//
// hasKey returns true if the key exists in the map.
func MapHasKey(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	k, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	_, ok := m.Value[k.String()]
	return vm.IoBool(ok)
}

// MapKeys is a Map method.
//
// keys returns a list of all keys in the map in random order.
func MapKeys(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	l := make([]Interface, 0, len(m.Value))
	for k := range m.Value {
		l = append(l, vm.NewString(k))
	}
	return vm.NewList(l...)
}

// MapRemoveAt is a Map method.
//
// removeAt removes a key from the map if it exists.
func MapRemoveAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	k, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	delete(m.Value, k.String())
	return target
}

// MapSize is a Map method.
//
// size returns the number of values in the map.
func MapSize(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	return vm.NewNumber(float64(len(m.Value)))
}

// MapValues is a Map method.
//
// values returns a list of all values in the map in random order.
func MapValues(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Map)
	l := make([]Interface, 0, len(m.Value))
	for _, v := range m.Value {
		l = append(l, v)
	}
	return vm.NewList(l...)
}
