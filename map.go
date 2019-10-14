package iolang

import (
	"fmt"
)

// tagMap is the Tag type for Map objects.
type tagMap struct{}

func (tagMap) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagMap) CloneValue(value interface{}) interface{} {
	m := value.(map[string]*Object)
	n := make(map[string]*Object, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

func (tagMap) String() string {
	return "Map"
}

// MapTag is the Tag for Map objects. Activate returns self. CloneValue copies
// the keys and values in the map.
var MapTag Tag = tagMap{}

// NewMap creates a new Map object with the given value, which may be nil.
func (vm *VM) NewMap(value map[string]*Object) *Object {
	if value == nil {
		value = make(map[string]*Object)
	}
	return &Object{
		Protos: vm.CoreProto("Map"),
		Value:  value,
		Tag:    MapTag,
	}
}

func (vm *VM) initMap() {
	slots := Slots{
		"at":            vm.NewCFunction(MapAt, MapTag),
		"atIfAbsentPut": vm.NewCFunction(MapAtIfAbsentPut, MapTag),
		"atPut":         vm.NewCFunction(MapAtPut, MapTag),
		"empty":         vm.NewCFunction(MapEmpty, MapTag),
		"foreach":       vm.NewCFunction(MapForeach, MapTag),
		"hasKey":        vm.NewCFunction(MapHasKey, MapTag),
		"keys":          vm.NewCFunction(MapKeys, MapTag),
		"removeAt":      vm.NewCFunction(MapRemoveAt, MapTag),
		"size":          vm.NewCFunction(MapSize, MapTag),
		"type":          vm.NewString("Map"),
		"values":        vm.NewCFunction(MapValues, MapTag),
	}
	vm.Core.SetSlot("Map", &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
		Value:  make(map[string]*Object, 0),
		Tag:    MapTag,
	})
}

// MapAt is a Map method.
//
// at returns the value at the given key, or the default value if it is
// missing.
func MapAt(vm *VM, target, locals *Object, msg *Message) *Object {
	key, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	m := target.Value.(map[string]*Object)
	v, ok := m[key]
	target.Unlock()
	if ok {
		return v
	}
	return vm.Stop(msg.EvalArgAt(vm, locals, 1))
}

// MapAtIfAbsentPut is a Map method.
//
// atIfAbsentPut sets the given key if it is not already in the map and returns
// the value at the key if it is.
func MapAtIfAbsentPut(vm *VM, target, locals *Object, msg *Message) *Object {
	key, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	// Io only evaluates the second argument if the key is not contained, but
	// we'll evaluate in all cases so that we can fully synchronize the
	// operation safely.
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	target.Lock()
	m := target.Value.(map[string]*Object)
	w, ok := m[key]
	if ok {
		target.Unlock()
		return w
	}
	m[key] = v
	target.Unlock()
	return v
}

// MapAtPut is a Map method.
//
// atPut sets the value of the given string key.
func MapAtPut(vm *VM, target, locals *Object, msg *Message) *Object {
	key, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	target.Lock()
	m := target.Value.(map[string]*Object)
	m[key] = v
	target.Unlock()
	return target
}

// MapEmpty is a Map method.
//
// empty removes all items from the map.
func MapEmpty(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	target.Value = map[string]*Object{}
	target.Unlock()
	return target
}

// MapForeach is a Map method.
//
// foreach performs a loop on each key of the map in random order, setting
// key and value variables, with the key variable being optional. If keys are
// added to the map while the loop is being evaluated, then those additions are
// not included in the loop; any keys removed during the loop are iterated with
// nil value.
func MapForeach(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, _, ev := ForeachArgs(msg)
	if !hkn {
		return vm.RaiseExceptionf("foreach requires 2 or 3 args")
	}
	// We don't want to hold the lock while performing code, so we first grab
	// the list of keys while holding the lock, then reacquire the lock for
	// each iteration of the loop while getting the item.
	target.Lock()
	m := target.Value.(map[string]*Object)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	target.Unlock()
	var control Stop
	for _, k := range keys {
		target.Lock()
		v := target.Value.(map[string]*Object)[k]
		target.Unlock()
		if v == nil {
			v = vm.Nil
		}
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

// MapHasKey is a Map method.
//
// hasKey returns true if the key exists in the map.
func MapHasKey(vm *VM, target, locals *Object, msg *Message) *Object {
	key, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	m := target.Value.(map[string]*Object)
	_, ok := m[key]
	target.Unlock()
	return vm.IoBool(ok)
}

// MapKeys is a Map method.
//
// keys returns a list of all keys in the map in random order.
func MapKeys(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	m := target.Value.(map[string]*Object)
	l := make([]*Object, 0, len(m))
	for k := range m {
		l = append(l, vm.NewString(k))
	}
	target.Unlock()
	return vm.NewList(l...)
}

// MapRemoveAt is a Map method.
//
// removeAt removes a key from the map if it exists.
func MapRemoveAt(vm *VM, target, locals *Object, msg *Message) *Object {
	key, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	m := target.Value.(map[string]*Object)
	delete(m, key)
	target.Unlock()
	return target
}

// MapSize is a Map method.
//
// size returns the number of values in the map.
func MapSize(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	n := len(target.Value.(map[string]*Object))
	target.Unlock()
	return vm.NewNumber(float64(n))
}

// MapValues is a Map method.
//
// values returns a list of all values in the map in random order.
func MapValues(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	m := target.Value.(map[string]*Object)
	l := make([]*Object, 0, len(m))
	for _, v := range m {
		l = append(l, v)
	}
	target.Unlock()
	return vm.NewList(l...)
}
