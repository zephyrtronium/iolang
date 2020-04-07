package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// tagList is the Tag type for List objects.
type tagList struct{}

func (tagList) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagList) CloneValue(value interface{}) interface{} {
	l := value.([]*Object)
	m := make([]*Object, len(l))
	copy(m, l)
	return m
}

func (tagList) String() string {
	return "List"
}

// ListTag is the Tag for List objects. Activate returns self. CloneValue
// creates a shallow copy of the parent's list value.
var ListTag tagList

// NewList creates a List with the given items.
func (vm *VM) NewList(items ...*Object) *Object {
	return vm.ObjectWith(nil, vm.CoreProto("List"), items, ListTag)
}

// ListArgAt evaluates the nth argument and returns it as a slice of objects
// along with the object holding that slice. If a stop occurs during
// evaluation, the slice will be nil, and the stop status and result will be
// returned. If the evaluated result is not a List, the result will be nil, and
// an exception will be returned with an ExceptionStop.
func (m *Message) ListArgAt(vm *VM, locals *Object, n int) ([]*Object, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		lst, ok := v.Value.([]*Object)
		v.Unlock()
		if ok {
			return lst, v, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be List, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return nil, v, s
}

// initList initializes List on this VM.
func (vm *VM) initList() {
	slots := Slots{
		"append":              vm.NewCFunction(ListAppend, ListTag),
		"appendIfAbsent":      vm.NewCFunction(ListAppendIfAbsent, ListTag),
		"appendSeq":           vm.NewCFunction(ListAppendSeq, ListTag),
		"asString":            vm.NewCFunction(ListAsString, ListTag),
		"at":                  vm.NewCFunction(ListAt, ListTag),
		"atInsert":            vm.NewCFunction(ListAtInsert, ListTag),
		"atPut":               vm.NewCFunction(ListAtPut, ListTag),
		"capacity":            vm.NewCFunction(ListCapacity, ListTag),
		"compare":             vm.NewCFunction(ListCompare, ListTag),
		"contains":            vm.NewCFunction(ListContains, ListTag),
		"containsAll":         vm.NewCFunction(ListContainsAll, ListTag),
		"containsAny":         vm.NewCFunction(ListContainsAny, ListTag),
		"containsIdenticalTo": vm.NewCFunction(ListContainsIdenticalTo, ListTag),
		"foreach":             vm.NewCFunction(ListForeach, ListTag),
		"indexOf":             vm.NewCFunction(ListIndexOf, ListTag),
		"preallocateToSize":   vm.NewCFunction(ListPreallocateToSize, ListTag),
		"prepend":             vm.NewCFunction(ListPrepend, ListTag),
		"remove":              vm.NewCFunction(ListRemove, ListTag),
		"removeAll":           vm.NewCFunction(ListRemoveAll, ListTag),
		"removeAt":            vm.NewCFunction(ListRemoveAt, ListTag),
		"reverseForeach":      vm.NewCFunction(ListReverseForeach, ListTag),
		"reverseInPlace":      vm.NewCFunction(ListReverseInPlace, ListTag),
		"setSize":             vm.NewCFunction(ListSetSize, ListTag),
		"size":                vm.NewCFunction(ListSize, ListTag),
		"slice":               vm.NewCFunction(ListSlice, ListTag),
		"sliceInPlace":        vm.NewCFunction(ListSliceInPlace, ListTag),
		"sortInPlace":         vm.NewCFunction(ListSortInPlace, ListTag),
		"sortInPlaceBy":       vm.NewCFunction(ListSortInPlaceBy, ListTag),
		"swapIndices":         vm.NewCFunction(ListSwapIndices, ListTag),
		"type":                vm.NewString("List"),
		"with":                vm.NewCFunction(ListWith, nil),
	}
	slots["empty"] = slots["removeAll"]
	slots["exSlice"] = slots["slice"]
	slots["push"] = slots["append"]
	vm.coreInstall("List", slots, []*Object{}, ListTag)
	vm.SetSlot(vm.BaseObject, "list", slots["with"])
}

// ListAppend is a List method.
//
// append adds items to the end of the list.
func ListAppend(vm *VM, target, locals *Object, msg *Message) *Object {
	n := make([]*Object, len(msg.Args))
	for i, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			// Append the values we've gotten so far.
			target.Lock()
			target.Value = append(target.Value.([]*Object), n[:i]...)
			target.Unlock()
			return vm.Stop(r, stop)
		}
		n[i] = r
	}
	target.Lock()
	target.Value = append(target.Value.([]*Object), n...)
	target.Unlock()
	return target
}

