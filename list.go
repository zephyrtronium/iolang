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
		"append":              vm.NewTypedCFunction(ListAppend),
		"appendIfAbsent":      vm.NewTypedCFunction(ListAppendIfAbsent),
		"appendSeq":           vm.NewTypedCFunction(ListAppendSeq),
		"asString":            vm.NewTypedCFunction(ListAsString),
		"at":                  vm.NewTypedCFunction(ListAt),
		"atInsert":            vm.NewTypedCFunction(ListAtInsert),
		"atPut":               vm.NewTypedCFunction(ListAtPut),
		"capacity":            vm.NewTypedCFunction(ListCapacity),
		"compare":             vm.NewTypedCFunction(ListCompare),
		"contains":            vm.NewTypedCFunction(ListContains),
		"containsAll":         vm.NewTypedCFunction(ListContainsAll),
		"containsAny":         vm.NewTypedCFunction(ListContainsAny),
		"containsIdenticalTo": vm.NewTypedCFunction(ListContainsIdenticalTo),
		"foreach":             vm.NewTypedCFunction(ListForeach),
		"indexOf":             vm.NewTypedCFunction(ListIndexOf),
		"preallocateToSize":   vm.NewTypedCFunction(ListPreallocateToSize),
		"prepend":             vm.NewTypedCFunction(ListPrepend),
		"remove":              vm.NewTypedCFunction(ListRemove),
		"removeAll":           vm.NewTypedCFunction(ListRemoveAll),
		"removeAt":            vm.NewTypedCFunction(ListRemoveAt),
		"reverseForeach":      vm.NewTypedCFunction(ListReverseForeach),
		"reverseInPlace":      vm.NewTypedCFunction(ListReverseInPlace),
		"setSize":             vm.NewTypedCFunction(ListSetSize),
		"size":                vm.NewTypedCFunction(ListSize),
		"slice":               vm.NewTypedCFunction(ListSlice),
		"sliceInPlace":        vm.NewTypedCFunction(ListSliceInPlace),
		"sortInPlace":         vm.NewTypedCFunction(ListSortInPlace),
		"sortInPlaceBy":       vm.NewTypedCFunction(ListSortInPlaceBy),
		"swapIndices":         vm.NewTypedCFunction(ListSwapIndices),
		"type":                vm.NewString("List"),
		"with":                vm.NewCFunction(ListWith),
	}
	SetSlot(vm.Core, "List", &List{Object: *vm.ObjectWith(slots)})
	SetSlot(vm.BaseObject, "list", slots["with"])
}

