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
	switch s.Kind {
	case SeqMU8:
		m := s.Value.([]byte)
		for _, x := range v {
			m = append(m, byte(x))
		}
		s.Value = m
	case SeqMU16:
		m := s.Value.([]uint16)
		for _, x := range v {
			m = append(m, uint16(x))
		}
		s.Value = m
	case SeqMU32:
		m := s.Value.([]uint32)
		for _, x := range v {
			m = append(m, uint32(x))
		}
		s.Value = m
	case SeqMU64:
		m := s.Value.([]uint64)
		for _, x := range v {
			m = append(m, uint64(x))
		}
		s.Value = m
	case SeqMS8:
		m := s.Value.([]int8)
		for _, x := range v {
			m = append(m, int8(x))
		}
		s.Value = m
	case SeqMS16:
		m := s.Value.([]int16)
		for _, x := range v {
			m = append(m, int16(x))
		}
		s.Value = m
	case SeqMS32:
		m := s.Value.([]int32)
		for _, x := range v {
			m = append(m, int32(x))
		}
		s.Value = m
	case SeqMS64:
		m := s.Value.([]int64)
		for _, x := range v {
			m = append(m, int64(x))
		}
		s.Value = m
	case SeqMF32:
		m := s.Value.([]float32)
		for _, x := range v {
			m = append(m, float32(x))
		}
		s.Value = m
	case SeqMF64:
		s.Value = append(s.Value.([]float64), v...)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
		kind = SeqMU8
	case "uint16":
		kind = SeqMU16
	case "uint32":
		kind = SeqMU32
	case "uint64":
		kind = SeqMU64
	case "int8":
		kind = SeqMS8
	case "int16":
		kind = SeqMS16
	case "int32":
		kind = SeqMS32
	case "int64":
		kind = SeqMS64
	case "float32":
		kind = SeqMF32
	case "float64":
		kind = SeqMF64
	default:
		return vm.RaiseException("invalid item type name")
	}
	ns := vm.SequenceFromBytes(s.Bytes(), kind)
	s.Value = ns.Value
	s.Kind = ns.Kind
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
		s.Slice(0, k, 1)
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
		s.Slice(0, k, 1)
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
		s.Slice(k, s.Len(), 1)
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
		s.Slice(k, s.Len(), 1)
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
		kind = SeqMU8
	case "uint16":
		kind = SeqMU16
	case "uint32":
		kind = SeqMU32
	case "uint64":
		kind = SeqMU64
	case "int8":
		kind = SeqMS8
	case "int16":
		kind = SeqMS16
	case "int32":
		kind = SeqMS32
	case "int64":
		kind = SeqMS64
	case "float32":
		kind = SeqMF32
	case "float64":
		kind = SeqMF64
	default:
		return vm.RaiseException("invalid item type name")
	}
	ns := s.Convert(vm, kind)
	s.Value = ns.Value
	s.Kind = ns.Kind
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
	switch other.Kind {
	case SeqMU8, SeqIU8:
		s.Value = append([]byte{}, other.Value.([]uint8)...)
	case SeqMU16, SeqIU16:
		s.Value = append([]uint16{}, other.Value.([]uint16)...)
	case SeqMU32, SeqIU32:
		s.Value = append([]uint32{}, other.Value.([]uint32)...)
	case SeqMU64, SeqIU64:
		s.Value = append([]uint64{}, other.Value.([]uint64)...)
	case SeqMS8, SeqIS8:
		s.Value = append([]int8{}, other.Value.([]int8)...)
	case SeqMS16, SeqIS16:
		s.Value = append([]int16{}, other.Value.([]int16)...)
	case SeqMS32, SeqIS32:
		s.Value = append([]int32{}, other.Value.([]int32)...)
	case SeqMS64, SeqIS64:
		s.Value = append([]int64{}, other.Value.([]int64)...)
	case SeqMF32, SeqIF32:
		s.Value = append([]float32{}, other.Value.([]float32)...)
	case SeqMF64, SeqIF64:
		s.Value = append([]float64{}, other.Value.([]float64)...)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", other.Kind))
	}
	s.Kind = other.Kind
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		w := make([]byte, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		w := make([]uint16, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		w := make([]uint32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		w := make([]uint64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		w := make([]int8, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		w := make([]int16, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		w := make([]int32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		w := make([]int64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		w := make([]float32, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		w := make([]float64, 0, 2*len(v))
		for _, c := range v {
			w = append(w, c, c)
		}
		s.Value = w
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i := range v {
			v[i] = 0
		}
		s.Value = v[:0]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
		other = other.Convert(vm, s.Kind)
	}
	if k == s.Len() {
		s.Append(other)
		return target, NoStop
	}
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
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
	case SeqMU16:
		v := s.Value.([]uint16)
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
	case SeqMU32:
		v := s.Value.([]uint32)
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
	case SeqMU64:
		v := s.Value.([]uint64)
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
	case SeqMS8:
		v := s.Value.([]int8)
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
	case SeqMS16:
		v := s.Value.([]int16)
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
	case SeqMS32:
		v := s.Value.([]int32)
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
	case SeqMS64:
		v := s.Value.([]int64)
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
	case SeqMF32:
		v := s.Value.([]float32)
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
	case SeqMF64:
		v := s.Value.([]float64)
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
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		if m == 0 {
			s.Value = v[:0]
			break
		}
		k := 0
		for i, o := 0, 0; i < len(v); i, o = i+m+n, o+m {
			k += copy(v[o:o+m], v[i:])
		}
		s.Value = v[:k]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i := range v {
			v[i] = byte(i)
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i := range v {
			v[i] = uint16(i)
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i := range v {
			v[i] = uint32(i)
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i := range v {
			v[i] = uint64(i)
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i := range v {
			v[i] = int8(i)
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i := range v {
			v[i] = int16(i)
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i := range v {
			v[i] = int32(i)
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i := range v {
			v[i] = int64(i)
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i := range v {
			v[i] = float32(i)
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i := range v {
			v[i] = float64(i)
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:(len(v))/2]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i := 0; 2*i+1 < len(v); i++ {
			v[i] = v[2*i+1]
		}
		s.Value = v[:len(v)/2]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		if len(v) > 0 {
			s.Value = v[:len(v)-1]
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i := 0; 2*i < len(v); i++ {
			v[i] = v[2*i]
		}
		s.Value = v[:(len(v)+1)/2]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	if s.findMatch(other, 0, ol) {
		s.Slice(ol, s.Len(), 1)
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
	ol := other.Len()
	if s.findMatch(other, s.Len()-ol, ol) {
		s.Slice(0, s.Len()-ol, 1)
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
	repl, err, stop := msg.SequenceArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	sl := search.Len()
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
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i := 0; i < len(v)/2; i++ {
			v[i], v[len(v)-i-1] = v[len(v)-i-1], v[i]
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		for i := range v {
			v[i] = byte(x)
		}
	case SeqMU16:
		v := s.Value.([]uint16)
		for i := range v {
			v[i] = uint16(x)
		}
	case SeqMU32:
		v := s.Value.([]uint32)
		for i := range v {
			v[i] = uint32(x)
		}
	case SeqMU64:
		v := s.Value.([]uint64)
		for i := range v {
			v[i] = uint64(x)
		}
	case SeqMS8:
		v := s.Value.([]int8)
		for i := range v {
			v[i] = int8(x)
		}
	case SeqMS16:
		v := s.Value.([]int16)
		for i := range v {
			v[i] = int16(x)
		}
	case SeqMS32:
		v := s.Value.([]int32)
		for i := range v {
			v[i] = int32(x)
		}
	case SeqMS64:
		v := s.Value.([]int64)
		for i := range v {
			v[i] = int64(x)
		}
	case SeqMF32:
		v := s.Value.([]float32)
		for i := range v {
			v[i] = float32(x)
		}
	case SeqMF64:
		v := s.Value.([]float64)
		for i := range v {
			v[i] = float64(x)
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMU16:
		v := s.Value.([]uint16)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMU32:
		v := s.Value.([]uint32)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMU64:
		v := s.Value.([]uint64)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMS8:
		v := s.Value.([]int8)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMS16:
		v := s.Value.([]int16)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMS32:
		v := s.Value.([]int32)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMS64:
		v := s.Value.([]int64)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMF32:
		v := s.Value.([]float32)
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	case SeqMF64:
		v := s.Value.([]float64)
		sort.Float64s(v)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
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
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		for k := range v {
			v[k] = 0
		}
	case SeqMU16:
		v := s.Value.([]uint16)
		for k := range v {
			v[k] = 0
		}
	case SeqMU32:
		v := s.Value.([]uint32)
		for k := range v {
			v[k] = 0
		}
	case SeqMU64:
		v := s.Value.([]uint64)
		for k := range v {
			v[k] = 0
		}
	case SeqMS8:
		v := s.Value.([]int8)
		for k := range v {
			v[k] = 0
		}
	case SeqMS16:
		v := s.Value.([]int16)
		for k := range v {
			v[k] = 0
		}
	case SeqMS32:
		v := s.Value.([]int32)
		for k := range v {
			v[k] = 0
		}
	case SeqMS64:
		v := s.Value.([]int64)
		for k := range v {
			v[k] = 0
		}
	case SeqMF32:
		v := s.Value.([]float32)
		for k := range v {
			v[k] = 0
		}
	case SeqMF64:
		v := s.Value.([]float64)
		for k := range v {
			v[k] = 0
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return target, NoStop
}
