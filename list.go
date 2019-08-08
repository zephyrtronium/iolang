package iolang

import (
	"fmt"
	"sort"
	"strings"
)

// A List is a linear collection of arbitrary items.
type List struct {
	Object
	Value []Interface
}

// NewList creates a List with the given items.
func (vm *VM) NewList(items ...Interface) *List {
	return &List{
		*vm.CoreInstance("List"),
		items,
	}
}

// ListArgAt evaluates the nth argument and returns it as a List. If a stop
// occurs during evaluation, the List will be nil, and the stop status and
// result will be returned. If the evaluated result is not a List, the result
// will be nil, and an exception will be raised.
func (m *Message) ListArgAt(vm *VM, locals Interface, n int) (*List, Interface, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		if lst, ok := v.(*List); ok {
			return lst, nil, NoStop
		}
		// Not the expected type, so return an error.
		v, s = vm.RaiseExceptionf("argument %d to %s must be List, not %s", n, m.Text, vm.TypeName(v))
	}
	return nil, v, s
}

// Activate returns the list.
func (l *List) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return l, NoStop
}

// Clone creates a clone of this list and copies this list's values into it.
func (l *List) Clone() Interface {
	ll := make([]Interface, len(l.Value))
	copy(ll, l.Value)
	return &List{Object{Slots: Slots{}, Protos: []Interface{l}}, ll}
}

// String creates a simple string representation of this list.
func (l *List) String() string {
	return fmt.Sprintf("[list with %d items]", len(l.Value))
}

// initList initializes List on this VM.
func (vm *VM) initList() {
	var kind *List
	slots := Slots{
		"append":              vm.NewCFunction(ListAppend, kind),
		"appendIfAbsent":      vm.NewCFunction(ListAppendIfAbsent, kind),
		"appendSeq":           vm.NewCFunction(ListAppendSeq, kind),
		"asString":            vm.NewCFunction(ListAsString, kind),
		"at":                  vm.NewCFunction(ListAt, kind),
		"atInsert":            vm.NewCFunction(ListAtInsert, kind),
		"atPut":               vm.NewCFunction(ListAtPut, kind),
		"capacity":            vm.NewCFunction(ListCapacity, kind),
		"compare":             vm.NewCFunction(ListCompare, kind),
		"contains":            vm.NewCFunction(ListContains, kind),
		"containsAll":         vm.NewCFunction(ListContainsAll, kind),
		"containsAny":         vm.NewCFunction(ListContainsAny, kind),
		"containsIdenticalTo": vm.NewCFunction(ListContainsIdenticalTo, kind),
		"foreach":             vm.NewCFunction(ListForeach, kind),
		"indexOf":             vm.NewCFunction(ListIndexOf, kind),
		"preallocateToSize":   vm.NewCFunction(ListPreallocateToSize, kind),
		"prepend":             vm.NewCFunction(ListPrepend, kind),
		"remove":              vm.NewCFunction(ListRemove, kind),
		"removeAll":           vm.NewCFunction(ListRemoveAll, kind),
		"removeAt":            vm.NewCFunction(ListRemoveAt, kind),
		"reverseForeach":      vm.NewCFunction(ListReverseForeach, kind),
		"reverseInPlace":      vm.NewCFunction(ListReverseInPlace, kind),
		"setSize":             vm.NewCFunction(ListSetSize, kind),
		"size":                vm.NewCFunction(ListSize, kind),
		"slice":               vm.NewCFunction(ListSlice, kind),
		"sliceInPlace":        vm.NewCFunction(ListSliceInPlace, kind),
		"sortInPlace":         vm.NewCFunction(ListSortInPlace, kind),
		"sortInPlaceBy":       vm.NewCFunction(ListSortInPlaceBy, kind),
		"swapIndices":         vm.NewCFunction(ListSwapIndices, kind),
		"type":                vm.NewString("List"),
		"with":                vm.NewCFunction(ListWith, nil),
	}
	slots["empty"] = slots["removeAll"]
	slots["exSlice"] = slots["slice"]
	slots["push"] = slots["append"]
	vm.SetSlot(vm.Core, "List", &List{Object: *vm.ObjectWith(slots)})
	vm.SetSlot(vm.BaseObject, "list", slots["with"])
}

