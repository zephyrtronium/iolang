package iolang

import (
	"fmt"
	"reflect"
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
func SequenceAsMutable(vm *VM, target, locals Interface, msg *Message) Interface {
	// This isn't actually a mutable method, but it feels more appropriate
	// here with them.
	s := target.(*Sequence)
	return vm.NewSequence(s.Value, true, s.Code)
}

// SequenceAppend is a Sequence method.
//
// append adds numbers to the sequence.
func SequenceAppend(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("append"); err != nil {
		return vm.IoError(err)
	}
	v := make([]float64, msg.ArgCount())
	for i, arg := range msg.Args {
		r, ok := CheckStop(arg.Eval(vm, locals), LoopStops)
		if !ok {
			return r
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
	return target
}

// SequenceAppendSeq is a Sequence method.
//
// appendSeq appends the contents of the given sequence to the receiver. If the
// receiver's item size is smaller than that of the argument, then the receiver
// is converted to the argument's item type; otherwise, the argument's values
// are converted to the receiver's item type as they are appended.
func SequenceAppendSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("appendSeq"); err != nil {
		return vm.IoError(err)
	}
	for i := range msg.Args {
		other, stop := msg.SequenceArgAt(vm, locals, i)
		if stop != nil {
			return stop
		}
		s.Append(other)
	}
	return target
}

// SequenceAtInsertSeq is a Sequence method.
//
// atInsertSeq inserts at the index given in the first argument the object
// asString in the second.
func SequenceAtInsertSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("atInsertSeq"); err != nil {
		return vm.IoError(err)
	}
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	p := int(n.Value)
	if p < 0 || p > s.Len() {
		return vm.RaiseException("index out of bounds")
	}
	r, stop := msg.AsStringArgAt(vm, locals, 1)
	if stop != nil {
		return stop
	}
	// Since the inserted sequence's values are always converted to the type of
	// the receiver, it would be _possible_ to use a switch for this, but I'm
	// lazy and not entirely convinced switches are actually more efficient.
	if p == s.Len() {
		// shortcut
		s.appendSameKind(r.Convert(vm, s.Kind))
		return target
	}
	u := reflect.ValueOf(r.Value)
	v := reflect.ValueOf(s.Value)
	w := reflect.MakeSlice(v.Type(), u.Len(), u.Len())
	x := reflect.MakeSlice(v.Type(), v.Len()-p, v.Len()-p)
	t := v.Type().Elem()
	for i := 0; i < w.Len(); i++ {
		w.Index(i).Set(u.Index(i).Convert(t))
	}
	reflect.Copy(x, v.Slice(p, v.Len()))
	v = reflect.AppendSlice(v.Slice(0, p), w)
	v = reflect.AppendSlice(v, x)
	s.Value = v.Interface()
	return target
}

// SequenceAtPut is a Sequence method.
//
// atPut replaces the element at the given position with the given value,
// growing the sequence if necessary.
func SequenceAtPut(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("atPut"); err != nil {
		return vm.IoError(err)
	}
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	x, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != nil {
		return stop
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
	return s
}

// SequenceSetItemType is a Sequence method.
//
// setItemType effectively reinterprets the bit pattern of the sequence data in
// the given type, which may be uint8, uint16, uint32, uint64, int8, int16,
// int32, int64, float32, or float64.
func SequenceSetItemType(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("setItemType"); err != nil {
		return vm.IoError(err)
	}
	n, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return target
}

// SequenceClipAfterSeq is a Sequence method.
//
// clipAfterSeq removes the portion of the sequence which follows the end of
// the argument sequence.
func SequenceClipAfterSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipAfterSeq"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		k += other.Len()
		s.Slice(0, k, 1)
	}
	return target
}

// SequenceClipAfterStartOfSeq is a Sequence method.
//
// clipAfterStartOfSeq removes the portion of the sequence which follows the
// beginning of the argument sequence.
func SequenceClipAfterStartOfSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipAfterStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		s.Slice(0, k, 1)
	}
	return target
}

// SequenceClipBeforeEndOfSeq is a Sequence method.
//
// clipBeforeEndOfSeq removes the portion of the sequence which precedes the end
// of the argument sequence.
func SequenceClipBeforeEndOfSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipBeforeStartOfSeq"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		k += other.Len()
		s.Slice(k, s.Len(), 1)
	}
	return target
}

// SequenceClipBeforeSeq is a Sequence method.
//
// clipBeforeSeq removes the portion of the sequence which precedes the
// beginning of the argument sequence.
func SequenceClipBeforeSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("clipBeforeSeq"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	k := s.Find(other, 0)
	if k >= 0 {
		s.Slice(k, s.Len(), 1)
	}
	return target
}

// SequenceConvertToItemType is a Sequence method.
//
// convertToItemType changes the item type of the sequence.
func SequenceConvertToItemType(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("convertToItemType"); err != nil {
		return vm.IoError(err)
	}
	n, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return target
}

// SequenceCopy is a Sequence method.
//
// copy sets the receiver to be a copy of the given sequence.
func SequenceCopy(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("copy"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return target
}

// SequenceDuplicateIndexes is a Sequence method.
//
// duplicateIndexes inserts a copy of each item in the sequence after its
// position.
func SequenceDuplicateIndexes(vm *VM, target, locals Interface, msg *Message) Interface {
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
	return target
}

// SequenceSetSize is a Sequence method.
//
// setSize sets the size of the sequence.
func SequenceSetSize(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("setSize"); err != nil {
		return vm.IoError(err)
	}
	nn, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return target
}
