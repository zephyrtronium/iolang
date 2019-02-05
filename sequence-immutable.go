package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"reflect"
	"strings"
)

// SequenceAt is a Sequence method.
//
// at returns a value of the sequence as a number.
func SequenceAt(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	x, ok := s.At(int(arg.Value))
	if ok {
		return vm.NewNumber(x)
	}
	return vm.Nil
}

// SequenceSize is a Sequence method.
//
// size returns the number of items in the sequence.
func SequenceSize(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.NewNumber(float64(s.Len()))
}

// SequenceItemSize is a Sequence method.
//
// itemSize returns the size in bytes of each item in the sequence.
func SequenceItemSize(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.NewNumber(float64(s.ItemSize()))
}

// SequenceItemType is a Sequence method.
//
// itemType returns the type of the values in the sequence.
func SequenceItemType(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		return vm.NewString("uint8")
	case SeqMU16, SeqIU16:
		return vm.NewString("uint16")
	case SeqMU32, SeqIU32:
		return vm.NewString("uint32")
	case SeqMU64, SeqIU64:
		return vm.NewString("uint64")
	case SeqMS8, SeqIS8:
		return vm.NewString("int8")
	case SeqMS16, SeqIS16:
		return vm.NewString("int16")
	case SeqMS32, SeqIS32:
		return vm.NewString("int32")
	case SeqMS64, SeqIS64:
		return vm.NewString("int64")
	case SeqMF32, SeqIF32:
		return vm.NewString("float32")
	case SeqMF64, SeqIF64:
		return vm.NewString("float64")
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceIsMutable is a Sequence method.
//
// isMutable returns whether the sequence is mutable.
func SequenceIsMutable(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.IoBool(s.IsMutable())
}

// SequenceCompare is a Sequence method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func SequenceCompare(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	t, ok := r.(*Sequence)
	if !ok {
		return vm.NewNumber(float64(ptrCompare(target, r)))
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
			return vm.NewNumber(-1)
		}
		if x > y {
			return vm.NewNumber(1)
		}
	}
	if la < lb {
		return vm.NewNumber(-1)
	}
	if la > lb {
		return vm.NewNumber(1)
	}
	return vm.NewNumber(0)
}

// SequenceCloneAppendSeq is a Sequence method.
//
// cloneAppendSeq creates a new symbol with the elements of the argument appended
// to those of the receiver.
func SequenceCloneAppendSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	other, ok := r.(*Sequence)
	if !ok {
		n, ok := r.(*Number)
		if !ok {
			return vm.RaiseException("argument 0 to cloneAppendSeq must be Sequence or Number, not " + vm.TypeName(r))
		}
		other = vm.NewString(n.String())
	}
	v := vm.NewSequence(s.Value, true, s.Code)
	v.Append(other)
	v.Kind = -v.Kind
	return v
}

// SequenceAfterSeq is a Sequence method.
//
// afterSeq returns the portion of the sequence which follows the first
// instance of the argument sequence.
func SequenceAfterSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	p := s.Find(other, 0)
	if p < 0 {
		return vm.Nil
	}
	p += other.Len()
	l := s.Len() - p
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), l, l)
	reflect.Copy(v, sv.Slice(p, sv.Len()))
	return vm.NewSequence(v.Interface(), s.IsMutable(), s.Code)
}