// ListAppend is a List method.
//
// append adds items to the end of the list.
func ListAppend(vm *VM, target, locals Interface, msg *Message) Interface {
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
			c, ok := CheckStop(vm.Compare(v, r), ReturnStop)
			if !ok {
				return c
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
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
			return vm.RaiseExceptionf("all arguments to %s must be lists, not %s", msg.Name(), vm.TypeName(v))
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
			if n, ok := x.(*Number); ok && n.Value != 0 {
				return n
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
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	for _, v := range target.(*List).Value {
		c, ok := CheckStop(vm.Compare(v, r), ReturnStop)
		if !ok {
			return c
		}
		if n, ok := c.(*Number); ok && n.Value == 0 {
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
	r := make([]Interface, len(msg.Args))
	var ok bool
	for i := range msg.Args {
		r[i], ok = CheckStop(msg.EvalArgAt(vm, locals, i), LoopStops)
		if !ok {
			return r[i]
		}
	}
outer:
	for _, v := range r {
		for _, ri := range target.(*List).Value {
			c, ok := CheckStop(vm.Compare(ri, v), ReturnStop)
			if !ok {
				return c
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
				continue outer
			}
		}
		return vm.False
	}
	return vm.True
}

// ListContainsAny is a List method.
//
// containsAny returns true if the list contains an item equal to any of the
// given objects.
func ListContainsAny(vm *VM, target, locals Interface, msg *Message) Interface {
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
			c, ok := CheckStop(vm.Compare(ri, v), ReturnStop)
			if !ok {
				return c
			}
			if n, ok := c.(*Number); ok && n.Value == 0 {
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
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	for i, v := range target.(*List).Value {
		c, ok := CheckStop(vm.Compare(v, r), ReturnStop)
		if !ok {
			return c
		}
		if n, ok := c.(*Number); ok && n.Value == 0 {
			return vm.NewNumber(float64(i))
		}
	}
	return vm.NewNumber(-1)
}

// ListPreallocateToSize is a List method.
//
// preallocateToSize ensures that the list has capacity for at least n items.
func ListPreallocateToSize(vm *VM, target, locals Interface, msg *Message) Interface {
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
	defer func() { l.Value = l.Value[:len(l.Value)-j] }()
outer:
	for k, v := range l.Value {
		// Check whether the value exists by ID first, then check via compare.
		if _, ok := rv[v]; ok {
			j++
		} else {
			// TODO: use Io comparison
			// reminder: vvv is an else branch if Io comparison didn't match.
			for r := range rv {
				c, ok := CheckStop(vm.Compare(v, r), ReturnStop)
				if !ok {
					return c
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
	return target
}

// ListRemoveAll is a List method.
//
// removeAll removes all items from the list.
func ListRemoveAll(vm *VM, target, locals Interface, msg *Message) Interface {
	l := target.(*List)
	l.Value = l.Value[:0]
	return target
}

// ListRemoveAt is a List method.
//
// removeAt removes the item in the given position from the list.
func ListRemoveAt(vm *VM, target, locals Interface, msg *Message) Interface {
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

type listSorter struct {
	v   []Interface // values to sort
	vm  *VM         // VM to use for compare, &c.
	e   *Message    // message to send to items, arg of sortInPlace
	l   Interface   // locals for message send or sortInPlaceBy block
	b   *Block      // block to use in place of compare, arg of sortInPlaceBy
	m   *Message    // message to hold arguments to sortInPlaceBy block
	err Interface   // error during compare
}

func (l *listSorter) Len() int {
	return len(l.v)
}

func (l *listSorter) Swap(i, j int) {
	l.v[i], l.v[j] = l.v[j], l.v[i]
}

func (l *listSorter) Less(i, j int) bool {
	if l.err != nil {
		// If an error has occurred, treat the list as already sorted.
		return i < j
	}
	if l.b == nil {
		a, b := l.v[i], l.v[j]
		var ok bool
		if l.e != nil {
			a, ok = CheckStop(l.e.Send(l.vm, a, l.l), ReturnStop)
			if !ok {
				l.err = a
				return i < j
			}
			b, ok = CheckStop(l.e.Send(l.vm, b, l.l), ReturnStop)
			if !ok {
				l.err = b
				return i < j
			}
		}
		r, ok := CheckStop(l.vm.Compare(a, b), ReturnStop)
		if !ok {
			l.err = r
			return i < j
		}
		if n, ok := r.(*Number); ok {
			return n.Value < 0
		}
		return l.vm.AsBool(r)
	}
	l.m.Args[0].Memo, l.m.Args[1].Memo = l.v[i], l.v[j]
	r, ok := CheckStop(l.b.reallyActivate(l.vm, l.l, l.l, l.m), ReturnStop)
	if !ok {
		l.err = r
		return i < j
	}
	return l.vm.AsBool(r)
}

// ListSortInPlace is a List method.
//
// sortInPlace sorts the list according to the items' compare method.
func ListSortInPlace(vm *VM, target, locals Interface, msg *Message) Interface {
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
	if ls.err != nil {
		return ls.err
	}
	return target
}

// ListSortInPlaceBy is a List method.
//
// sortInPlaceBy sorts the list using a given compare block.
func ListSortInPlaceBy(vm *VM, target, locals Interface, msg *Message) Interface {
	l := target.(*List)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), ReturnStop)
	if !ok {
		return r
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
	if ls.err != nil {
		return ls.err
	}
	return target
}

// ListSwapIndices is a List method.
//
// swapIndices swaps the values in two positions in the list.
func ListSwapIndices(vm *VM, target, locals Interface, msg *Message) Interface {
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
