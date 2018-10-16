package iolang

import (
	"fmt"
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
	slots := Slots{
		"append":              vm.NewTypedCFunction(ListAppend, "ListAppend(v, ...)"),
		"appendIfAbsent":      vm.NewTypedCFunction(ListAppendIfAbsent, "ListAppendIfAbsent(v, ...)"),
		"appendSeq":           vm.NewTypedCFunction(ListAppendSeq, "ListAppendSeq(v, ...)"),
		"asString":            vm.NewTypedCFunction(ListAsString, "ListAsString()"),
		"at":                  vm.NewTypedCFunction(ListAt, "ListAt(k)"),
		"atInsert":            vm.NewTypedCFunction(ListAtInsert, "ListAtInsert(k, v)"),
		"atPut":               vm.NewTypedCFunction(ListAtPut, "ListAtPut(k, v)"),
		"capacity":            vm.NewTypedCFunction(ListCapacity, "ListCapacity()"),
		"compare":             vm.NewTypedCFunction(ListCompare, "ListCompare(v)"),
		"contains":            vm.NewTypedCFunction(ListContains, "ListContains(a)"),
		"containsAll":         vm.NewTypedCFunction(ListContainsAll, "ListContainsAll(a, b, ...)"),
		"containsAny":         vm.NewTypedCFunction(ListContainsAny, "ListContainsAny(a, b, ...)"),
		"containsIdenticalTo": vm.NewTypedCFunction(ListContainsIdenticalTo, "ListContainsIdenticalTo(a)"),
		"foreach":             vm.NewTypedCFunction(ListForeach, "ListForeach([[k, ]v, ]m"),
		"indexOf":             vm.NewTypedCFunction(ListIndexOf, "ListIndexOf(v)"),
		"preallocateToSize":   vm.NewTypedCFunction(ListPreallocateToSize, "ListPreallocateToSize(n)"),
		"prepend":             vm.NewTypedCFunction(ListPrepend, "ListPrepend(v)"),
		"remove":              vm.NewTypedCFunction(ListRemove, "ListRemove(a, b, ...)"),
		"removeAll":           vm.NewTypedCFunction(ListRemoveAll, "ListRemoveAll()"),
		"removeAt":            vm.NewTypedCFunction(ListRemoveAt, "ListRemoveAt(k)"),
		"reverseForeach":      vm.NewTypedCFunction(ListReverseForeach, "ListReverseForeach([[k, ]v, ]m"),
		"reverseInPlace":      vm.NewTypedCFunction(ListReverseInPlace, "ListReverseInPlace()"),
		"setSize":             vm.NewTypedCFunction(ListSetSize, "ListSetSize(n)"),
		"size":                vm.NewTypedCFunction(ListSize, "ListSize()"),
		"slice":               vm.NewTypedCFunction(ListSlice, "ListSlice(start[, step][, stop])"),
		"sliceInPlace":        vm.NewTypedCFunction(ListSliceInPlace, "ListSliceInPlace(start[, step][, stop])"),
		"swapIndices":         vm.NewTypedCFunction(ListSwapIndices, "ListSwapIndices(i, j)"),
		"type":                vm.NewString("List"),
		"with":                vm.NewCFunction(ListWith, "ListWith(a, b, ...)"),
	}
	SetSlot(vm.Core, "List", &List{Object: *vm.ObjectWith(slots)})
}

// ListAppend is a List method.
//
// append adds items to the end of the list.
func ListAppend(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	nv := make([]Interface, 0, len(msg.Args))
	defer func() {
		l.Value = append(l.Value, nv...)
	}()
	for _, m := range msg.Args {
		r, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if !ok {
			return r
		}
		nv = append(nv, r)
	}
	return target
}

// ListAppendIfAbsent is a List method.
//
// appendIfAbsent adds items to the end of the list if they are not already in
// it.
func ListAppendIfAbsent(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	nv := make([]Interface, 0, len(msg.Args))
	defer func() {
		l.Value = append(l.Value, nv...)
	}()
outer:
	for _, m := range msg.Args {
		r, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if !ok {
			return r
		}
		for _, v := range l.Value {
			// TODO: use Io comparison
			if r == v {
				continue outer
			}
		}
		nv = append(nv, r)
	}
	return target
}

// ListAppendSeq is a List method.
//
// appendSeq adds the items in the given lists to the list.
func ListAppendSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	nv := []Interface{}
	defer func() {
		l.Value = append(l.Value, nv...)
	}()
	for _, m := range msg.Args {
		v, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if !ok {
			return v
		}
		if r, ok := v.(*List); ok {
			if r == l {
				return vm.RaiseException("can't add a list to itself")
			}
			nv = append(nv, r.Value...)
		} else {
			return vm.RaiseExceptionf("all arguments to %s must be lists, not %s", msg.Symbol.Text, vm.TypeName(v))
		}
	}
	return target
}