// ListAppendIfAbsent is a List method.
//
// appendIfAbsent adds items to the end of the list if they are not already in
// it.
func ListAppendIfAbsent(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	n := make([]*Object, 0, len(msg.Args))
	defer func() {
		target.Lock()
		target.Value = append(target.Value.([]*Object), n...)
		target.Unlock()
	}()
outer:
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		for _, v := range l {
			c, obj, stop := vm.Compare(v, r)
			if stop != NoStop {
				return vm.Stop(obj, stop)
			}
			if obj == nil && c == 0 {
				continue outer
			}
		}
		n = append(n, r)
	}
	return target
}

// ListAppendSeq is a List method.
//
// appendSeq adds the items in the given lists to the list.
func ListAppendSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	var n []*Object
	k := msg.ArgCount()
	for i := 0; i < k; i++ {
		r, obj, stop := msg.ListArgAt(vm, locals, i)
		if stop != NoStop {
			target.Lock()
			target.Value = append(target.Value.([]*Object), n...)
			target.Unlock()
			return vm.Stop(obj, stop)
		}
		obj.Lock()
		n = append(n, r...)
		obj.Unlock()
	}
	target.Lock()
	target.Value = append(target.Value.([]*Object), n...)
	target.Unlock()
	return target
}

// ListAsString is a List method.
//
// asString creates a string representation of an object.
func ListAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	b := strings.Builder{}
	b.WriteString("list(")
	target.Lock()
	l := target.Value.([]*Object)
	for i := 0; i < len(l); i++ {
		v := l[i]
		target.Unlock()
		b.WriteString(vm.AsString(v))
		target.Lock()
		l = target.Value.([]*Object)
		if i != len(l)-1 {
			b.WriteString(", ")
		}
	}
	target.Unlock()
	b.WriteString(")")
	return vm.NewString(b.String())
}

// ListAt is a List method.
//
// at returns the nth item in the list. All out-of-bounds values are nil.
func ListAt(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := int(n)
	r := vm.Nil
	target.Lock()
	l := target.Value.([]*Object)
	if k >= 0 && k < len(l) {
		r = l[k]
	}
	target.Unlock()
	return r
}

// ListAtInsert is a List method.
//
// atInsert adds an item to the list at the given position, moving back
// existing items at or past that point.
func ListAtInsert(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := int(n)
	r, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	switch {
	case k < 0 || k > len(l):
		target.Unlock()
		return vm.RaiseExceptionf("index out of bounds")
	case k == len(l):
		target.Value = append(l, r)
	default:
		// Make space for the new item, then copy items after its new location
		// up a spot.
		l = append(l, nil)
		copy(l[k+1:], l[k:])
		l[k] = r
		target.Value = l
	}
	target.Unlock()
	return target
}

// ListAtPut is a List method.
//
// atPut replaces an item in the list.
func ListAtPut(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := int(n)
	r, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	if k < 0 || k >= len(l) {
		target.Unlock()
		return vm.RaiseExceptionf("index %d out of bounds", k)
	}
	l[k] = r
	target.Unlock()
	return target
}

// ListCapacity is a List method.
//
// capacity is the number of items for which the list has allocated space.
func ListCapacity(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	return vm.NewNumber(float64(cap(l)))
}

