package iolang

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CheckMutable returns an error if the sequence is not mutable, or nil
// otherwise.
func (s *Sequence) CheckMutable(name string) error {
	if s.IsMutable() {
		return nil
	}
	return fmt.Errorf("'%s' cannot be called on an immutable sequence", name)
}

// SequenceAsMutable is a Sequence method.
//
// asMutable creates a mutable copy of the sequence.
func SequenceAsMutable(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// This isn't actually a mutable method, but it feels more appropriate
	// here with them.
	s := target.(*Sequence)
	return vm.NewSequence(s.Value, true, s.Code), NoStop
}

// SequenceAppend is a Sequence method.
//
// append adds numbers to the sequence.
func SequenceAppend(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("append"); err != nil {
		return vm.IoError(err)
	}
	v := make([]float64, msg.ArgCount())
	for i, arg := range msg.Args {
		r, stop := arg.Eval(vm, locals)
		if stop != NoStop {
			return r, stop
		}
		n, ok := r.(*Number)
		if !ok {
			return vm.RaiseException("all arguments to append must be Number, not " + vm.TypeName(r))
		}
		v[i] = n.Value
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
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target, NoStop
}

// SequenceAppendSeq is a Sequence method.
//
// appendSeq appends the contents of the given sequence to the receiver. If the
// receiver's item size is smaller than that of the argument, then the receiver
// is converted to the argument's item type; otherwise, the argument's values
// are converted to the receiver's item type as they are appended.
func SequenceAppendSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("appendSeq"); err != nil {
		return vm.IoError(err)
	}
	for i := range msg.Args {
		other, err, stop := msg.SequenceArgAt(vm, locals, i)
		if stop != NoStop {
			return err, stop
		}
		s.Append(other)
	}
	return target, NoStop
}

// SequenceAtInsertSeq is a Sequence method.
//
// atInsertSeq inserts at the index given in the first argument the object
// asString in the second.
func SequenceAtInsertSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("atInsertSeq"); err != nil {
		return vm.IoError(err)
	}
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	p := int(n.Value)
	if p < 0 || p > s.Len() {
		return vm.RaiseException("index out of bounds")
	}
	r, err, stop := msg.AsStringArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	s.Insert(r, p)
	return target, NoStop
}

// SequenceAtPut is a Sequence method.
//
// atPut replaces the element at the given position with the given value,
// growing the sequence if necessary.
func SequenceAtPut(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("atPut"); err != nil {
		return vm.IoError(err)
	}
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	x, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	p := int(n.Value)
	if p < 0 {
		return vm.RaiseException("index out of bounds")
	}
	v := reflect.ValueOf(s.Value)
	if p >= v.Len() {
		k := p - s.Len() + 1
		w := reflect.MakeSlice(v.Type(), k, k)
		v = reflect.AppendSlice(v, w)
	}
	v.Index(p).Set(reflect.ValueOf(x.Value).Convert(v.Type().Elem()))
	s.Value = v.Interface()
	return s, NoStop
}

// SequenceSetItemType is a Sequence method.
//
// setItemType effectively reinterprets the bit pattern of the sequence data in
// the given type, which may be uint8, uint16, uint32, uint64, int8, int16,
// int32, int64, float32, or float64.
func SequenceSetItemType(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("setItemType"); err != nil {
		return vm.IoError(err)
	}
	n, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	var kind SeqKind
	switch strings.ToLower(n.String()) {
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
		return vm.RaiseException("invalid item type name")
	}
	ns := vm.SequenceFromBytes(s.Bytes(), kind)
	s.Value = ns.Value
	s.Code = ns.Code
	return target, NoStop
}

// SequenceClipAfterSeq is a Sequence method.
//
// clipAfterSeq removes the portion of the sequence which follows the end of
// the argument sequence.
func SequenceClipAfterSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipAfterSeq"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		k += other.Len()
		s.Remove(k, s.Len())
	}
	return target, NoStop
}

// SequenceClipAfterStartOfSeq is a Sequence method.
//
// clipAfterStartOfSeq removes the portion of the sequence which follows the
// beginning of the argument sequence.
func SequenceClipAfterStartOfSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipAfterStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		s.Remove(k, s.Len())
	}
	return target, NoStop
}

// SequenceClipBeforeEndOfSeq is a Sequence method.
//
// clipBeforeEndOfSeq removes the portion of the sequence which precedes the end
// of the argument sequence.
func SequenceClipBeforeEndOfSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipBeforeStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		k += other.Len()
		s.Remove(0, k)
	}
	return target, NoStop
}

// SequenceClipBeforeSeq is a Sequence method.
//
// clipBeforeSeq removes the portion of the sequence which precedes the
// beginning of the argument sequence.
func SequenceClipBeforeSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipBeforeSeq"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		s.Remove(0, k)
	}
	return target, NoStop
}

