package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"
)

// SequenceAt is a Sequence method.
//
// at returns a value of the sequence as a number.
func SequenceAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	x, ok := s.At(int(arg.Value))
	if ok {
		return vm.NewNumber(x), NoStop
	}
	return vm.Nil, NoStop
}

// SequenceSize is a Sequence method.
//
// size returns the number of items in the sequence.
func SequenceSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(float64(s.Len())), NoStop
}

// SequenceItemSize is a Sequence method.
//
// itemSize returns the size in bytes of each item in the sequence.
func SequenceItemSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(float64(s.ItemSize())), NoStop
}

// SequenceItemType is a Sequence method.
//
// itemType returns the type of the values in the sequence.
func SequenceItemType(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		return vm.NewString("uint8"), NoStop
	case SeqMU16, SeqIU16:
		return vm.NewString("uint16"), NoStop
	case SeqMU32, SeqIU32:
		return vm.NewString("uint32"), NoStop
	case SeqMU64, SeqIU64:
		return vm.NewString("uint64"), NoStop
	case SeqMS8, SeqIS8:
		return vm.NewString("int8"), NoStop
	case SeqMS16, SeqIS16:
		return vm.NewString("int16"), NoStop
	case SeqMS32, SeqIS32:
		return vm.NewString("int32"), NoStop
	case SeqMS64, SeqIS64:
		return vm.NewString("int64"), NoStop
	case SeqMF32, SeqIF32:
		return vm.NewString("float32"), NoStop
	case SeqMF64, SeqIF64:
		return vm.NewString("float64"), NoStop
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceIsMutable is a Sequence method.
//
// isMutable returns whether the sequence is mutable.
func SequenceIsMutable(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.IoBool(s.IsMutable()), NoStop
}

// SequenceCompare is a Sequence method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func SequenceCompare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	t, ok := r.(*Sequence)
	if !ok {
		return vm.NewNumber(float64(PtrCompare(target, r))), NoStop
	}
	la, lb := s.Len(), t.Len()
	ml := la
	if lb < la {
		ml = lb
	}
	// Getting this right is actually very tricky, considering cases like
	// comparing uint64 and float64 - both have values that the other can't
	// represent exactly. It might be worthwhile to revisit this at some point
	// to address inconsistencies, but float64 is the most complete kind
	// available, so for now, we'll make all comparisons in that type.
	for i := 0; i < ml; i++ {
		x, _ := s.At(i)
		y, _ := t.At(i)
		if x < y {
			return vm.NewNumber(-1), NoStop
		}
		if x > y {
			return vm.NewNumber(1), NoStop
		}
	}
	if la < lb {
		return vm.NewNumber(-1), NoStop
	}
	if la > lb {
		return vm.NewNumber(1), NoStop
	}
	return vm.NewNumber(0), NoStop
}

// SequenceCloneAppendSeq is a Sequence method.
//
// cloneAppendSeq creates a new symbol with the elements of the argument appended
// to those of the receiver.
func SequenceCloneAppendSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if n != nil {
		other = vm.NewString(n.String())
	}
	v := vm.NewSequence(s.Value, true, s.Code)
	v.Append(other)
	v.Kind = -v.Kind
	return v, NoStop
}

// SequenceAfterSeq is a Sequence method.
//
// afterSeq returns the portion of the sequence which follows the first
// instance of the argument sequence.
func SequenceAfterSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	p := s.Find(other, 0)
	if p < 0 {
		return vm.Nil, NoStop
	}
	p += other.Len()
	l := s.Len() - p
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), l, l)
	reflect.Copy(v, sv.Slice(p, sv.Len()))
	return vm.NewSequence(v.Interface(), s.IsMutable(), s.Code), NoStop
}