// ListAppend is a List method.
//
// append adds items to the end of the list.
func ListAppend(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return r, stop
		}
		l.Value = append(l.Value, r)
	}
	return target, NoStop
}

// ListAppendIfAbsent is a List method.
//
// appendIfAbsent adds items to the end of the list if they are not already in
// it.
func ListAppendIfAbsent(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
outer:
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return r, stop
		}
		for _, v := range l.Value {
			c, stop := vm.Compare(v, r)
			if stop != NoStop {
				return c, stop
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
				continue outer
			}
		}
		l.Value = append(l.Value, r)
	}
	return target, NoStop
}

// ListAppendSeq is a List method.
//
// appendSeq adds the items in the given lists to the list.
func ListAppendSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	for _, m := range msg.Args {
		v, stop := m.Eval(vm, locals)
		if stop != NoStop {
			return v, stop
		}
		if r, ok := v.(*List); ok {
			if r == l {
				return vm.RaiseException("can't add a list to itself")
			}
			l.Value = append(l.Value, r.Value...)
		} else {
			return vm.RaiseExceptionf("all arguments to %s must be lists, not %s", msg.Name(), vm.TypeName(v))
		}
	}
	return target, NoStop
}

// ListAsString is a List method.
//
// asString creates a string representation of an object.
func ListAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	b := strings.Builder{}
	b.WriteString("list(")
	for i, v := range l.Value {
		b.WriteString(vm.AsString(v))
		if i != len(l.Value)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(")")
	return vm.NewString(b.String()), NoStop
}

// ListAt is a List method.
//
// at returns the nth item in the list. All out-of-bounds values are nil.
func ListAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	l := target.(*List)
	k := int(n.Value)
	if k < 0 || k >= len(l.Value) {
		return vm.Nil, NoStop
	}
	return l.Value[k], NoStop
}

// ListAtInsert is a List method.
//
// atInsert adds an item to the list at the given position, moving back
// existing items at or past that point.
func ListAtInsert(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return r, stop
	}
	l := target.(*List)
	k := int(n.Value)
	switch {
	case k < 0 || k > len(l.Value):
		return vm.RaiseException("index out of bounds")
	case k == len(l.Value):
		l.Value = append(l.Value, r)
	default:
		// Make space for the new item, then copy items after its new location
		// up a spot.
		l.Value = append(l.Value, nil)
		copy(l.Value[k+1:], l.Value[k:])
		l.Value[k] = r
	}
	return target, NoStop
}

// ListAtPut is a List method.
//
// atPut replaces an item in the list.
func ListAtPut(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return r, stop
	}
	l := target.(*List)
	k := int(n.Value)
	if k < 0 || k >= len(l.Value) {
		return vm.RaiseException("index out of bounds")
	}
	l.Value[k] = r
	return target, NoStop
}

// ListCapacity is a List method.
//
// capacity is the number of items for which the list has allocated space.
func ListCapacity(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(cap(target.(*List).Value))), NoStop
}

// ListCompare is a List method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func ListCompare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	if r, ok := v.(*List); ok {
		s1, s2 := len(l.Value), len(r.Value)
		if s1 != s2 {
			// This is not proper lexicographical order, but it is Io's order.
			if s1 < s2 {
				return vm.NewNumber(-1), NoStop
			}
			return vm.NewNumber(1), NoStop
		}
		for i, v := range l.Value {
			x, stop := vm.Compare(v, r.Value[i])
			if stop != NoStop {
				return x, stop
			}
			if n, ok := x.(*Number); ok && n.Value != 0 {
				return n, NoStop
			}
		}
		return vm.NewNumber(0), NoStop
	}
	return vm.NewNumber(float64(PtrCompare(l, v))), NoStop
}

// ListContains is a List method.
//
// contains returns true if the list contains an item equal to the given
// object.
func ListContains(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	for _, v := range l.Value {
		c, stop := vm.Compare(v, r)
		if stop != NoStop {
			return c, stop
		}
		if n, ok := c.(*Number); ok && n.Value == 0 {
			return vm.True, NoStop
		}
	}
	return vm.False, NoStop
}

