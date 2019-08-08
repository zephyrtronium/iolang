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
func (m *Map) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return m, NoStop
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
	var kind *Map
	slots := Slots{
		"at":            vm.NewCFunction(MapAt, kind),
		"atIfAbsentPut": vm.NewCFunction(MapAtIfAbsentPut, kind),
		"atPut":         vm.NewCFunction(MapAtPut, kind),
		"empty":         vm.NewCFunction(MapEmpty, kind),
		"foreach":       vm.NewCFunction(MapForeach, kind),
		"hasKey":        vm.NewCFunction(MapHasKey, kind),
		"keys":          vm.NewCFunction(MapKeys, kind),
		"removeAt":      vm.NewCFunction(MapRemoveAt, kind),
		"size":          vm.NewCFunction(MapSize, kind),
		"type":          vm.NewString("Map"),
		"values":        vm.NewCFunction(MapValues, kind),
	}
	vm.SetSlot(vm.Core, "Map", &Map{Object: *vm.ObjectWith(slots), Value: map[string]Interface{}})
}

// MapAt is a Map method.
//
// at returns the value at the given key, or the default value if it is
// missing.
func MapAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	k, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, ok := m.Value[k.String()]
	if ok {
		return v, NoStop
	}
	return msg.EvalArgAt(vm, locals, 1)
}

// MapAtIfAbsentPut is a Map method.
//
// atIfAbsentPut sets the given key if it is not already in the map and returns
// the value at the key if it is.
func MapAtIfAbsentPut(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	kk, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := kk.String()
	v, ok := m.Value[k]
	if ok {
		return v, NoStop
	}
	v, stop = msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return v, stop
	}
	m.Value[k] = v
	return v, NoStop
}

// MapAtPut is a Map method.
//
// atPut sets the value of the given string key.
func MapAtPut(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	k, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return v, stop
	}
	m.Value[k.String()] = v
	return target, NoStop
}

// MapEmpty is a Map method.
//
// empty removes all items from the map.
func MapEmpty(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	m.Value = map[string]Interface{}
	return target, NoStop
}

// MapForeach is a Map method.
//
// foreach performs a loop on each key of the map in random order, setting
// key and value variables, with the key variable being optional.
func MapForeach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseException("foreach requires 2 or 3 args")
	}
	m := target.(*Map)
	for k, v := range m.Value {
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

// MapHasKey is a Map method.
//
// hasKey returns true if the key exists in the map.
func MapHasKey(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	k, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	_, ok := m.Value[k.String()]
	return vm.IoBool(ok), NoStop
}

// MapKeys is a Map method.
//
// keys returns a list of all keys in the map in random order.
func MapKeys(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	l := make([]Interface, 0, len(m.Value))
	for k := range m.Value {
		l = append(l, vm.NewString(k))
	}
	return vm.NewList(l...), NoStop
}

// MapRemoveAt is a Map method.
//
// removeAt removes a key from the map if it exists.
func MapRemoveAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	k, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	delete(m.Value, k.String())
	return target, NoStop
}

// MapSize is a Map method.
//
// size returns the number of values in the map.
func MapSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	return vm.NewNumber(float64(len(m.Value))), NoStop
}

// MapValues is a Map method.
//
// values returns a list of all values in the map in random order.
func MapValues(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := target.(*Map)
	l := make([]Interface, 0, len(m.Value))
	for _, v := range m.Value {
		l = append(l, v)
	}
	return vm.NewList(l...), NoStop
}