// SequenceAsList is a Sequence method.
//
// asList creates a list containing each element of the sequence.
func SequenceAsList(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{c}, false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		x := make([]Interface, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		x := make([]Interface, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		x := make([]Interface, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{byte(c)}, false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		x := make([]Interface, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, uint16(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		x := make([]Interface, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, uint32(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		x := make([]Interface, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, uint64(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...), NoStop
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(float64(c))
		}
		return vm.NewList(x...), NoStop
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(c)
		}
		return vm.NewList(x...), NoStop
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceAsStruct is a Sequence method.
//
// asStruct reinterprets a sequence as a packed binary structure described by
// the argument list, with list elements alternating between types and slot
// names.
func SequenceAsStruct(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	b := s.Bytes()
	l, err, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	slots := make(Slots, len(l.Value)/2)
	for i := 0; i < len(l.Value)/2; i++ {
		typ, ok := l.Value[2*i].(*Sequence)
		if !ok {
			return vm.RaiseExceptionf("types must be strings, not %s", vm.TypeName(l.Value[2*i]))
		}
		nam, ok := l.Value[2*i+1].(*Sequence)
		if !ok {
			return vm.RaiseExceptionf("names must be strings, not %s", vm.TypeName(l.Value[2*i+1]))
		}
		var v float64
		switch typs := typ.String(); strings.ToLower(typs) {
		case "uint8":
			if len(b) < 1 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(b[0])
			b = b[1:]
		case "uint16":
			if len(b) < 2 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(binary.LittleEndian.Uint16(b))
			b = b[2:]
		case "uint32":
			if len(b) < 4 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(binary.LittleEndian.Uint32(b))
			b = b[4:]
		case "uint64":
			if len(b) < 8 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(binary.LittleEndian.Uint64(b))
			b = b[8:]
		case "int8":
			if len(b) < 1 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(int8(b[0]))
			b = b[1:]
		case "int16":
			if len(b) < 2 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(int16(binary.LittleEndian.Uint16(b)))
			b = b[2:]
		case "int32":
			if len(b) < 4 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(int32(binary.LittleEndian.Uint32(b)))
			b = b[4:]
		case "int64":
			if len(b) < 8 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(int64(binary.LittleEndian.Uint64(b)))
			b = b[8:]
		case "float32":
			if len(b) < 4 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
			b = b[4:]
		case "float64":
			if len(b) < 8 {
				return vm.RaiseExceptionf("struct field %d out of bounds", i)
			}
			v = math.Float64frombits(binary.LittleEndian.Uint64(b))
			b = b[8:]
		default:
			return vm.RaiseExceptionf("unrecognized struct field type %q", typs)
		}
		slots[nam.String()] = vm.NewNumber(v)
	}
	return vm.ObjectWith(slots), NoStop
}

// SequenceAsSymbol is a Sequence method.
//
// asSymbol creates an immutable copy of the sequence.
func SequenceAsSymbol(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewSequence(s.Value, false, s.Code), NoStop
}

// SequenceBeforeSeq is a Sequence method.
//
// beforeSeq returns the portion of the sequence which precedes the first
// instance of the argument sequence.
func SequenceBeforeSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	p := s.Find(other, 0)
	if p < 0 {
		return target, NoStop
	}
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), p, p)
	reflect.Copy(v, sv.Slice(0, p))
	return vm.NewSequence(v.Interface(), s.IsMutable(), s.Code), NoStop
}

// SequenceBeginsWithSeq is a Sequence method.
//
// beginsWithSeq determines whether the sequence begins with the argument
// sequence in the bytewise sense.
func SequenceBeginsWithSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n := other.Len() * other.ItemSize()
	if n > s.Len()*s.ItemSize() {
		return vm.False, NoStop
	}
	a := s.BytesN(n)
	b := other.BytesN(n)
	return vm.IoBool(bytes.Equal(a, b)), NoStop
}

// SequenceBetween is a Sequence method.
//
// between returns the portion of the sequence between the first occurrence of
// the first argument sequence and the first following occurrence of the
// second.
func SequenceBetween(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	k := 0
	if other, ok := r.(*Sequence); ok {
		k = s.Find(other, 0)
		if k < 0 {
			return vm.Nil, NoStop
		}
		k += other.Len()
	} else if r != vm.Nil {
		return vm.RaiseExceptionf("argument 0 to between must be Sequence or nil, not %s", vm.TypeName(r))
	}
	r, stop = msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return r, stop
	}
	l := 0
	if other, ok := r.(*Sequence); ok {
		l = s.Find(other, k)
		if l < 0 {
			// The original returns nil in this case.
			l = s.Len()
		}
	} else if r == vm.Nil {
		l = s.Len()
	} else {
		return vm.RaiseExceptionf("argument 1 to between must be Sequence or nil, not %s", vm.TypeName(r))
	}
	v := reflect.ValueOf(s.Value)
	w := reflect.MakeSlice(v.Type(), l-k, l-k)
	reflect.Copy(w, v.Slice(k, l))
	return vm.NewSequence(w.Interface(), s.IsMutable(), s.Code), NoStop
}

// SequenceBitAt is a Sequence method.
//
// bitAt returns the value of the selected bit within the sequence, 0 or 1. If
// the index is out of bounds, the result is always 0.
func SequenceBitAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	// Would it be more accurate to do these divisions before converting to
	// int? Do we care?
	j := int(n.Value) / 8
	k := uint(n.Value) % 8
	// Explicitly test j in addition to n.Value to account for over/underflow.
	if n.Value < 0 || j < 0 || j >= s.Len()*s.ItemSize() {
		return vm.NewNumber(0), NoStop
	}
	var c byte
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		c = v[j]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		c = byte(v[j])
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		x := math.Float32bits(v[j/4])
		c = byte(x >> uint(j&3*8))
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		x := math.Float64bits(v[j/8])
		c = byte(x >> uint(j&7*8))
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.NewNumber(float64(c >> k & 1)), NoStop
}

// SequenceByteAt is a Sequence method.
//
// byteAt returns the value of the selected byte of the sequence's underlying
// representation, 0 to 255. If the index is out of bounds, the result is 0.
func SequenceByteAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	j := int(n.Value)
	// Explicitly test j rather than n.Value to account for over/underflow.
	if j < 0 || j >= s.Len()*s.ItemSize() {
		return vm.NewNumber(0), NoStop
	}
	var c byte
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		c = v[j]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		c = byte(v[j])
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		x := math.Float32bits(v[j/4])
		c = byte(x >> uint(j&3*8))
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		x := math.Float64bits(v[j/8])
		c = byte(x >> uint(j&7*8))
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.NewNumber(float64(c)), NoStop
}

