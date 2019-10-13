package iolang

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CheckMutable returns an error if the sequence is not mutable, or nil
// otherwise. Callers do not need to hold the object's lock, as mutability of a
// sequence should never change.
func (s Sequence) CheckMutable(name string) error {
	if s.IsMutable() {
		return nil
	}
	return fmt.Errorf("'%s' cannot be called on an immutable sequence", name)
}

// SequenceAsMutable is a Sequence method.
//
// asMutable creates a mutable copy of the sequence.
func SequenceAsMutable(vm *VM, target, locals Interface, msg *Message) *Object {
	// This isn't actually a mutable method, but it feels more appropriate
	// here with them.
	s := holdSeq(target)
	r := vm.NewSequence(s.Value, true, s.Code)
	unholdSeq(s.Mutable, target)
	return r
}

// SequenceAppend is a Sequence method.
//
// append adds numbers to the sequence.
func SequenceAppend(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("append"); err != nil {
		return vm.IoError(err)
	}
	v := make([]float64, msg.ArgCount())
	for i, arg := range msg.Args {
		r, stop := arg.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(r, stop)
		}
		n, ok := r.Value.(float64)
		if !ok {
			return vm.RaiseExceptionf("all arguments to append must be Number, not %s", vm.TypeName(r))
		}
		v[i] = n
	}
	switch m := s.Value.(type) {
	case []byte:
		for _, x := range v {
			m = append(m, byte(x))
		}
		s.Value = m
	case []uint16:
		for _, x := range v {
			m = append(m, uint16(x))
		}
		s.Value = m
	case []uint32:
		for _, x := range v {
			m = append(m, uint32(x))
		}
		s.Value = m
	case []uint64:
		for _, x := range v {
			m = append(m, uint64(x))
		}
		s.Value = m
	case []int8:
		for _, x := range v {
			m = append(m, int8(x))
		}
		s.Value = m
	case []int16:
		for _, x := range v {
			m = append(m, int16(x))
		}
		s.Value = m
	case []int32:
		for _, x := range v {
			m = append(m, int32(x))
		}
		s.Value = m
	case []int64:
		for _, x := range v {
			m = append(m, int64(x))
		}
		s.Value = m
	case []float32:
		for _, x := range v {
			m = append(m, float32(x))
		}
		s.Value = m
	case []float64:
		for _, x := range v {
			m = append(m, x)
		}
		s.Value = m
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceAppendSeq is a Sequence method.
//
// appendSeq appends the contents of the given sequence to the receiver. If the
// receiver's item size is smaller than that of the argument, then the receiver
// is converted to the argument's item type; otherwise, the argument's values
// are converted to the receiver's item type as they are appended.
func SequenceAppendSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("appendSeq"); err != nil {
		return vm.IoError(err)
	}
	for i := range msg.Args {
		other, obj, stop := msg.SequenceArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.Stop(obj, stop)
		}
		if other.IsMutable() {
			obj.Lock()
		}
		s = s.Append(other)
		if other.IsMutable() {
			obj.Unlock()
		}
	}
	target.Value = s
	return target
}

// SequenceAtInsertSeq is a Sequence method.
//
// atInsertSeq inserts at the index given in the first argument the object
// asString in the second.
func SequenceAtInsertSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("atInsertSeq"); err != nil {
		return vm.IoError(err)
	}
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	p := int(n)
	if p < 0 || p > s.Len() {
		return vm.RaiseExceptionf("index %d out of bounds", p)
	}
	r, exc, stop := msg.AsStringArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Value = s.Insert(Sequence{Value: []byte(r), Mutable: false, Code: "utf8"}, p)
	return target
}

// SequenceAtPut is a Sequence method.
//
// atPut replaces the element at the given position with the given value,
// growing the sequence if necessary.
func SequenceAtPut(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("atPut"); err != nil {
		return vm.IoError(err)
	}
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	x, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	p := int(n)
	if p < 0 {
		return vm.RaiseExceptionf("index %d out of bounds", p)
	}
	v := reflect.ValueOf(s.Value)
	if p >= v.Len() {
		k := p - s.Len() + 1
		w := reflect.MakeSlice(v.Type(), k, k)
		v = reflect.AppendSlice(v, w)
	}
	v.Index(p).Set(reflect.ValueOf(x).Convert(v.Type().Elem()))
	s.Value = v.Interface()
	target.Value = s
	return target
}