// SequenceAsList is a Sequence method.
//
// asList creates a list containing each element of the sequence.
func SequenceAsList(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{c}, false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		x := make([]Interface, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		x := make([]Interface, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		x := make([]Interface, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{byte(c)}, false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		x := make([]Interface, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, uint16(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		x := make([]Interface, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, uint32(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		x := make([]Interface, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, uint64(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "ascii")
		}
		return vm.NewList(x...)
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(float64(c))
		}
		return vm.NewList(x...)
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		x := make([]Interface, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(c)
		}
		return vm.NewList(x...)
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
func SequenceAsStruct(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	b := s.Bytes()
	l, stop := msg.ListArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return vm.ObjectWith(slots)
}

// SequenceAsSymbol is a Sequence method.
//
// asSymbol creates an immutable copy of the sequence.
func SequenceAsSymbol(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.NewSequence(s.Value, false, s.Code)
}

// SequenceBeforeSeq is a Sequence method.
//
// beforeSeq returns the portion of the sequence which precedes the first
// instance of the argument sequence.
func SequenceBeforeSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	p := s.Find(other, 0)
	if p < 0 {
		return target
	}
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), p, p)
	reflect.Copy(v, sv.Slice(0, p))
	return vm.NewSequence(v.Interface(), s.IsMutable(), s.Code)
}

// SequenceBeginsWithSeq is a Sequence method.
//
// beginsWithSeq determines whether the sequence begins with the argument
// sequence in the bytewise sense.
func SequenceBeginsWithSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	n := other.Len() * other.ItemSize()
	if n > s.Len()*s.ItemSize() {
		return vm.False
	}
	a := s.BytesN(n)
	b := other.BytesN(n)
	return vm.IoBool(bytes.Equal(a, b))
}

// SequenceBetween is a Sequence method.
//
// between returns the portion of the sequence between the first occurrence of
// the first argument sequence and the first following occurrence of the
// second.
func SequenceBetween(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	k := 0
	if other, ok := r.(*Sequence); ok {
		k = s.Find(other, 0)
		if k < 0 {
			return vm.Nil
		}
		k += other.Len()
	} else if r != vm.Nil {
		return vm.RaiseExceptionf("argument 0 to between must be Sequence or nil, not %s", vm.TypeName(r))
	}
	r, ok = CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return r
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
	return vm.NewSequence(w.Interface(), s.IsMutable(), s.Code)
}

// SequenceBitAt is a Sequence method.
//
// bitAt returns the value of the selected bit within the sequence, 0 or 1. If
// the index is out of bounds, the result is always 0.
func SequenceBitAt(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	// Would it be more accurate to do these divisions before converting to
	// int? Do we care?
	j := int(n.Value) / 8
	k := uint(n.Value) % 8
	// Explicitly test j in addition to n.Value to account for over/underflow.
	if n.Value < 0 || j < 0 || j >= s.Len()*s.ItemSize() {
		return vm.NewNumber(0)
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
	return vm.NewNumber(float64(c >> k & 1))
}

// SequenceByteAt is a Sequence method.
//
// byteAt returns the value of the selected byte of the sequence's underlying
// representation, 0 to 255. If the index is out of bounds, the result is 0.
func SequenceByteAt(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	j := int(n.Value)
	// Explicitly test j rather than n.Value to account for over/underflow.
	if j < 0 || j >= s.Len()*s.ItemSize() {
		return vm.NewNumber(0)
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
	return vm.NewNumber(float64(c))
}

// SequenceContains is a Sequence method.
//
// contains returns true if any element of the sequence is equal to the given
// Number.
func SequenceContains(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	x := n.Value
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, c := range v {
			if c == x {
				return vm.True
			}
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.False
}

// SequenceContainsSeq is a Sequence method.
//
// containsSeq returns true if the receiver contains the argument sequence.
func SequenceContainsSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.IoBool(s.Find(other, 0) >= 0)
}

// SequenceEndsWithSeq is a Sequence method.
//
// endsWithSeq determines whether the sequence ends with the argument sequence
// in the bytewise sense.
func SequenceEndsWithSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	v := s.Bytes()
	w := other.Bytes()
	if len(w) > len(v) {
		return vm.False
	}
	return vm.IoBool(bytes.Equal(v[len(v)-len(w):], w))
}

// SequenceExSlice is a Sequence method.
//
// exSlice creates a copy from the first argument index, inclusive, to the
// second argument index, exclusive, or to the end if the second is not given.
func SequenceExSlice(vm *VM, target, locals Interface, msg *Message) Interface {
	// We have Sequence.Slice(), but since there's no step argument to these
	// methods and we want a copy, it's better to do it this way.
	s := target.(*Sequence)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a := int(n.Value)
	m := s.Len()
	b := m
	if msg.ArgCount() > 1 {
		n, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != nil {
			return stop
		}
		b = int(n.Value)
	}
	a = fixSliceIndex(a, 1, m)
	b = fixSliceIndex(b, 1, m)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
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
func SequenceFindSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a := 0
	if msg.ArgCount() > 1 {
		n, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != nil {
			return stop
		}
		a = int(n.Value)
		if a < 0 || a > s.Len()-other.Len() {
			return vm.Nil
		}
	}
	k := s.Find(other, a)
	if k >= 0 {
		return vm.NewNumber(float64(k))
	}
	return vm.Nil
}

// SequenceFindSeqs is a Sequence method.
//
// findSeqs finds the first occurrence of any sequence in the argument List and
// returns an object with its "match" slot set to the sequence which matched
// and its "index" slot set to the index of the match.
func SequenceFindSeqs(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	l, stop := msg.ListArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a, j := 0, -1
	var m *Sequence
	if msg.ArgCount() > 1 {
		n, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != nil {
			return stop
		}
		a = int(n.Value)
		if a < 0 {
			return vm.Nil
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
		return vm.ObjectWith(Slots{"match": m, "index": vm.NewNumber(float64(j))})
	}
	return vm.Nil
}

// SequenceForeach is a Sequence method.
//
// foreach performs a loop for each element of the sequence.
func SequenceForeach(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseException("foreach requires 2 or 3 arguments")
	}
	s := target.(*Sequence)
	sl := s.Len()
	for k := 0; k < sl; k++ {
		v, _ := s.At(k)
		SetSlot(locals, vn, vm.NewNumber(v))
		if hkn {
			SetSlot(locals, kn, vm.NewNumber(float64(k)))
		}
		result = ev.Eval(vm, locals)
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
	return result
}

// SequenceHash is a Sequence method.
//
// hash returns a hash of the sequence as a number.
func SequenceHash(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	h := fnv.New32()
	h.Write(s.Bytes())
	return vm.NewNumber(float64(uint64(h.Sum32()) << 2)) // ????????
}

// SequenceInSlice is a Sequence method.
//
// inSlice creates a copy from the first argument index, inclusive, to the
// second argument index, inclusive, or to the end if the second is not given.
func SequenceInSlice(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	n, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a := int(n.Value)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return r
	}
	m := s.Len()
	b := m
	if msg.ArgCount() > 1 {
		n, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != nil {
			return stop
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
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SequenceIsZero is a Sequence method.
//
// isZero returns whether all elements of the sequence are zero.
func SequenceIsZero(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return vm.True
}

// SequenceReverseFindSeq is a Sequence method.
//
// reverseFindSeq locates the last occurrence of the argument sequence in the
// receiver, optionally ending before a given stop index.
func SequenceReverseFindSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a := s.Len()
	if msg.ArgCount() > 1 {
		n, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != nil {
			return stop
		}
		a = int(n.Value)
		if a < 0 || a > s.Len() {
			return vm.Nil
		}
		a += other.Len() - 1
	}
	k := s.RFind(other, a)
	if k >= 0 {
		return vm.NewNumber(float64(k))
	}
	return vm.Nil
}

// SequenceWithStruct is a Sequence method.
//
// withStruct creates a packed binary sequence representing the values in the
// argument list, with list elements alternating between types and values. Note
// that while 64-bit types are valid, not all their values can be represented.
func SequenceWithStruct(vm *VM, target, locals Interface, msg *Message) Interface {
	l, stop := msg.ListArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	return vm.NewSequence(b, true, "ascii")
}