// SequenceContains is a Sequence method.
//
// contains returns true if any element of the sequence is equal to the given
// Number.
func SequenceContains(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	x := n.Value
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True, NoStop
			}
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, c := range v {
			if c == x {
				return vm.True, NoStop
			}
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.False, NoStop
}

// SequenceContainsSeq is a Sequence method.
//
// containsSeq returns true if the receiver contains the argument sequence.
func SequenceContainsSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.IoBool(s.Find(other, 0) >= 0), NoStop
}

// SequenceEndsWithSeq is a Sequence method.
//
// endsWithSeq determines whether the sequence ends with the argument sequence
// in the bytewise sense.
func SequenceEndsWithSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v := s.Bytes()
	w := other.Bytes()
	if len(w) > len(v) {
		return vm.False, NoStop
	}
	return vm.IoBool(bytes.Equal(v[len(v)-len(w):], w)), NoStop
}

// SequenceExSlice is a Sequence method.
//
// exSlice creates a copy from the first argument index, inclusive, to the
// second argument index, exclusive, or to the end if the second is not given.
func SequenceExSlice(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// We have Sequence.Slice(), but since there's no step argument to these
	// methods and we want a copy, it's better to do it this way.
	s := target.(*Sequence)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a := int(n.Value)
	m := s.Len()
	b := m
	if msg.ArgCount() > 1 {
		n, err, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		b = int(n.Value)
	}
	a = fixSliceIndex(a, 1, m)
	b = fixSliceIndex(b, 1, m)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceFindSeq is a Sequence method.
//
// findSeq locates the first occurrence of the argument sequence in the
// receiver, optionally following a given start index.
func SequenceFindSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a := 0
	if msg.ArgCount() > 1 {
		n, err, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		a = int(n.Value)
		if a < 0 || a > s.Len()-other.Len() {
			return vm.Nil, NoStop
		}
	}
	k := s.Find(other, a)
	if k >= 0 {
		return vm.NewNumber(float64(k)), NoStop
	}
	return vm.Nil, NoStop
}