// SequenceSetItemType is a Sequence method.
//
// setItemType effectively reinterprets the bit pattern of the sequence data in
// the given type, which may be uint8, uint16, uint32, uint64, int8, int16,
// int32, int64, float32, or float64.
func SequenceSetItemType(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("setItemType"); err != nil {
		return vm.IoError(err)
	}
	k, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	var kind SeqKind
	switch strings.ToLower(k) {
	case "uint8":
		kind = SeqU8
	case "uint16":
		kind = SeqU16
	case "uint32":
		kind = SeqU32
	case "uint64":
		kind = SeqU64
	case "int8":
		kind = SeqS8
	case "int16":
		kind = SeqS16
	case "int32":
		kind = SeqS32
	case "int64":
		kind = SeqS64
	case "float32":
		kind = SeqF32
	case "float64":
		kind = SeqF64
	default:
		return vm.RaiseExceptionf("invalid item type name %q", k)
	}
	target.Value = vm.SequenceFromBytes(s.Bytes(), kind)
	return target
}

// SequenceClipAfterSeq is a Sequence method.
//
// clipAfterSeq removes the portion of the sequence which follows the end of
// the argument sequence.
func SequenceClipAfterSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("clipAfterSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, 0)
	n := other.Len()
	if other.IsMutable() {
		obj.Unlock()
	}
	if k >= 0 {
		s.Remove(k+n, s.Len())
	}
	return target
}

// SequenceClipAfterStartOfSeq is a Sequence method.
//
// clipAfterStartOfSeq removes the portion of the sequence which follows the
// beginning of the argument sequence.
func SequenceClipAfterStartOfSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("clipAfterStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, 0)
	if other.IsMutable() {
		obj.Unlock()
	}
	if k >= 0 {
		s.Remove(k, s.Len())
	}
	return target
}

// SequenceClipBeforeEndOfSeq is a Sequence method.
//
// clipBeforeEndOfSeq removes the portion of the sequence which precedes the end
// of the argument sequence.
func SequenceClipBeforeEndOfSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("clipBeforeStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, 0)
	n := other.Len()
	if other.IsMutable() {
		obj.Unlock()
	}
	if k >= 0 {
		s.Remove(0, k+n)
	}
	return target
}

// SequenceClipBeforeSeq is a Sequence method.
//
// clipBeforeSeq removes the portion of the sequence which precedes the
// beginning of the argument sequence.
func SequenceClipBeforeSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("clipBeforeSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, 0)
	if other.IsMutable() {
		obj.Unlock()
	}
	if k >= 0 {
		s.Remove(0, k)
	}
	return target
}

// SequenceConvertToItemType is a Sequence method.
//
// convertToItemType changes the item type of the sequence.
func SequenceConvertToItemType(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("convertToItemType"); err != nil {
		return vm.IoError(err)
	}
	k, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	var kind SeqKind
	switch strings.ToLower(k) {
	case "uint8":
		kind = SeqU8
	case "uint16":
		kind = SeqU16
	case "uint32":
		kind = SeqU32
	case "uint64":
		kind = SeqU64
	case "int8":
		kind = SeqS8
	case "int16":
		kind = SeqS16
	case "int32":
		kind = SeqS32
	case "int64":
		kind = SeqS64
	case "float32":
		kind = SeqF32
	case "float64":
		kind = SeqF64
	default:
		return vm.RaiseExceptionf("invalid item type name %q", k)
	}
	target.Value = s.Convert(kind)
	return target
}