// ListCompare is a List method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func ListCompare(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	v.Lock()
	r, ok := v.Value.([]*Object)
	if !ok {
		target.Unlock()
		v.Unlock()
		return vm.NewNumber(float64(PtrCompare(target, v)))
	}
	s1, s2 := len(l), len(r)
	if s1 != s2 {
		// This is not proper lexicographical order, but it is Io's order.
		target.Unlock()
		v.Unlock()
		if s1 < s2 {
			return vm.NewNumber(-1)
		}
		return vm.NewNumber(1)
	}
	for i := 0; i < len(l) && i < len(r); i++ {
		// Many List methods range over the list value and perform some code
		// (often through vm.Compare) inside the loop. We have to be careful
		// with loops like these, because if the code we're performing tries to
		// use the list while we're holding its lock, then we have a trivial
		// deadlock. To avoid this, we unlock the objects at the beginning of
		// each iteration, then lock again at the end, and finally unlock after
		// the loop exits. We also re-acquire the list value at each iteration
		// so that we see all changes to it during the loop.
		t := l[i]
		u := r[i]
		target.Unlock()
		v.Unlock()
		x, obj, stop := vm.Compare(t, u)
		if stop != NoStop {
			return vm.Stop(obj, stop)
		}
		if obj == nil && x != 0 {
			return vm.NewNumber(float64(x))
		}
		target.Lock()
		v.Lock()
		l = target.Value.([]*Object)
		r = v.Value.([]*Object)
	}
	target.Unlock()
	v.Unlock()
	return vm.NewNumber(0)
}

// ListContains is a List method.
//
// contains returns true if the list contains an item equal to the given
// object.
func ListContains(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	for i := 0; i < len(l); i++ {
		v := l[i]
		target.Unlock()
		c, obj, stop := vm.Compare(v, r)
		if stop != NoStop {
			return vm.Stop(obj, stop)
		}
		if obj == nil && c == 0 {
			return vm.True
		}
		target.Lock()
		l = target.Value.([]*Object)
	}
	target.Unlock()
	return vm.False
}

// ListContainsAll is a List method.
//
// containsAll returns true if the list contains items equal to each of the
// given objects.
func ListContainsAll(vm *VM, target, locals *Object, msg *Message) *Object {
	r := make([]*Object, len(msg.Args))
	for i := range msg.Args {
		var stop Stop
		r[i], stop = msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.Stop(r[i], stop)
		}
	}
outer:
	for _, v := range r {
		target.Lock()
		l := target.Value.([]*Object)
		for i := 0; i < len(l); i++ {
			u := l[i]
			target.Unlock()
			c, obj, stop := vm.Compare(u, v)
			if stop != NoStop {
				return vm.Stop(obj, stop)
			}
			if obj == nil && c == 0 {
				continue outer
			}
			target.Lock()
			l = target.Value.([]*Object)
		}
		target.Unlock()
		return vm.False
	}
	return vm.True
}

// ListContainsAny is a List method.
//
// containsAny returns true if the list contains an item equal to any of the
// given objects.
func ListContainsAny(vm *VM, target, locals *Object, msg *Message) *Object {
	// TODO: use ID checks like ListRemove does
	r := make([]*Object, len(msg.Args))
	var stop Stop
	for i := range msg.Args {
		r[i], stop = msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.Stop(r[i], stop)
		}
	}
	target.Lock()
	l := target.Value.([]*Object)
	for i := 0; i < len(l); i++ {
		v := l[i]
		target.Unlock()
		for _, ri := range r {
			c, obj, stop := vm.Compare(ri, v)
			if stop != NoStop {
				return vm.Stop(obj, stop)
			}
			if obj == nil && c == 0 {
				return vm.True
			}
		}
		target.Lock()
		l = target.Value.([]*Object)
	}
	target.Unlock()
	return vm.False
}

// ListContainsIdenticalTo is a List method.
//
// containsIdenticalTo returns true if the list contains exactly the given
// object.
func ListContainsIdenticalTo(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	for _, v := range l {
		if r == v {
			target.Unlock()
			return vm.True
		}
	}
	target.Unlock()
	return vm.False
}

// ListForeach is a List method.
//
// foreach performs a loop on each item of a list in order, optionally setting
// index and value variables.
func ListForeach(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseExceptionf("foreach requires 1, 2, or 3 arguments")
	}
	target.Lock()
	l := target.Value.([]*Object)
	var control Stop
	for k := 0; k < len(l); k++ {
		v := l[k]
		target.Unlock()
		if hvn {
			vm.SetSlot(locals, vn, v)
			if hkn {
				vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
			}
			result, control = ev.Eval(vm, locals)
		} else {
			result, control = ev.Send(vm, v, locals)
		}
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop, ExitStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
		target.Lock()
		l = target.Value.([]*Object)
	}
	target.Unlock()
	return result
}