// SequenceFindSeqs is a Sequence method.
//
// findSeqs finds the first occurrence of any sequence in the argument List and
// returns an object with its "match" slot set to the sequence which matched
// and its "index" slot set to the index of the match.
func SequenceFindSeqs(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	l, err, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a, j := 0, -1
	var m *Sequence
	if msg.ArgCount() > 1 {
		n, err, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		a = int(n.Value)
		if a < 0 {
			return vm.Nil, NoStop
		}
	}
	for _, v := range l.Value {
		x, ok := v.(*Sequence)
		if !ok {
			return vm.RaiseExceptionf("list elements for findSeqs must be Sequence, not %s", vm.TypeName(v))
		}
		k := s.Find(x, a)
		if k >= 0 && (j < 0 || k < j) {
			j = k
			m = x
		}
	}
	if j >= 0 {
		return vm.ObjectWith(Slots{"match": m, "index": vm.NewNumber(float64(j))}), NoStop
	}
	return vm.Nil, NoStop
}

// SequenceForeach is a Sequence method.
//
// foreach performs a loop for each element of the sequence.
func SequenceForeach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseException("foreach requires 2 or 3 arguments")
	}
	s := target.(*Sequence)
	sl := s.Len()
	for k := 0; k < sl; k++ {
		v, _ := s.At(k)
		locals.SetSlot(vn, vm.NewNumber(v))
		if hkn {
			locals.SetSlot(kn, vm.NewNumber(float64(k)))
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
	return result, control
}

// SequenceHash is a Sequence method.
//
// hash returns a hash of the sequence as a number.
func SequenceHash(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	h := fnv.New32()
	h.Write(s.Bytes())
	return vm.NewNumber(float64(uint64(h.Sum32()) << 2)), NoStop // ????????
}

// SequenceInSlice is a Sequence method.
//
// inSlice creates a copy from the first argument index, inclusive, to the
// second argument index, inclusive, or to the end if the second is not given.
func SequenceInSlice(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a := int(n.Value)
	r, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return r, stop
	}
	m := s.Len()
	b := m
	if msg.ArgCount() > 1 {
		n, err, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		b = int(n.Value)
		if b == -1 {
			b = m
		} else {
			b = fixSliceIndex(b+1, 1, m)
		}
	}
	a = fixSliceIndex(a, 1, m)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code), NoStop
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceIsZero is a Sequence method.
//
// isZero returns whether all elements of the sequence are zero.
func SequenceIsZero(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, c := range v {
			if c != 0 {
				return vm.False, NoStop
			}
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.True, NoStop
}

// SequenceOccurrencesOfSeq is a Sequence method.
//
// occurrencesOfSeq counts the number of non-overlapping occurrences of the
// given sequence in the receiver. Raises an exception if the argument is an
// empty sequence.
func SequenceOccurrencesOfSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ol := other.Len()
	if ol == 0 {
		return vm.RaiseException("cannot count occurrences of empty sequence")
	}
	n := 0
	for k := s.Find(other, 0); k >= 0; k = s.Find(other, k+ol) {
		n++
	}
	return vm.NewNumber(float64(n)), NoStop
}