// ListContainsAll is a List method.
//
// containsAll returns true if the list contains items equal to each of the
// given objects.
func ListContainsAll(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r := make([]Interface, len(msg.Args))
	var stop Stop
	for i := range msg.Args {
		r[i], stop = msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return r[i], stop
		}
	}
outer:
	for _, v := range r {
		for _, ri := range l.Value {
			c, stop := vm.Compare(ri, v)
			if stop != NoStop {
				return c, stop
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
				continue outer
			}
		}
		return vm.False, NoStop
	}
	return vm.True, NoStop
}

// ListContainsAny is a List method.
//
// containsAny returns true if the list contains an item equal to any of the
// given objects.
func ListContainsAny(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	// TODO: use ID checks like ListRemove does
	r := make([]Interface, len(msg.Args))
	var stop Stop
	for i := range msg.Args {
		r[i], stop = msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return r[i], stop
		}
	}
	for _, v := range l.Value {
		for _, ri := range r {
			c, stop := vm.Compare(ri, v)
			if stop != NoStop {
				return c, stop
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
				return vm.True, NoStop
			}
		}
	}
	return vm.False, NoStop
}

// ListContainsIdenticalTo is a List method.
//
// containsIdenticalTo returns true if the list contains exactly the given
// object.
func ListContainsIdenticalTo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	for _, v := range l.Value {
		if r == v {
			return vm.True, NoStop
		}
	}
	return vm.False, NoStop
}

// ListForeach is a List method.
//
// foreach performs a loop on each item of a list in order, optionally setting
// index and value variables.
func ListForeach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 1, 2, or 3 arguments")
	}
	l := target.(*List)
	for k, v := range l.Value {
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
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, control
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
	return result, NoStop
}

// ListIndexOf is a List method.
//
// indexOf returns the first index from the left of an item equal to the
// argument. If there is no such item in the list, nil is returned.
func ListIndexOf(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	for i, v := range l.Value {
		c, stop := vm.Compare(v, r)
		if stop != NoStop {
			return c, stop
		}
		if n, ok := c.(*Number); ok && n.Value == 0 {
			return vm.NewNumber(float64(i)), NoStop
		}
	}
	return vm.Nil, NoStop
}

// ListPreallocateToSize is a List method.
//
// preallocateToSize ensures that the list has capacity for at least n items.
func ListPreallocateToSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if r.Value < 0 {
		return vm.RaiseException("can't preallocate to negative size")
	}
	n := int(r.Value)
	if n > cap(l.Value) {
		v := make([]Interface, len(l.Value), n)
		copy(v, l.Value)
		l.Value = v
	}
	return target, NoStop
}

// ListPrepend is a List method.
//
// prepend adds items to the beginning of the list.
func ListPrepend(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	nv := make([]Interface, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop == NoStop {
			nv = append(nv, r)
		} else {
			return r, stop
		}
	}
	// We want to make sure that we use our existing space if we can.
	n := len(nv) + len(l.Value)
	if cap(l.Value) >= n {
		l.Value = l.Value[:n]
		copy(l.Value[len(nv):], l.Value)
		copy(l.Value, nv)
	} else {
		l.Value = append(nv, l.Value...)
	}
	return target, NoStop
}

// ListRemove is a List method.
//
// remove removes all occurrences of each item from the list.
func ListRemove(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	rv := make(map[Interface]struct{}, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop == NoStop {
			rv[r] = struct{}{}
		} else {
			return r, stop
		}
	}
	j := 0
	defer func() { l.Value = l.Value[:len(l.Value)-j] }()
outer:
	for k, v := range l.Value {
		// Check whether the value exists by ID first, then check via compare.
		if _, ok := rv[v]; ok {
			j++
		} else {
			for r := range rv {
				c, stop := vm.Compare(v, r)
				if stop != NoStop {
					return c, stop
				}
				if n, ok := c.(*Number); ok && n.Value == 0 {
					j++
					continue outer
				}
			}
			if k-j >= 0 {
				l.Value[k-j] = v
			}
		}
	}
	return target, NoStop
}