// ListIndexOf is a List method.
//
// indexOf returns the first index from the left of an item equal to the
// argument. If there is no such item in the list, nil is returned.
func ListIndexOf(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	target.Lock()
	l := target.Value.([]*Object)
	for i := 0; i < len(l); i++ {
		v := l[i]
		target.Unlock()
		c, obj, stop := vm.Compare(v, r)
		if stop != NoStop {
			return vm.Stop(obj, stop)
		}
		if obj == nil && c == 0 {
			return vm.NewNumber(float64(i))
		}
		target.Lock()
		l = target.Value.([]*Object)
	}
	target.Unlock()
	return vm.Nil
}

// ListPreallocateToSize is a List method.
//
// preallocateToSize ensures that the list has capacity for at least n items.
func ListPreallocateToSize(vm *VM, target, locals *Object, msg *Message) *Object {
	r, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	n := int(r)
	if n < 0 {
		return vm.RaiseExceptionf("can't preallocate to negative size %d", n)
	}
	target.Lock()
	l := target.Value.([]*Object)
	if n > cap(l) {
		v := make([]*Object, len(l), n)
		copy(v, l)
		target.Value = v
	}
	target.Unlock()
	return target
}

// ListPrepend is a List method.
//
// prepend adds items to the beginning of the list.
func ListPrepend(vm *VM, target, locals *Object, msg *Message) *Object {
	nv := make([]*Object, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop == NoStop {
			nv = append(nv, r)
		} else {
			return vm.Stop(r, stop)
		}
	}
	target.Lock()
	l := target.Value.([]*Object)
	// Use our existing space if we can.
	n := len(nv) + len(l)
	if cap(l) >= n {
		l = l[:n]
		copy(l[len(nv):], l)
		copy(l, nv)
	} else {
		l = append(nv, l...)
	}
	target.Value = l
	target.Unlock()
	return target
}

// ListRemove is a List method.
//
// remove removes all occurrences of each item from the list. The behavior of
// this method may be unpredictable if the list is modified concurrently.
func ListRemove(vm *VM, target, locals *Object, msg *Message) *Object {
	rv := make(map[*Object]bool, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		rv[r] = true
	}
	j := 0
	target.Lock()
	l := target.Value.([]*Object)
	for k, v := range l {
		target.Unlock()
		is := false
		// Check whether the value exists by ID first, then check via compare.
		if rv[v] {
			is = true
		} else {
			for r := range rv {
				c, obj, stop := vm.Compare(v, r)
				if stop != NoStop {
					target.Value = l[:len(l)-j]
					return vm.Stop(obj, stop)
				}
				if obj == nil && c == 0 {
					is = true
				}
			}
		}
		target.Lock()
		if is {
			j++
		} else {
			l[k-j] = l[k]
		}
	}
	target.Value = l[:len(l)-j]
	target.Unlock()
	return target
}

// ListRemoveAll is a List method.
//
// removeAll removes all items from the list.
func ListRemoveAll(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Value = l[:0]
	target.Unlock()
	return target
}

// ListRemoveAt is a List method.
//
// removeAt removes the item in the given position from the list.
func ListRemoveAt(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := int(n)
	target.Lock()
	l := target.Value.([]*Object)
	if k < 0 || k >= len(l) {
		target.Unlock()
		return vm.RaiseExceptionf("index %d out of bounds", k)
	}
	v := l[k]
	copy(l[k:], l[k+1:])
	target.Value = l[:len(l)-1]
	target.Unlock()
	return v
}

// ListReverseForeach is a List method.
//
// reverseForeach performs a loop on each item of a list in order, optionally
// setting index and value variables, proceeding from the end of the list to
// the start.
func ListReverseForeach(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseExceptionf("reverseForeach requires 2 or 3 arguments")
	}
	target.Lock()
	l := target.Value.([]*Object)
	var control Stop
	for k := len(l) - 1; k >= 0; k-- {
		v := l[k]
		target.Unlock()
		vm.SetSlot(locals, vn, v)
		if hkn {
			vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
		}
		result, control = ev.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop, ExitStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
		target.Lock()
		l = target.Value.([]*Object)
	}
	target.Unlock()
	return result
}

// ListReverseInPlace is a List method.
//
// reverseInPlace reverses the order of items in the list.
func ListReverseInPlace(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	ll := len(l)
	for i := 0; i < ll/2; i++ {
		l[i], l[ll-1-i] = l[ll-1-i], l[i]
	}
	target.Unlock()
	return target
}