// SequencePack is a Sequence method.
//
// pack forms a packed binary sequence with the given format.
func SequencePack(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	format, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	f := format.String()
	b := []byte{}
	arg := 1
	count := 0
	var ed binary.ByteOrder = binary.LittleEndian
	if len(f) > 0 && f[0] == '*' {
		ed = binary.BigEndian
		f = f[1:]
	}
	for _, c := range f {
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			count = count*10 + int(c) - '0'
			continue
		case 'b', 'B':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				b = append(b, uint8(v.Value))
				arg++
				count--
			}
		case 'h', 'H':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				x := [2]uint8{}
				ed.PutUint16(x[:], uint16(v.Value))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'i', 'I':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				x := [4]uint8{}
				ed.PutUint32(x[:], uint32(v.Value))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'l', 'L':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				x := [8]uint8{}
				ed.PutUint64(x[:], uint64(v.Value))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'f':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				x := [4]uint8{}
				ed.PutUint32(x[:], math.Float32bits(float32(v.Value)))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'F':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, err, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				x := [8]uint8{}
				ed.PutUint64(x[:], math.Float64bits(v.Value))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'c', 'C':
			if count == 0 {
				count++
			}
			var a [4]uint8
			for count > 0 {
				v, err, stop := msg.StringArgAt(vm, locals, arg)
				if stop != NoStop {
					return err, stop
				}
				r, rl := v.FirstRune()
				if rl == 0 {
					return vm.RaiseExceptionf("cannot use empty string with pack format %c", c)
				}
				x := a[:rl]
				utf8.EncodeRune(x, r)
				b = append(b, x...)
				arg++
				count--
			}
		case 's':
			v, err, stop := msg.StringArgAt(vm, locals, arg)
			if stop != NoStop {
				return err, stop
			}
			s := v.String()
			if count == 0 {
				// Io uses a one-length string in this case, but this seems
				// more reasonable for actual use.
				b = append(b, []byte(s)...)
				b = append(b, 0)
			} else {
				// Encode count bytes (not runes) and pad with 0 as necessary.
				x := make([]byte, count)
				copy(x, s)
				b = append(b, x...)
			}
		}
	}
	return vm.NewSequence(b, true, "number"), NoStop
}

// SequenceReverseFindSeq is a Sequence method.
//
// reverseFindSeq locates the last occurrence of the argument sequence in the
// receiver, optionally ending before a given stop index.
func SequenceReverseFindSeq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a := s.Len()
	if msg.ArgCount() > 1 {
		n, err, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return err, stop
		}
		a = int(n.Value)
		if a < 0 || a > s.Len() {
			return vm.Nil, NoStop
		}
		a += other.Len() - 1
	}
	k := s.RFind(other, a)
	if k >= 0 {
		return vm.NewNumber(float64(k)), NoStop
	}
	return vm.Nil, NoStop
}

// SequenceSplitAt is a Sequence method.
//
// splitAt splits the sequence at the given index.
func SequenceSplitAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	idx, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := s.FixIndex(int(idx.Value))
	s1 := vm.NewSequence(s.Value, true, s.Code)
	s2 := vm.NewSequence(s.Value, true, s.Code)
	s1.Slice(0, k, 1)
	s2.Slice(k, s2.Len(), 1)
	return vm.NewList(s1, s2), NoStop
}