// ListRemoveAll is a List method.
//
// removeAll removes all items from the list.
func ListRemoveAll(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	l.Value = l.Value[:0]
	return target, NoStop
}

// ListRemoveAt is a List method.
//
// removeAt removes the item in the given position from the list.
func ListRemoveAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := int(n.Value)
	if k < 0 || k >= len(l.Value) {
		return vm.RaiseException("index out of bounds")
	}
	v := l.Value[k]
	copy(l.Value[k:], l.Value[k+1:])
	l.Value = l.Value[:len(l.Value)-1]
	return v, NoStop
}

// ListReverseForeach is a List method.
//
// reverseForeach performs a loop on each item of a list in order, optionally
// setting index and value variables, proceeding from the end of the list to
// the start.
func ListReverseForeach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseException("reverseForeach requires 2 or 3 arguments")
	}
	l := target.(*List)
	for k := len(l.Value) - 1; k >= 0; k-- {
		v := l.Value[k]
		vm.SetSlot(locals, vn, v)
		if hkn {
			vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
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

// ListReverseInPlace is a List method.
//
// reverseInPlace reverses the order of items in the list.
func ListReverseInPlace(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	ll := len(l.Value)
	for i := 0; i < ll/2; i++ {
		l.Value[i], l.Value[ll-1-i] = l.Value[ll-1-i], l.Value[i]
	}
	return target, NoStop
}

// ListSetSize is a List method.
//
// setSize changes the size of the list, removing items from or adding nils to
// the end as necessary.
func ListSetSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n := int(v.Value)
	if n <= len(l.Value) {
		l.Value = l.Value[:n]
	} else {
		ll := len(l.Value)
		nn := n - ll
		l.Value = append(l.Value, make([]Interface, nn)...)
		for i := ll; i < len(l.Value); i++ {
			l.Value[i] = vm.Nil
		}
	}
	return target, NoStop
}

// ListSize is a List method.
//
// size is the number of items in the list.
func ListSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(len(target.(*List).Value))), NoStop
}