// ListSetSize is a List method.
//
// setSize changes the size of the list, removing items from or adding nils to
// the end as necessary.
func ListSetSize(vm *VM, target, locals *Object, msg *Message) *Object {
	v, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	n := int(v)
	target.Lock()
	l := target.Value.([]*Object)
	if n <= len(l) {
		target.Value = l[:n]
	} else {
		ll := len(l)
		nn := n - ll
		target.Value = append(l, make([]*Object, nn)...)
		for i := ll; i < len(l); i++ {
			l[i] = vm.Nil
		}
	}
	target.Unlock()
	return target
}

// ListSize is a List method.
//
// size is the number of items in the list.
func ListSize(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	n := len(target.Value.([]*Object))
	target.Unlock()
	return vm.NewNumber(float64(n))
}

// SliceArgs gets start, stop, and step values for a standard slice-like
// method invocation, which may be any of the following:
//
// 	slice(start)
// 	slice(start, stop)
// 	slice(start, stop, step)
//
// start and stop are fixed in the following sense: for each, if it is less
// than zero, then size is added to it, then, if it is still less than zero,
// it becomes -1 if the step is negative and 0 otherwise; if it is greater than
// or equal to the size, then it becomes size - 1 if step is negative and size
// otherwise.
func SliceArgs(vm *VM, locals *Object, msg *Message, size int) (start, step, stop int, exc *Object, control Stop) {
	start = 0
	step = 1
	stop = size
	n := msg.ArgCount()
	x, exc, control := msg.NumberArgAt(vm, locals, 0)
	if control != NoStop {
		return
	}
	start = int(x)
	if n >= 2 {
		x, exc, control = msg.NumberArgAt(vm, locals, 1)
		if control != NoStop {
			return
		}
		stop = int(x)
		if n >= 3 {
			x, exc, control = msg.NumberArgAt(vm, locals, 2)
			if control != NoStop {
				return
			}
			step = int(x)
			if step == 0 {
				exc, control = vm.NewExceptionf("slice step cannot be zero"), ExceptionStop
				return
			}
		}
	}
	start = fixSliceIndex(start, step, size)
	stop = fixSliceIndex(stop, step, size)
	return
}

func fixSliceIndex(k, step, size int) int {
	if k < 0 {
		k += size
		if k < 0 {
			if step < 0 {
				return -1
			}
			return 0
		}
	} else if k >= size {
		if step < 0 {
			return size - 1
		}
		return size
	}
	return k
}

// ListSlice is a List method.
//
// slice returns a selected linear portion of the list.
func ListSlice(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	start, step, stop, exc, control := SliceArgs(vm, locals, msg, len(l))
	if control != NoStop {
		return vm.Stop(exc, control)
	}
	nn := 0
	if step > 0 {
		nn = (stop - start + step - 1) / step
	} else {
		nn = (stop - start + step + 1) / step
	}
	if nn <= 0 {
		return vm.NewList()
	}
	v := make([]*Object, 0, nn)
	target.Lock()
	// Don't fetch an updated list value in case the slice args become invalid.
	if step > 0 {
		for start < stop {
			v = append(v, l[start])
			start += step
		}
	} else {
		for start > stop {
			v = append(v, l[start])
			start += step
		}
	}
	target.Unlock()
	return vm.NewList(v...)
}

// ListSliceInPlace is a List method.
//
// sliceInPlace reduces the list to a selected linear portion.
func ListSliceInPlace(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	start, step, stop, exc, control := SliceArgs(vm, locals, msg, len(l))
	if control != NoStop {
		return vm.Stop(exc, control)
	}
	nn := 0
	if step > 0 {
		nn = (stop - start + step - 1) / step
	} else {
		nn = (stop - start + step + 1) / step
	}
	target.Lock()
	if nn <= 0 {
		target.Value = l[:0]
		target.Unlock()
		return target
	}
	if step > 0 {
		for j := 0; start < stop; j++ {
			l[j] = l[start]
			start += step
		}
	} else {
		// Swap items between the input and output cursors until they pass each
		// other, and then use the location to which a value would have been
		// swapped as the input.
		i, j := start, 0
		for i > j && i > stop {
			l[j], l[i] = l[i], l[j]
			i += step
			j++
		}
		for i > stop {
			l[j] = l[start+i*step]
			i += step
			j++
		}
	}
	target.Value = l[:nn]
	target.Unlock()
	return target
}