// SequenceConvertToItemType is a Sequence method.
//
// convertToItemType changes the item type of the sequence.
func SequenceConvertToItemType(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("convertToItemType"); err != nil {
		return vm.IoError(err)
	}
	n, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	var kind SeqKind
	switch strings.ToLower(n.String()) {
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
		return vm.RaiseException("invalid item type name")
	}
	ns := s.Convert(vm, kind)
	s.Value = ns.Value
	s.Code = ns.Code
	return target, NoStop
}

// SequenceCopy is a Sequence method.
//
// copy sets the receiver to be a copy of the given sequence.
func SequenceCopy(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("copy"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	switch v := other.Value.(type) {
	case []byte:
		s.Value = append([]byte{}, v...)
	case []uint16:
		s.Value = append([]uint16{}, v...)
	case []uint32:
		s.Value = append([]uint32{}, v...)
	case []uint64:
		s.Value = append([]uint64{}, v...)
	case []int8:
		s.Value = append([]int8{}, v...)
	case []int16:
		s.Value = append([]int16{}, v...)
	case []int32:
		s.Value = append([]int32{}, v...)
	case []int64:
		s.Value = append([]int64{}, v...)
	case []float32:
		s.Value = append([]float32{}, v...)
	case []float64:
		s.Value = append([]float64{}, v...)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", other.Value))
	}
	s.Code = other.Code
	return target, NoStop
}

// SequenceDuplicateIndexes is a Sequence method.
//
// duplicateIndexes inserts a copy of each item in the sequence after its
// position.
func SequenceDuplicateIndexes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// Did you mean: indices?
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceEmpty is a Sequence method.
//
// empty zeroes all values in the sequence and sets its length to zero.
func SequenceEmpty(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceInsertSeqEvery is a Sequence method.
//
// insertSeqEvery inserts the argument sequence into the receiver at every nth
// position.
func SequenceInsertSeqEvery(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("insertSeqEvery"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	k := int(n.Value)
	if k <= 0 {
		return vm.RaiseException("distance must be > 0")
	}
	if k > s.Len() {
		return vm.RaiseException("out of bounds")
	}
	if !s.SameType(other) {
		other = other.Convert(vm, s.Kind())
	}
	if k == s.Len() {
		s.Append(other)
		return target, NoStop
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
	return target, NoStop
}

// SequenceLeaveThenRemove is a Sequence method.
//
// leaveThenRemove(m, n) keeps the first m items, removes the following n, and
// repeats this process on the remainder.
func SequenceLeaveThenRemove(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("leaveThenRemove"); err != nil {
		return vm.IoError(err)
	}
	mm, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	m := int(mm.Value)
	if m < 0 {
		return vm.RaiseException("leaveThenRemove arguments must be nonnegative")
	}
	nn, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	n := int(nn.Value)
	if n < 0 {
		return vm.RaiseException("leaveThenRemove arguments must be nonnegative")
	}
	if n == 0 {
		return target, NoStop
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
	return target, NoStop
}

// SequencePreallocateToSize is a Sequence method.
//
// preallocateToSize ensures that the receiver can grow to be at least n bytes
// without reallocating.
func SequencePreallocateToSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("preallocateToSize"); err != nil {
		return vm.IoError(err)
	}
	nn, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n := (int(nn.Value) + s.ItemSize() - 1) / s.ItemSize()
	v := reflect.ValueOf(s.Value)
	if v.Cap() < n {
		nv := reflect.MakeSlice(v.Type(), v.Len(), n)
		reflect.Copy(nv, v)
		s.Value = nv.Interface()
	}
	return target, NoStop
}

// SequenceRangeFill is a Sequence method.
//
// rangeFill sets each element of the sequence to its index.
func SequenceRangeFill(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceRemoveAt is a Sequence method.
//
// removeAt removes the nth element from the sequence.
func SequenceRemoveAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("removeAt"); err != nil {
		return vm.IoError(err)
	}
	nn, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.FixIndex(int(nn.Value))
	n := s.Len()
	if k < n {
		s.Remove(k, k+1)
	}
	return target, NoStop
}

// SequenceRemoveEvenIndexes is a Sequence method.
//
// removeEvenIndexes deletes each element whose index in the sequence is even.
func SequenceRemoveEvenIndexes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceRemoveLast is a Sequence method.
//
// removeLast removes the last element from the sequence and returns the
// receiver.
func SequenceRemoveLast(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceRemoveOddIndexes is a Sequence method.
//
// removeOddIndexes deletes each element whose index in the sequence is odd.
func SequenceRemoveOddIndexes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceRemovePrefix is a Sequence method.
//
// removePrefix removes a prefix from the receiver, if present.
func SequenceRemovePrefix(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("removePrefix"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ol := other.Len()
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		if findUMatch(m, o, 0, ol, mt) {
			s.Remove(0, ol)
		}
	case []int8, []int16, []int32, []int64:
		if findIMatch(m, o, 0, ol, mt) {
			s.Remove(0, ol)
		}
	case []float32, []float64:
		if findFMatch(m, o, 0, ol, mt) {
			s.Remove(0, ol)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target, NoStop
}

// SequenceRemoveSeq is a Sequence method.
//
// removeSeq removes all occurrences of a sequence from the receiver.
func SequenceRemoveSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("removeSeq"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ol := other.Len()
	for {
		i := s.RFind(other, s.Len())
		if i < 0 {
			break
		}
		s.Remove(i, i+ol)
	}
	return target, NoStop
}

// SequenceRemoveSlice is a Sequence method.
//
// removeSlice removes items between the given start and end, inclusive.
func SequenceRemoveSlice(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("removeSlice"); err != nil {
		return vm.IoError(err)
	}
	l, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	i := s.FixIndex(int(l.Value))
	j := s.FixIndex(int(r.Value) + 1)
	if i < s.Len() {
		s.Remove(i, j)
	}
	return target, NoStop
}

// SequenceRemoveSuffix is a Sequence method.
//
// removeSuffix removes a suffix from the receiver, if present.
func SequenceRemoveSuffix(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("removeSuffix"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	sl := s.Len()
	ol := other.Len()
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		if findUMatch(m, o, sl-ol, ol, mt) {
			s.Remove(sl-ol, sl)
		}
	case []int8, []int16, []int32, []int64:
		if findIMatch(m, o, sl-ol, ol, mt) {
			s.Remove(sl-ol, sl)
		}
	case []float32, []float64:
		if findFMatch(m, o, sl-ol, ol, mt) {
			s.Remove(sl-ol, sl)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return target, NoStop
}

// SequenceReplaceFirstSeq is a Sequence method.
//
// replaceFirstSeq replaces the first instance of a sequence with another.
func SequenceReplaceFirstSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("replaceFirstSeq"); err != nil {
		return vm.IoError(err)
	}
	search, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	repl, err, stop := msg.SequenceArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	k := 0
	if msg.ArgCount() > 2 {
		start, err, stop := msg.NumberArgAt(vm, locals, 2)
		if stop != NoStop {
			return err, stop
		}
		k = int(start.Value)
		if k < 0 {
			k = 0
		}
	}
	p := s.Find(search, k)
	if p >= 0 {
		s.Remove(p, p+search.Len())
		s.Insert(repl, p)
	}
	return target, NoStop
}

// SequenceReplaceSeq is a Sequence method.
//
// replaceSeq replaces all instances of a sequence with another.
func SequenceReplaceSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("replaceSeq"); err != nil {
		return vm.IoError(err)
	}
	search, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	sl := search.Len()
	if sl == 0 {
		return vm.RaiseException("cannot replace length 0 sequence")
	}
	repl, err, stop := msg.SequenceArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	rl := repl.Len()
	for k := s.Find(search, 0); k >= 0; k = s.Find(search, k+rl) {
		s.Remove(k, k+sl)
		s.Insert(repl, k)
	}
	return target, NoStop
}

// SequenceReverseInPlace is a Sequence method.
//
// reverseInPlace reverses the elements of the sequence.
func SequenceReverseInPlace(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceSetItemsToDouble is a Sequence method.
//
// setItemsToDouble sets all items to the given value.
func SequenceSetItemsToDouble(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("setItemsToDouble"); err != nil {
		return vm.IoError(err)
	}
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	x := n.Value
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
	return target, NoStop
}

// SequenceSetSize is a Sequence method.
//
// setSize sets the size of the sequence.
func SequenceSetSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("setSize"); err != nil {
		return vm.IoError(err)
	}
	nn, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n := int(nn.Value)
	if n < 0 {
		return vm.RaiseException("size must be nonnegative")
	}
	l := s.Len()
	if n < l {
		s.Value = reflect.ValueOf(s.Value).Slice(0, n).Interface()
	} else if n > l {
		v := reflect.ValueOf(s.Value)
		s.Value = reflect.AppendSlice(v, reflect.MakeSlice(v.Type(), n-l, n-l)).Interface()
	}
	return target, NoStop
}

// SequenceSort is a Sequence method.
//
// sort sorts the elements of the sequence.
func SequenceSort(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}

// SequenceZero is a Sequence method.
//
// zero sets each element of the receiver to zero.
func SequenceZero(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
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
	return target, NoStop
}