// SequenceUnpack is a Sequence method.
//
// unpack reads a packed binary sequence into a List.
func SequenceUnpack(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	format, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	b := s.Bytes()
	f := format.String()
	l := []Interface{}
	count := 0
	var ed binary.ByteOrder = binary.LittleEndian
	if len(f) > 0 && f[0] == '*' {
		ed = binary.BigEndian
		f = f[1:]
	}
	for _, c := range f {
		if len(b) == 0 {
			break
		}
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			count = count*10 + int(c) - '0'
		case 'b':
			if count == 0 {
				count++
			}
			for count > 0 {
				x := int8(b[0])
				l = append(l, vm.NewNumber(float64(x)))
				b = b[1:]
				count--
			}
		case 'B':
			if count == 0 {
				count++
			}
			for count > 0 {
				l = append(l, vm.NewNumber(float64(b[0])))
				b = b[1:]
				count--
			}
		case 'h':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [2]byte{}
				n := copy(v[:], b)
				x := int16(ed.Uint16(v[:]))
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'H':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [2]byte{}
				n := copy(v[:], b)
				x := ed.Uint16(v[:])
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'i':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [4]byte{}
				n := copy(v[:], b)
				x := int32(ed.Uint32(v[:]))
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'I':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [4]byte{}
				n := copy(v[:], b)
				x := ed.Uint32(v[:])
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'l':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [8]byte{}
				n := copy(v[:], b)
				x := int64(ed.Uint64(v[:]))
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'L':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [8]byte{}
				n := copy(v[:], b)
				x := ed.Uint64(v[:])
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'f':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [4]byte{}
				n := copy(v[:], b)
				x := math.Float32frombits(ed.Uint32(v[:]))
				l = append(l, vm.NewNumber(float64(x)))
				b = b[n:]
				count--
			}
		case 'F':
			if count == 0 {
				count++
			}
			for count > 0 {
				v := [8]byte{}
				n := copy(v[:], b)
				x := math.Float64frombits(ed.Uint64(v[:]))
				l = append(l, vm.NewNumber(x))
				b = b[n:]
				count--
			}
		case 'c', 'C':
			if count == 0 {
				count++
			}
			for count > 0 {
				x, n := utf8.DecodeRune(b)
				l = append(l, vm.NewString(string(x)))
				b = b[n:]
				count--
			}
		case 's':
			if count == 0 {
				n := bytes.IndexByte(b, 0)
				if n < 0 {
					n = len(b)
				}
				l = append(l, vm.NewString(string(b[:n])))
				b = b[n:]
			} else {
				v := make([]byte, count)
				n := copy(v, b)
				k := bytes.IndexByte(v, 0)
				if k < 0 {
					k = n
				}
				l = append(l, vm.NewString(string(v[:k])))
				b = b[n:]
			}
		}
	}
	return vm.NewList(l...), NoStop
}

// SequenceWithStruct is a Sequence method.
//
// withStruct creates a packed binary sequence representing the values in the
// argument list, with list elements alternating between types and values. Note
// that while 64-bit types are valid, not all their values can be represented.
func SequenceWithStruct(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	l, err, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	b := make([]byte, 0, len(l.Value)*2)
	p := []byte{7: 0}
	for i := 0; i < len(l.Value)/2; i++ {
		typ, ok := l.Value[2*i].(*Sequence)
		if !ok {
			return vm.RaiseExceptionf("types must be strings, not %s", vm.TypeName(l.Value[2*i]))
		}
		val, ok := l.Value[2*i+1].(*Number)
		if !ok {
			return vm.RaiseExceptionf("values must be numbers, not %s", vm.TypeName(l.Value[2*i]))
		}
		switch typs := typ.String(); strings.ToLower(typs) {
		case "uint8", "int8":
			b = append(b, byte(val.Value))
		case "uint16", "int16":
			binary.LittleEndian.PutUint16(p, uint16(val.Value))
			b = append(b, p[:2]...)
		case "uint32", "int32":
			binary.LittleEndian.PutUint32(p, uint32(val.Value))
			b = append(b, p[:4]...)
		case "uint64", "int64":
			binary.LittleEndian.PutUint64(p, uint64(val.Value))
			b = append(b, p...)
		case "float32":
			binary.LittleEndian.PutUint32(p, math.Float32bits(float32(val.Value)))
			b = append(b, p[:4]...)
		case "float64":
			binary.LittleEndian.PutUint64(p, math.Float64bits(val.Value))
			b = append(b, p...)
		default:
			return vm.RaiseExceptionf("unrecognized struct field type %q", typs)
		}
	}
	return vm.NewSequence(b, true, "latin1"), NoStop
}