// SliceArgs gets start, stop, and step values for a standard slice-like
// method invocation, which may be any of the following:
//
// 	slice(start)
// 	slice(start, stop)
// 	slice(start, stop, step)
func SliceArgs(vm *VM, locals Interface, msg *Message, size int) (start, step, stop int, err Interface, control Stop) {
	start = 0
	step = 1
	stop = size
	n := msg.ArgCount()
	x, err, control := msg.NumberArgAt(vm, locals, 0)
	if control != NoStop {
		return
	}
	start = int(x.Value)
	if n >= 2 {
		x, err, control = msg.NumberArgAt(vm, locals, 1)
		if control != NoStop {
			return
		}
		stop = int(x.Value)
		if n >= 3 {
			x, err, control = msg.NumberArgAt(vm, locals, 2)
			if control != NoStop {
				return
			}
			step = int(x.Value)
			if step == 0 {
				err, control = vm.RaiseException("slice step cannot be zero")
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
func ListSlice(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	start, step, stop, err, control := SliceArgs(vm, locals, msg, len(l.Value))
	if control != NoStop {
		return err, control
	}
	nn := 0
	if step > 0 {
		nn = (stop - start + step - 1) / step
	} else {
		nn = (stop - start + step + 1) / step
	}
	if nn <= 0 {
		return vm.NewList(), NoStop
	}
	v := make([]Interface, 0, nn)
	if step > 0 {
		for start < stop {
			v = append(v, l.Value[start])
			start += step
		}
	} else {
		for start > stop {
			v = append(v, l.Value[start])
			start += step
		}
	}
	return vm.NewList(v...), NoStop
}

// ListSliceInPlace is a List method.
//
// sliceInPlace reduces the list to a selected linear portion.
func ListSliceInPlace(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	start, step, stop, err, control := SliceArgs(vm, locals, msg, len(l.Value))
	if control != NoStop {
		return err, control
	}
	nn := 0
	if step > 0 {
		nn = (stop - start + step - 1) / step
	} else {
		nn = (stop - start + step + 1) / step
	}
	if nn <= 0 {
		l.Value = l.Value[:0]
		return target, NoStop
	}
	if step > 0 {
		for j := 0; start < stop; j++ {
			l.Value[j] = l.Value[start]
			start += step
		}
	} else {
		// Swap items between the input and output cursors until they pass each
		// other, and then use the location to which a value would have been
		// swapped as the input.
		i, j := start, 0
		for i > j && i > stop {
			l.Value[j], l.Value[i] = l.Value[i], l.Value[j]
			i += step
			j++
		}
		for i > stop {
			l.Value[j] = l.Value[start+i*step]
			i += step
			j++
		}
	}
	l.Value = l.Value[:nn]
	return target, NoStop
}

type listSorter struct {
	v   []Interface // values to sort
	vm  *VM         // VM to use for compare, &c.
	e   *Message    // message to send to items, arg of sortInPlace
	l   Interface   // locals for message send or sortInPlaceBy block
	b   *Block      // block to use in place of compare, arg of sortInPlaceBy
	m   *Message    // message to hold arguments to sortInPlaceBy block
	err Interface   // error during compare
	c   Stop        // control flow type during compare
}

func (l *listSorter) Len() int {
	return len(l.v)
}

func (l *listSorter) Swap(i, j int) {
	l.v[i], l.v[j] = l.v[j], l.v[i]
}

func (l *listSorter) Less(i, j int) bool {
	if l.c != NoStop {
		// If an error has occurred, treat the list as already sorted.
		return i < j
	}
	if l.b == nil {
		a, b := l.v[i], l.v[j]
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
		r, stop := l.vm.Compare(a, b)
		if stop != NoStop {
			l.err, l.c = r, stop
			return i < j
		}
		if n, ok := r.(*Number); ok {
			return n.Value < 0
		}
		return l.vm.AsBool(r)
	}
	l.m.Args[0].Memo, l.m.Args[1].Memo = l.v[i], l.v[j]
	r, stop := l.b.reallyActivate(l.vm, l.l, l.l, l.l, l.m)
	if stop != NoStop {
		l.err, l.c = r, stop
		return i < j
	}
	return l.vm.AsBool(r)
}

// ListSortInPlace is a List method.
//
// sortInPlace sorts the list according to the items' compare method.
func ListSortInPlace(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	ls := listSorter{
		v:  l.Value,
		vm: vm,
	}
	if len(msg.Args) > 0 {
		ls.e = msg.Args[0]
		ls.l = locals
	}
	sort.Sort(&ls)
	if ls.c != NoStop {
		return ls.err, ls.c
	}
	return target, NoStop
}

// ListSortInPlaceBy is a List method.
//
// sortInPlaceBy sorts the list using a given compare block.
func ListSortInPlaceBy(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	b, ok := r.(*Block)
	if !ok {
		return vm.RaiseException("argument 0 to List sortInPlaceBy must be Block, not " + vm.TypeName(b))
	}
	ls := listSorter{
		v:  l.Value,
		vm: vm,
		l:  locals,
		b:  b,
		m:  vm.IdentMessage("", vm.IdentMessage(""), vm.IdentMessage("")),
	}
	sort.Sort(&ls)
	if ls.c != NoStop {
		return ls.err, ls.c
	}
	return target, NoStop
}

// ListSwapIndices is a List method.
//
// swapIndices swaps the values in two positions in the list.
func ListSwapIndices(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l := target.(*List)
	a, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	b, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	i, j := int(a.Value), int(b.Value)
	if i < 0 || i >= len(l.Value) || j < 0 || j >= len(l.Value) {
		return vm.RaiseException("index out of bounds")
	}
	l.Value[i], l.Value[j] = l.Value[j], l.Value[i]
	return target, NoStop
}

// ListWith is a List method.
//
// with creates a new list with the given values as items.
func ListWith(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v := make([]Interface, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, stop := m.Eval(vm, locals)
		if stop == NoStop {
			v = append(v, r)
		} else {
			return r, stop
		}
	}
	return vm.NewList(v...), NoStop
}