// ListAsString is a List method.
//
// asString creates a string representation of an object.
func ListAsString(vm *VM, target, locals Interface, msg *Message) Interface {
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
	return vm.NewString(b.String())
}

// ListAt is a List method.
//
// at returns the nth item in the list. All out-of-bounds values are nil.
func ListAt(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	n, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	l := target.(*List)
	k := int(n.Value)
	if k < 0 || k >= len(l.Value) {
		return vm.Nil
	}
	return l.Value[k]
}

// ListAtInsert is a List method.
//
// atInsert adds an item to the list at the given position, moving back
// existing items at or past that point.
func ListAtInsert(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	n, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return r
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
	return target
}

// ListAtPut is a List method.
//
// atPut replaces an item in the list.
func ListAtPut(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	n, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return r
	}
	l := target.(*List)
	k := int(n.Value)
	if k < 0 || k >= len(l.Value) {
		return vm.RaiseException("index out of bounds")
	}
	l.Value[k] = r
	return target
}

// ListCapacity is a List method.
//
// capacity is the number of items for which the list has allocated space.
func ListCapacity(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	return vm.NewNumber(float64(cap(target.(*List).Value)))
}

// ListCompare is a List method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func ListCompare(vm *VM, target, locals Interface, msg *Message) Interface {
	l := target.(*List)
	v, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return v
	}
	if r, ok := v.(*List); ok {
		s1, s2 := len(l.Value), len(r.Value)
		if s1 != s2 {
			// This is not proper lexicographical order, but it is Io's order.
			if s1 < s2 {
				return vm.NewNumber(-1)
			}
			return vm.NewNumber(1)
		}
		for i, v := range l.Value {
			x, ok := CheckStop(vm.Compare(v, r.Value[i]), ReturnStop)
			if !ok {
				return x
			}
			if n, ok := x.(*Number); ok {
				if n.Value != 0 {
					return n
				}
			} else {
				return vm.RaiseExceptionf("compare returned non-number %s: %s", vm.TypeName(x), vm.AsString(x))
			}
		}
		return vm.NewNumber(0)
	}
	return vm.NewNumber(float64(ptrCompare(l, v)))
}

// ListContains is a List method.
//
// contains returns true if the list contains an item equal to the given
// object.
func ListContains(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	for _, v := range target.(*List).Value {
		// TODO: use Io comparison
		if r == v {
			return vm.True
		}
	}
	return vm.False
}

// ListContainsAll is a List method.
//
// containsAll returns true if the list contains items equal to each of the
// given objects.
func ListContainsAll(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	r := make([]Interface, len(msg.Args))
	var ok bool
	for i := range msg.Args {
		r[i], ok = CheckStop(msg.EvalArgAt(vm, locals, i), LoopStops)
		if !ok {
			return r[i]
		}
	}
	c := make([]bool, len(r))
	for _, v := range target.(*List).Value {
		for i, ri := range r {
			// TODO: use Io comparison
			if ri == v {
				c[i] = true
				break
			}
		}
	}
	for _, v := range c {
		if !v {
			return vm.False
		}
	}
	return vm.True
}

// ListContainsAny is a List method.
//
// containsAny returns true if the list contains an item equal to any of the
// given objects.
func ListContainsAny(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	// TODO: use ID checks like ListRemove does
	r := make([]Interface, len(msg.Args))
	var ok bool
	for i := range msg.Args {
		r[i], ok = CheckStop(msg.EvalArgAt(vm, locals, i), LoopStops)
		if !ok {
			return r[i]
		}
	}
	for _, v := range target.(*List).Value {
		for _, ri := range r {
			// TODO: use Io comparison
			if ri == v {
				return vm.True
			}
		}
	}
	return vm.False
}

// ListContainsIdenticalTo is a List method.
//
// containsIdenticalTo returns true if the list contains exactly the given
// object.
func ListContainsIdenticalTo(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	for _, v := range target.(*List).Value {
		if r == v {
			return vm.True
		}
	}
	return vm.False
}

// ListIndexOf is a List method.
//
// indexOf returns the first index from the left of an item equal to the
// argument. If there is no such item in the list, -1 is returned.
func ListIndexOf(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	for i, v := range target.(*List).Value {
		// TODO: use Io comparison
		if r == v {
			return vm.NewNumber(float64(i))
		}
	}
	return vm.NewNumber(-1)
}

// ListPreallocateToSize is a List method.
//
// preallocateToSize ensures that the list has capacity for at least n items.
func ListPreallocateToSize(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	r, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	if r.Value < 0 {
		return vm.RaiseException("can't preallocate to negative size")
	}
	n := int(r.Value)
	l := target.(*List)
	if n < cap(l.Value) {
		v := make([]Interface, len(l.Value), n)
		copy(v, l.Value)
		l.Value = v
	}
	return target
}