type listSorter struct {
	mu  *sync.Mutex // list's mutex
	v   []*Object   // values to sort
	vm  *VM         // VM to use for compare, &c.
	e   *Message    // message to send to items, arg of sortInPlace
	l   *Object     // locals for message send or sortInPlaceBy block
	b   *Object     // block to use in place of compare, arg of sortInPlaceBy
	m   *Message    // message to hold arguments to sortInPlaceBy block
	err *Object     // error during compare
	c   Stop        // control flow type during compare
}

func (l *listSorter) Len() int {
	return len(l.v)
}

func (l *listSorter) Swap(i, j int) {
	l.mu.Lock()
	l.v[i], l.v[j] = l.v[j], l.v[i]
	l.mu.Unlock()
}

func (l *listSorter) Less(i, j int) bool {
	if l.c != NoStop {
		// If an error has occurred, treat the list as already sorted.
		return i < j
	}
	if l.b == nil {
		l.mu.Lock()
		a, b := l.v[i], l.v[j]
		l.mu.Unlock()
		var stop Stop
		if l.e != nil {
			a, stop = l.e.Send(l.vm, a, l.l)
			if stop != NoStop {
				l.err, l.c = a, stop
				return i < j
			}
			b, stop = l.e.Send(l.vm, b, l.l)
			if stop != NoStop {
				l.err, l.c = b, stop
				return i < j
			}
		}
		r, obj, stop := l.vm.Compare(a, b)
		if stop != NoStop {
			l.err, l.c = obj, stop
			return i < j
		}
		if obj == nil {
			return r < 0
		}
		return l.vm.AsBool(obj)
	}
	l.mu.Lock()
	l.m.Args[0].Memo, l.m.Args[1].Memo = l.v[i], l.v[j]
	l.mu.Unlock()
	r := l.vm.ActivateBlock(l.b, l.l, l.l, l.l, l.m)
	// Check whether the block gave us any control flow signal.
	if l.err, l.c = l.vm.Status(nil); l.c != NoStop {
		return i < j
	}
	return l.vm.AsBool(r)
}

// ListSortInPlace is a List method.
//
// sortInPlace sorts the list according to the items' compare method.
func ListSortInPlace(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	ls := listSorter{
		mu: &target.Mutex,
		v:  l,
		vm: vm,
	}
	if msg.ArgCount() > 0 {
		ls.e = msg.ArgAt(0)
		ls.l = locals
	}
	sort.Sort(&ls)
	if ls.c != NoStop {
		return vm.Stop(ls.err, ls.c)
	}
	return target
}

// ListSortInPlaceBy is a List method.
//
// sortInPlaceBy sorts the list using a given compare block.
func ListSortInPlaceBy(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	if r.Tag() != BlockTag {
		return vm.RaiseExceptionf("argument 0 to List sortInPlaceBy must be Block, not %s", vm.TypeName(r))
	}
	target.Lock()
	l := target.Value.([]*Object)
	target.Unlock()
	ls := listSorter{
		mu: &target.Mutex,
		v:  l,
		vm: vm,
		l:  locals,
		b:  r,
		m:  vm.IdentMessage("", vm.IdentMessage(""), vm.IdentMessage("")),
	}
	sort.Sort(&ls)
	if ls.c != NoStop {
		return vm.Stop(ls.err, ls.c)
	}
	return target
}

// ListSwapIndices is a List method.
//
// swapIndices swaps the values in two positions in the list.
func ListSwapIndices(vm *VM, target, locals *Object, msg *Message) *Object {
	a, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	b, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	i, j := int(a), int(b)
	target.Lock()
	l := target.Value.([]*Object)
	if i < 0 || i >= len(l) || j < 0 || j >= len(l) {
		target.Unlock()
		return vm.RaiseExceptionf("index out of bounds")
	}
	l[i], l[j] = l[j], l[i]
	target.Unlock()
	return target
}

// ListWith is a List method.
//
// with creates a new list with the given values as items.
func ListWith(vm *VM, target, locals *Object, msg *Message) *Object {
	v := make([]*Object, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		v = append(v, r)
	}
	return vm.NewList(v...)
}