// SequenceCopy is a Sequence method.
//
// copy sets the receiver to be a copy of the given sequence.
func SequenceCopy(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("copy"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	s.Value = copySeqVal(other.Value)
	s.Code = other.Code
	target.Value = s
	return target
}

// SequenceDuplicateIndexes is a Sequence method.
//
// duplicateIndexes inserts a copy of each item in the sequence after its
// position.
func SequenceDuplicateIndexes(vm *VM, target, locals Interface, msg *Message) *Object {
	// Did you mean: indices?
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("duplicateIndexes"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		w := make([]byte, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []uint16:
		w := make([]uint16, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []uint32:
		w := make([]uint32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []uint64:
		w := make([]uint64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []int8:
		w := make([]int8, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []int16:
		w := make([]int16, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []int32:
		w := make([]int32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []int64:
		w := make([]int64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []float32:
		w := make([]float32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case []float64:
		w := make([]float64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceEmpty is a Sequence method.
//
// empty zeroes all values in the sequence and sets its length to zero.
func SequenceEmpty(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("empty"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []uint16:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []uint32:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []uint64:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []int8:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []int16:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []int32:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []int64:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []float32:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case []float64:
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceInsertSeqEvery is a Sequence method.
//
// insertSeqEvery inserts the argument sequence into the receiver at every nth
// position.
func SequenceInsertSeqEvery(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("insertSeqEvery"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	n, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	k := int(n)
	if k <= 0 {
		return vm.RaiseExceptionf("distance must be > 0")
	}
	if k > s.Len() {
		return vm.RaiseExceptionf("index %d out of bounds", k)
	}
	if other.IsMutable() {
		obj.Lock()
		defer obj.Unlock()
	}
	if !s.SameType(other) {
		other = other.Convert(s.Kind())
	}
	if k == s.Len() {
		target.Value = s.Append(other)
		return target
	}
	switch v := s.Value.(type) {
	case []byte:
		w := other.Value.([]byte)
		n := len(v) / k
		x := make([]byte, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []uint16:
		w := other.Value.([]uint16)
		n := len(v) / k
		x := make([]uint16, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []uint32:
		w := other.Value.([]uint32)
		n := len(v) / k
		x := make([]uint32, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []uint64:
		w := other.Value.([]uint64)
		n := len(v) / k
		x := make([]uint64, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []int8:
		w := other.Value.([]int8)
		n := len(v) / k
		x := make([]int8, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []int16:
		w := other.Value.([]int16)
		n := len(v) / k
		x := make([]int16, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []int32:
		w := other.Value.([]int32)
		n := len(v) / k
		x := make([]int32, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []int64:
		w := other.Value.([]int64)
		n := len(v) / k
		x := make([]int64, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []float32:
		w := other.Value.([]float32)
		n := len(v) / k
		x := make([]float32, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	case []float64:
		w := other.Value.([]float64)
		n := len(v) / k
		x := make([]float64, 0, len(v)+len(w)*n)
		for i := 0; i < n; i++ {
			x = append(x, v[i*k:(i+1)*k]...)
			x = append(x, w...)
		}
		if len(v)%k != 0 {
			x = append(x, v[n*k:]...)
		}
		s.Value = x
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceLeaveThenRemove is a Sequence method.
//
// leaveThenRemove(m, n) keeps the first m items, removes the following n, and
// repeats this process on the remainder.
func SequenceLeaveThenRemove(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("leaveThenRemove"); err != nil {
		return vm.IoError(err)
	}
	mm, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m := int(mm)
	if m < 0 {
		return vm.RaiseExceptionf("leaveThenRemove arguments must be nonnegative")
	}
	nn, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	n := int(nn)
	if n < 0 {
		return vm.RaiseExceptionf("leaveThenRemove arguments must be nonnegative")
	}
	if n == 0 {
		return target
	}
	switch v := s.Value.(type) {
	case []byte:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []uint16:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []uint32:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []uint64:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []int8:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []int16:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []int32:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []int64:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []float32:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case []float64:
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequencePreallocateToSize is a Sequence method.
//
// preallocateToSize ensures that the receiver can grow to be at least n bytes
// without reallocating.
func SequencePreallocateToSize(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("preallocateToSize"); err != nil {
		return vm.IoError(err)
	}
	nn, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	n := (int(nn) + s.ItemSize() - 1) / s.ItemSize()
	v := reflect.ValueOf(s.Value)
	if v.Cap() < n {
		nv := reflect.MakeSlice(v.Type(), v.Len(), n)
		reflect.Copy(nv, v)
		s.Value = nv.Interface()
		target.Value = s
	}
	return target
}

// SequenceRangeFill is a Sequence method.
//
// rangeFill sets each element of the sequence to its index.
func SequenceRangeFill(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("rangeFill"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := range v {
			v[i] = byte(i)
		}
	case []uint16:
		for i := range v {
			v[i] = uint16(i)
		}
	case []uint32:
		for i := range v {
			v[i] = uint32(i)
		}
	case []uint64:
		for i := range v {
			v[i] = uint64(i)
		}
	case []int8:
		for i := range v {
			v[i] = int8(i)
		}
	case []int16:
		for i := range v {
			v[i] = int16(i)
		}
	case []int32:
		for i := range v {
			v[i] = int32(i)
		}
	case []int64:
		for i := range v {
			v[i] = int64(i)
		}
	case []float32:
		for i := range v {
			v[i] = float32(i)
		}
	case []float64:
		for i := range v {
			v[i] = float64(i)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target
}

// SequenceRemoveAt is a Sequence method.
//
// removeAt removes the nth element from the sequence.
func SequenceRemoveAt(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeAt"); err != nil {
		return vm.IoError(err)
	}
	nn, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := s.FixIndex(int(nn))
	n := s.Len()
	if k < n {
		target.Value = s.Remove(k, k+1)
	}
	return target
}

// SequenceRemoveEvenIndexes is a Sequence method.
//
// removeEvenIndexes deletes each element whose index in the sequence is even.
func SequenceRemoveEvenIndexes(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeEvenIndexes"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:(len(v))/2]
	case []uint16:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []uint32:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []uint64:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []int8:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []int16:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []int32:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []int64:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []float32:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case []float64:
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceRemoveLast is a Sequence method.
//
// removeLast removes the last element from the sequence and returns the
// receiver.
func SequenceRemoveLast(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeLast"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []uint16:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []uint32:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []uint64:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []int8:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []int16:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []int32:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []int64:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []float32:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case []float64:
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceRemoveOddIndexes is a Sequence method.
//
// removeOddIndexes deletes each element whose index in the sequence is odd.
func SequenceRemoveOddIndexes(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeOddIndexes"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []uint16:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []uint32:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []uint64:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []int8:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []int16:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []int32:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []int64:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []float32:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case []float64:
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	target.Value = s
	return target
}

// SequenceRemovePrefix is a Sequence method.
//
// removePrefix removes a prefix from the receiver, if present.
func SequenceRemovePrefix(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removePrefix"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	ol := other.Len()
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		if findUMatch(m, o, 0, ol, mt) {
			target.Value = s.Remove(0, ol)
		}
	case []int8, []int16, []int32, []int64:
		if findIMatch(m, o, 0, ol, mt) {
			target.Value = s.Remove(0, ol)
		}
	case []float32, []float64:
		if findFMatch(m, o, 0, ol, mt) {
			target.Value = s.Remove(0, ol)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	if other.IsMutable() {
		obj.Unlock()
	}
	return target
}

// SequenceRemoveSeq is a Sequence method.
//
// removeSeq removes all occurrences of a sequence from the receiver.
func SequenceRemoveSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	ol := other.Len()
	for {
		i := s.RFind(other, s.Len())
		if i < 0 {
			break
		}
		s = s.Remove(i, i+ol)
	}
	if other.IsMutable() {
		obj.Unlock()
	}
	target.Value = s
	return target
}

// SequenceRemoveSlice is a Sequence method.
//
// removeSlice removes items between the given start and end, inclusive.
func SequenceRemoveSlice(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeSlice"); err != nil {
		return vm.IoError(err)
	}
	l, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	r, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	i := s.FixIndex(int(l))
	j := s.FixIndex(int(r) + 1)
	if i < s.Len() {
		target.Value = s.Remove(i, j)
	}
	return target
}

// SequenceRemoveSuffix is a Sequence method.
//
// removeSuffix removes a suffix from the receiver, if present.
func SequenceRemoveSuffix(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("removeSuffix"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	sl := s.Len()
	ol := other.Len()
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		if findUMatch(m, o, sl-ol, ol, mt) {
			target.Value = s.Remove(sl-ol, sl)
		}
	case []int8, []int16, []int32, []int64:
		if findIMatch(m, o, sl-ol, ol, mt) {
			target.Value = s.Remove(sl-ol, sl)
		}
	case []float32, []float64:
		if findFMatch(m, o, sl-ol, ol, mt) {
			target.Value = s.Remove(sl-ol, sl)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	if other.IsMutable() {
		obj.Unlock()
	}
	return target
}

// SequenceReplaceFirstSeq is a Sequence method.
//
// replaceFirstSeq replaces the first instance of a sequence with another.
func SequenceReplaceFirstSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("replaceFirstSeq"); err != nil {
		return vm.IoError(err)
	}
	search, sobj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(sobj, stop)
	}
	repl, robj, stop := msg.SequenceArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(robj, stop)
	}
	k := 0
	if msg.ArgCount() > 2 {
		start, exc, stop := msg.NumberArgAt(vm, locals, 2)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		k = int(start)
		if k < 0 {
			k = 0
		}
	}
	if search.IsMutable() {
		sobj.Lock()
	}
	p := s.Find(search, k)
	sl := search.Len()
	if search.IsMutable() {
		sobj.Unlock()
	}
	if p >= 0 {
		s = s.Remove(p, p+sl)
		if repl.IsMutable() {
			robj.Lock()
		}
		target.Value = s.Insert(repl, p)
		if repl.IsMutable() {
			robj.Unlock()
		}
	}
	return target
}

// SequenceReplaceSeq is a Sequence method.
//
// replaceSeq replaces all instances of a sequence with another.
func SequenceReplaceSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("replaceSeq"); err != nil {
		return vm.IoError(err)
	}
	search, sobj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(sobj, stop)
	}
	if search.IsMutable() {
		sobj.Lock()
		defer sobj.Unlock()
	}
	sl := search.Len()
	if sl == 0 {
		return vm.RaiseExceptionf("cannot replace length 0 sequence")
	}
	repl, robj, stop := msg.SequenceArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(robj, stop)
	}
	if repl.IsMutable() {
		robj.Lock()
	}
	rl := repl.Len()
	for k := s.Find(search, 0); k >= 0; k = s.Find(search, k+rl) {
		target.Value = s.Remove(k, k+sl).Insert(repl, k)
	}
	if repl.IsMutable() {
		robj.Unlock()
	}
	return target
}

// SequenceReverseInPlace is a Sequence method.
//
// reverseInPlace reverses the elements of the sequence.
func SequenceReverseInPlace(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("reverseInPlace"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []uint16:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []uint32:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []uint64:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []int8:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []int16:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []int32:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []int64:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []float32:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case []float64:
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target
}

// SequenceSetItemsToDouble is a Sequence method.
//
// setItemsToDouble sets all items to the given value.
func SequenceSetItemsToDouble(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("setItemsToDouble"); err != nil {
		return vm.IoError(err)
	}
	x, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	switch v := s.Value.(type) {
	case []byte:
		for i := range v {
			v[i] = byte(x)
		}
	case []uint16:
		for i := range v {
			v[i] = uint16(x)
		}
	case []uint32:
		for i := range v {
			v[i] = uint32(x)
		}
	case []uint64:
		for i := range v {
			v[i] = uint64(x)
		}
	case []int8:
		for i := range v {
			v[i] = int8(x)
		}
	case []int16:
		for i := range v {
			v[i] = int16(x)
		}
	case []int32:
		for i := range v {
			v[i] = int32(x)
		}
	case []int64:
		for i := range v {
			v[i] = int64(x)
		}
	case []float32:
		for i := range v {
			v[i] = float32(x)
		}
	case []float64:
		for i := range v {
			v[i] = float64(x)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target
}

// SequenceSetSize is a Sequence method.
//
// setSize sets the size of the sequence.
func SequenceSetSize(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("setSize"); err != nil {
		return vm.IoError(err)
	}
	nn, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	n := int(nn)
	if n < 0 {
		return vm.RaiseExceptionf("size must be nonnegative")
	}
	l := s.Len()
	if n < l {
		s.Value = reflect.ValueOf(s.Value).Slice(0, n).Interface()
	} else if n > l {
		v := reflect.ValueOf(s.Value)
		s.Value = reflect.AppendSlice(v, reflect.MakeSlice(v.Type(), n-l, n-l)).Interface()
	}
	target.Value = s
	return target
}

// SequenceSort is a Sequence method.
//
// sort sorts the elements of the sequence.
func SequenceSort(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("sort"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []uint16:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []uint32:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []uint64:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []int8:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []int16:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []int32:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []int64:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []float32:
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case []float64:
		sort.Float64s(v)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target
}

// SequenceZero is a Sequence method.
//
// zero sets each element of the receiver to zero.
func SequenceZero(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("zero"); err != nil {
		return vm.IoError(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		for k := range v {
			v[k] = 0
		}
	case []uint16:
		for k := range v {
			v[k] = 0
		}
	case []uint32:
		for k := range v {
			v[k] = 0
		}
	case []uint64:
		for k := range v {
			v[k] = 0
		}
	case []int8:
		for k := range v {
			v[k] = 0
		}
	case []int16:
		for k := range v {
			v[k] = 0
		}
	case []int32:
		for k := range v {
			v[k] = 0
		}
	case []int64:
		for k := range v {
			v[k] = 0
		}
	case []float32:
		for k := range v {
			v[k] = 0
		}
	case []float64:
		for k := range v {
			v[k] = 0
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target
}