// ListPrepend is a List method.
//
// prepend adds items to the beginning of the list.
func ListPrepend(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	nv := make([]Interface, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if ok {
			nv = append(nv, r)
		} else {
			return r
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
	return target
}

// ListRemove is a List method.
//
// remove removes all occurrences of each item from the list.
func ListRemove(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	rv := make(map[Interface]struct{}, len(msg.Args))
	for _, m := range msg.Args {
		r, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if ok {
			rv[r] = struct{}{}
		} else {
			return r
		}
	}
	j := 0
	for k, v := range l.Value {
		// Check whether the value exists by ID first, then check via compare.
		if _, ok := rv[v]; ok {
			j++
		} else {
			// TODO: use Io comparison
			// reminder: vvv is an else branch if Io comparison didn't match.
			if k-j >= 0 {
				l.Value[k-j] = v
			}
		}
	}
	l.Value = l.Value[:len(l.Value)-j]
	return target
}

// ListRemoveAll is a List method.
//
// removeAll removes all items from the list.
func ListRemoveAll(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	l.Value = l.Value[:0]
	return target
}

// ListRemoveAt is a List method.
//
// removeAt removes the item in the given position from the list.
func ListRemoveAt(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	n, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	k := int(n.Value)
	copy(l.Value[k:], l.Value[k+1:])
	l.Value = l.Value[:len(l.Value)-1]
	return target
}

// ListReverseInPlace is a List method.
//
// reverseInPlace reverses the order of items in the list.
func ListReverseInPlace(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	ll := len(l.Value)
	for i := 0; i < ll/2; i++ {
		l.Value[i], l.Value[ll-1-i] = l.Value[ll-1-i], l.Value[i]
	}
	return target
}

// ListSetSize is a List method.
//
// setSize changes the size of the list, removing items from or adding nils to
// the end as necessary.
func ListSetSize(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	n := int(v.Value)
	if n <= len(l.Value) {
		l.Value = l.Value[:n]
	} else {
		ll := len(l.Value)
		nn := n - ll
		l.Value = append(l.Value, make([]Interface, nn)...)
		// It probably would be fine to leave the added items as Go nils, but
		// I'm more comfortable changing them to Io nils.
		for i := ll; i < len(l.Value); i++ {
			l.Value[i] = vm.Nil
		}
	}
	return target
}

// ListSize is a List method.
//
// size is the number of items in the list.
func ListSize(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	return vm.NewNumber(float64(len(target.(*List).Value)))
}

func sliceArgs(vm *VM, locals Interface, msg *Message, size int) (start, step, stop int, err error) {
	start = 0
	step = 1
	stop = size
	n := len(msg.Args)
	x, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return
	}
	start = int(x.Value)
	if n >= 2 {
		x, err = msg.NumberArgAt(vm, locals, 1)
		if err != nil {
			return
		}
		stop = int(x.Value)
		if n >= 3 {
			x, err = msg.NumberArgAt(vm, locals, 2)
			if err != nil {
				return
			}
			step = int(x.Value)
			if step == 0 {
				err = vm.NewException("slice step cannot be zero")
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
func ListSlice(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	start, step, stop, err := sliceArgs(vm, locals, msg, len(l.Value))
	if err != nil {
		return vm.IoError(err)
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
	return vm.NewList(v...)
}

// ListSliceInPlace is a List method.
//
// sliceInPlace reduces the list to a selected linear portion.
func ListSliceInPlace(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	start, step, stop, err := sliceArgs(vm, locals, msg, len(l.Value))
	if err != nil {
		return vm.IoError(err)
	}
	nn := 0
	if step > 0 {
		nn = (stop - start + step - 1) / step
	} else {
		nn = (stop - start + step + 1) / step
	}
	if nn <= 0 {
		l.Value = l.Value[:0]
		return target
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
	return target
}

// TODO: sortInPlace, sortInPlaceBy

// ListSwapIndices is a List method.
//
// swapIndices swaps the values in two positions in the list.
func ListSwapIndices(vm *VM, target, locals Interface, msg *Message) Interface {
	o := target.SP()
	o.L.Lock()
	defer o.L.Unlock()
	l := target.(*List)
	a, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	b, err := msg.NumberArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
	}
	i, j := int(a.Value), int(b.Value)
	l.Value[i], l.Value[j] = l.Value[j], l.Value[i]
	return target
}

// ListWith is a List method.
//
// with creates a new list with the given values as items.
func ListWith(vm *VM, target, locals Interface, msg *Message) Interface {
	v := make([]Interface, 0, len(msg.Args))
	for _, m := range msg.Args {
		r, ok := CheckStop(m.Eval(vm, locals), LoopStops)
		if ok {
			v = append(v, r)
		} else {
			return r
		}
	}
	return vm.NewList(v...)
}
