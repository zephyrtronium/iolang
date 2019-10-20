package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SequenceAt is a Sequence method.
//
// at returns a value of the sequence as a number.
func SequenceAt(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	x, ok := s.At(int(arg))
	unholdSeq(s.Mutable, target)
	if ok {
		return vm.NewNumber(x)
	}
	return vm.Nil
}

// SequenceSize is a Sequence method.
//
// size returns the number of items in the sequence.
func SequenceSize(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	n := s.Len()
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(n))
}

// SequenceItemSize is a Sequence method.
//
// itemSize returns the size in bytes of each item in the sequence.
func SequenceItemSize(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	n := s.ItemSize()
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(n))
}

// SequenceItemType is a Sequence method.
//
// itemType returns the type of the values in the sequence.
func SequenceItemType(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	switch s.Value.(type) {
	case []byte:
		return vm.NewString("uint8")
	case []uint16:
		return vm.NewString("uint16")
	case []uint32:
		return vm.NewString("uint32")
	case []uint64:
		return vm.NewString("uint64")
	case []int8:
		return vm.NewString("int8")
	case []int16:
		return vm.NewString("int16")
	case []int32:
		return vm.NewString("int32")
	case []int64:
		return vm.NewString("int64")
	case []float32:
		return vm.NewString("float32")
	case []float64:
		return vm.NewString("float64")
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// SequenceIsMutable is a Sequence method.
//
// isMutable returns whether the sequence is mutable.
func SequenceIsMutable(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	r := s.IsMutable()
	unholdSeq(s.Mutable, target)
	return vm.IoBool(r)
}

// SequenceCompare is a Sequence method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func SequenceCompare(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	s := holdSeq(target)
	r.Lock()
	t, ok := r.Value.(Sequence)
	if !ok {
		r.Unlock()
		return vm.NewNumber(float64(PtrCompare(target, r)))
	}
	n := s.Compare(t)
	r.Unlock()
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(n))
}

// SequenceCloneAppendSeq is a Sequence method.
//
// cloneAppendSeq creates a new symbol with the elements of the argument appended
// to those of the receiver.
func SequenceCloneAppendSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, n, obj, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.Value == nil {
		other = Sequence{Value: []byte(strconv.FormatFloat(n, 'g', -1, 64)), Mutable: false, Code: "utf8"}
	}
	s := holdSeq(target)
	v := Sequence{Value: copySeqVal(s.Value), Mutable: true, Code: s.Code}
	unholdSeq(s.Mutable, target)
	obj.Lock() // unnecessary if we got n, but this is easy and fast
	v = v.Append(other)
	obj.Unlock()
	v.Mutable = false
	return vm.SequenceObject(v)
}

// SequenceAfterSeq is a Sequence method.
//
// afterSeq returns the portion of the sequence which follows the first
// instance of the argument sequence.
func SequenceAfterSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	if other.IsMutable() {
		obj.Lock()
	}
	ol := other.Len()
	if other.IsMutable() {
		obj.Unlock()
	}
	p := s.Find(other, 0)
	if p < 0 {
		unholdSeq(s.Mutable, target)
		return vm.Nil
	}
	p += ol
	l := s.Len() - p
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), l, l)
	reflect.Copy(v, sv.Slice(p, sv.Len()))
	code := s.Code
	unholdSeq(s.Mutable, target)
	return vm.NewSequence(v.Interface(), s.Mutable, code)
}

// SequenceAsList is a Sequence method.
//
// asList creates a list containing each element of the sequence.
func SequenceAsList(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	switch v := s.Value.(type) {
	case []byte:
		x := make([]*Object, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{c}, false, "latin1")
		}
		return vm.NewList(x...)
	case []uint16:
		x := make([]*Object, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []uint32:
		x := make([]*Object, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []uint64:
		x := make([]*Object, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, c)
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []int8:
		x := make([]*Object, len(v))
		for i, c := range v {
			x[i] = vm.NewSequence([]byte{byte(c)}, false, "latin1")
		}
		return vm.NewList(x...)
	case []int16:
		x := make([]*Object, len(v))
		p := []byte{1: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint16(p, uint16(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []int32:
		x := make([]*Object, len(v))
		p := []byte{3: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint32(p, uint32(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []int64:
		x := make([]*Object, len(v))
		p := []byte{7: 0}
		for i, c := range v {
			binary.LittleEndian.PutUint64(p, uint64(c))
			x[i] = vm.NewSequence(bytes.TrimRight(p, "\x00"), false, "latin1")
		}
		return vm.NewList(x...)
	case []float32:
		x := make([]*Object, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(float64(c))
		}
		return vm.NewList(x...)
	case []float64:
		x := make([]*Object, len(v))
		for i, c := range v {
			x[i] = vm.NewNumber(c)
		}
		return vm.NewList(x...)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// SequenceAsStruct is a Sequence method.
//
// asStruct reinterprets a sequence as a packed binary structure described by
// the argument list, with list elements alternating between types and slot
// names.
func SequenceAsStruct(vm *VM, target, locals *Object, msg *Message) *Object {
	l, obj, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	b := s.Bytes()
	unholdSeq(s.Mutable, target)
	obj.Lock()
	defer obj.Unlock()
	slots := make(Slots, len(l)/2)
	for i := 0; i < len(l)/2; i++ {
		typ, ok := l[2*i].Value.(Sequence)
		if !ok {
			return vm.RaiseExceptionf("types must be strings, not %s", vm.TypeName(l[2*i]))
		}
		nam, ok := l[2*i+1].Value.(Sequence)
		if !ok {
			return vm.RaiseExceptionf("names must be strings, not %s", vm.TypeName(l[2*i+1]))
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
func SequenceAsSymbol(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	r := vm.NewSequence(s.Value, false, s.Code)
	unholdSeq(s.Mutable, target)
	return r
}

// SequenceBeforeSeq is a Sequence method.
//
// beforeSeq returns the portion of the sequence which precedes the first
// instance of the argument sequence.
func SequenceBeforeSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	if other.IsMutable() {
		obj.Lock()
	}
	p := s.Find(other, 0)
	if other.IsMutable() {
		obj.Unlock()
	}
	if p < 0 {
		unholdSeq(s.Mutable, target)
		return target
	}
	sv := reflect.ValueOf(s.Value)
	v := reflect.MakeSlice(sv.Type(), p, p)
	reflect.Copy(v, sv.Slice(0, p))
	code := s.Code
	unholdSeq(s.Mutable, target)
	return vm.NewSequence(v.Interface(), s.Mutable, code)
}

// SequenceBeginsWithSeq is a Sequence method.
//
// beginsWithSeq determines whether the sequence begins with the argument
// sequence in the bytewise sense.
func SequenceBeginsWithSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	obj.Lock()
	n := other.Len() * other.ItemSize()
	if n > s.Len()*s.ItemSize() {
		unholdSeq(s.Mutable, target)
		obj.Unlock()
		return vm.False
	}
	a := s.BytesN(n)
	b := other.BytesN(n)
	unholdSeq(s.Mutable, target)
	obj.Unlock()
	return vm.IoBool(bytes.Equal(a, b))
}

// SequenceBetween is a Sequence method.
//
// between returns the portion of the sequence between the first occurrence of
// the first argument sequence and the first following occurrence of the
// second.
func SequenceBetween(vm *VM, target, locals *Object, msg *Message) *Object {
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	k := 0
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	if other, ok := r.Value.(Sequence); ok {
		if other.IsMutable() {
			r.Lock()
		}
		k = s.Find(other, 0)
		if other.IsMutable() {
			r.Unlock()
		}
		if k < 0 {
			return vm.Nil
		}
		k += other.Len()
	} else if r != vm.Nil {
		return vm.RaiseExceptionf("argument 0 to between must be Sequence or nil, not %s", vm.TypeName(r))
	}
	r, stop = msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	l := 0
	if other, ok := r.Value.(Sequence); ok {
		if other.IsMutable() {
			r.Lock()
		}
		l = s.Find(other, k)
		if other.IsMutable() {
			r.Unlock()
		}
		if l < 0 {
			return vm.Nil
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
func SequenceBitAt(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	// Would it be more accurate to do these divisions before converting to
	// int? Do we care?
	j := int(n) / 8
	k := uint(n) % 8
	s := holdSeq(target)
	// Explicitly test j in addition to n to account for over/underflow.
	if n < 0 || j < 0 || j >= s.Len()*s.ItemSize() {
		unholdSeq(s.Mutable, target)
		return vm.NewNumber(0)
	}
	var c byte
	switch v := s.Value.(type) {
	case []byte:
		c = v[j]
	case []uint16:
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case []uint32:
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case []uint64:
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case []int8:
		c = byte(v[j])
	case []int16:
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case []int32:
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case []int64:
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case []float32:
		x := math.Float32bits(v[j/4])
		c = byte(x >> uint(j&3*8))
	case []float64:
		x := math.Float64bits(v[j/8])
		c = byte(x >> uint(j&7*8))
	default:
		panic(fmt.Sprintf("unknown sequence kind %T", s.Value))
	}
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(c >> k & 1))
}

// SequenceByteAt is a Sequence method.
//
// byteAt returns the value of the selected byte of the sequence's underlying
// representation, 0 to 255. If the index is out of bounds, the result is 0.
func SequenceByteAt(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	j := int(n)
	s := holdSeq(target)
	// Explicitly test j rather than n to account for over/underflow.
	if j < 0 || j >= s.Len()*s.ItemSize() {
		return vm.NewNumber(0)
	}
	var c byte
	switch v := s.Value.(type) {
	case []byte:
		c = v[j]
	case []uint16:
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case []uint32:
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case []uint64:
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case []int8:
		c = byte(v[j])
	case []int16:
		x := v[j/2]
		c = byte(x >> uint(j&1*8))
	case []int32:
		x := v[j/4]
		c = byte(x >> uint(j&3*8))
	case []int64:
		x := v[j/8]
		c = byte(x >> uint(j&7*8))
	case []float32:
		x := math.Float32bits(v[j/4])
		c = byte(x >> uint(j&3*8))
	case []float64:
		x := math.Float64bits(v[j/8])
		c = byte(x >> uint(j&7*8))
	default:
		panic(fmt.Sprintf("unknown sequence kind %T", s.Value))
	}
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(c))
}

// SequenceContains is a Sequence method.
//
// contains returns true if any element of the sequence is equal to the given
// Number.
func SequenceContains(vm *VM, target, locals *Object, msg *Message) *Object {
	x, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	switch v := s.Value.(type) {
	case []byte:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []uint16:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []uint32:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []uint64:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []int8:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []int16:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []int32:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []int64:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []float32:
		for _, c := range v {
			if float64(c) == x {
				return vm.True
			}
		}
	case []float64:
		for _, c := range v {
			if c == x {
				return vm.True
			}
		}
	default:
		panic(fmt.Sprintf("unknown sequence kind %T", s.Value))
	}
	return vm.False
}

// SequenceContainsSeq is a Sequence method.
//
// containsSeq returns true if the receiver contains the argument sequence.
func SequenceContainsSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, 0)
	unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Unlock()
	}
	return vm.IoBool(k >= 0)
}

// SequenceEndsWithSeq is a Sequence method.
//
// endsWithSeq determines whether the sequence ends with the argument sequence
// in the bytewise sense.
func SequenceEndsWithSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	v := s.Bytes()
	unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Lock()
	}
	w := other.Bytes()
	if other.IsMutable() {
		obj.Unlock()
	}
	if len(w) > len(v) {
		return vm.False
	}
	return vm.IoBool(bytes.Equal(v[len(v)-len(w):], w))
}

// SequenceExSlice is a Sequence method.
//
// exSlice creates a copy from the first argument index, inclusive, to the
// second argument index, exclusive, or to the end if the second is not given.
func SequenceExSlice(vm *VM, target, locals *Object, msg *Message) *Object {
	// We have Sequence.Slice(), but since there's no step argument to these
	// methods and we want a copy, it's better to do it this way.
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	a := int(n)
	// We acquire target's lock to get the sequence length since we need it to
	// decide what to do with a second argument, but we don't want to hold the
	// lock while evaluating that argument.
	s := holdSeq(target)
	m := s.Len()
	unholdSeq(s.Mutable, target)
	b := m
	if msg.ArgCount() > 1 {
		n, exc, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		b = int(n)
	}
	a = fixSliceIndex(a, 1, m)
	b = fixSliceIndex(b, 1, m)
	// Acquire the target's lock again while reading its value.
	target.Lock()
	defer target.Unlock()
	switch v := s.Value.(type) {
	case []byte:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint16:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int8:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int16:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []float32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []float64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// SequenceFindSeq is a Sequence method.
//
// findSeq locates the first occurrence of the argument sequence in the
// receiver, optionally following a given start index.
func SequenceFindSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	ol := other.Len()
	if other.IsMutable() {
		obj.Unlock()
	}
	s := holdSeq(target)
	a := 0
	if msg.ArgCount() > 1 {
		unholdSeq(s.Mutable, target)
		n, exc, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		a = int(n)
		if s.IsMutable() {
			target.Lock()
		}
		if a < 0 || a > s.Len()-ol {
			unholdSeq(s.Mutable, target)
			return vm.Nil
		}
	}
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.Find(other, a)
	unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Unlock()
	}
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
func SequenceFindSeqs(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	l, obj, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	a, j := 0, -1
	var m *Object
	if msg.ArgCount() > 1 {
		n, exc, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		a = int(n)
		if a < 0 {
			return vm.Nil
		}
	}
	obj.Lock()
	for _, v := range l {
		x, ok := v.Value.(Sequence)
		if !ok {
			obj.Unlock()
			return vm.RaiseExceptionf("list elements for findSeqs must be Sequence, not %s", vm.TypeName(v))
		}
		if x.IsMutable() {
			v.Lock()
		}
		k := s.Find(x, a)
		if x.IsMutable() {
			v.Unlock()
		}
		if k >= 0 && (j < 0 || k < j) {
			j = k
			m = v
			break
		}
	}
	obj.Unlock()
	if j >= 0 {
		return vm.ObjectWith(Slots{"match": m, "index": vm.NewNumber(float64(j))})
	}
	return vm.Nil
}

// SequenceForeach is a Sequence method.
//
// foreach performs a loop for each element of the sequence.
func SequenceForeach(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseExceptionf("foreach requires 2 or 3 arguments")
	}
	s := holdSeq(target)
	k := 0
	var control Stop
	for {
		v, ok := s.At(k)
		unholdSeq(s.Mutable, target)
		if !ok {
			break
		}
		locals.SetSlot(vn, vm.NewNumber(v))
		if hkn {
			locals.SetSlot(kn, vm.NewNumber(float64(k)))
		}
		result, control = ev.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
		k++
		if s.IsMutable() {
			s = holdSeq(target)
		}
	}
	unholdSeq(s.Mutable, target)
	return result
}

// SequenceHash is a Sequence method.
//
// hash returns a hash of the sequence as a number.
func SequenceHash(vm *VM, target, locals *Object, msg *Message) *Object {
	h := fnv.New32()
	s := holdSeq(target)
	h.Write(s.Bytes())
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(float64(uint64(h.Sum32()) << 2)) // ????????
}

// SequenceInSlice is a Sequence method.
//
// inSlice creates a copy from the first argument index, inclusive, to the
// second argument index, inclusive, or to the end if the second is not given.
func SequenceInSlice(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	a := int(n)
	s := holdSeq(target)
	m := s.Len()
	unholdSeq(s.Mutable, target)
	b := m
	if msg.ArgCount() > 1 {
		n, exc, stop = msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		b = int(n)
		if b == -1 {
			b = m
		} else {
			b = fixSliceIndex(b+1, 1, m)
		}
	}
	if s.IsMutable() {
		target.Lock()
		defer target.Unlock()
	}
	a = fixSliceIndex(a, 1, m)
	switch v := s.Value.(type) {
	case []byte:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint16:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []uint64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int8:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int16:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []int64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []float32:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	case []float64:
		return vm.NewSequence(v[a:b], s.IsMutable(), s.Code)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// SequenceIsZero is a Sequence method.
//
// isZero returns whether all elements of the sequence are zero.
func SequenceIsZero(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	switch v := s.Value.(type) {
	case []byte:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []uint16:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []uint32:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []uint64:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []int8:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []int16:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []int32:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []int64:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []float32:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	case []float64:
		for _, c := range v {
			if c != 0 {
				return vm.False
			}
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return vm.True
}

// SequenceOccurrencesOfSeq is a Sequence method.
//
// occurrencesOfSeq counts the number of non-overlapping occurrences of the
// given sequence in the receiver. Raises an exception if the argument is an
// empty sequence.
func SequenceOccurrencesOfSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	s := holdSeq(target)
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
	}
	ol := other.Len()
	if ol == 0 {
		unholdSeq(s.Mutable, target)
		if other.IsMutable() {
			obj.Unlock()
		}
		return vm.RaiseExceptionf("cannot count occurrences of empty sequence")
	}
	n := 0
	for k := s.Find(other, 0); k >= 0; k = s.Find(other, k+ol) {
		n++
	}
	unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Unlock()
	}
	return vm.NewNumber(float64(n))
}

// SequencePack is a Sequence method.
//
// pack forms a packed binary sequence with the given format.
func SequencePack(vm *VM, target, locals *Object, msg *Message) *Object {
	f, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
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
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				b = append(b, uint8(v))
				arg++
				count--
			}
		case 'h', 'H':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				x := [2]uint8{}
				ed.PutUint16(x[:], uint16(v))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'i', 'I':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				x := [4]uint8{}
				ed.PutUint32(x[:], uint32(v))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'l', 'L':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				x := [8]uint8{}
				ed.PutUint64(x[:], uint64(v))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'f':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				x := [4]uint8{}
				ed.PutUint32(x[:], math.Float32bits(float32(v)))
				b = append(b, x[:]...)
				arg++
				count--
			}
		case 'F':
			if count == 0 {
				count++
			}
			for count > 0 {
				v, exc, stop := msg.NumberArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(exc, stop)
				}
				x := [8]uint8{}
				ed.PutUint64(x[:], math.Float64bits(v))
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
				v, obj, stop := msg.SequenceArgAt(vm, locals, arg)
				if stop != NoStop {
					return vm.Stop(obj, stop)
				}
				if v.IsMutable() {
					obj.Lock()
				}
				r, rl := v.FirstRune()
				if v.IsMutable() {
					obj.Unlock()
				}
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
			s, exc, stop := msg.StringArgAt(vm, locals, arg)
			if stop != NoStop {
				return vm.Stop(exc, stop)
			}
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
	return vm.NewSequence(b, true, "number")
}

// SequenceReverseFindSeq is a Sequence method.
//
// reverseFindSeq locates the last occurrence of the argument sequence in the
// receiver, optionally ending before a given stop index.
func SequenceReverseFindSeq(vm *VM, target, locals *Object, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	a := s.Len()
	unholdSeq(s.Mutable, target)
	if msg.ArgCount() > 1 {
		n, exc, stop := msg.NumberArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		a = int(n)
		if a < 0 || a > s.Len() {
			return vm.Nil
		}
		if other.IsMutable() {
			obj.Lock()
		}
		a += other.Len() - 1
		if other.IsMutable() {
			obj.Unlock()
		}
	}
	s = holdSeq(target)
	if other.IsMutable() {
		obj.Lock()
	}
	k := s.RFind(other, a)
	unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Unlock()
	}
	if k >= 0 {
		return vm.NewNumber(float64(k))
	}
	return vm.Nil
}

// SequenceSplitAt is a Sequence method.
//
// splitAt splits the sequence at the given index.
func SequenceSplitAt(vm *VM, target, locals *Object, msg *Message) *Object {
	idx, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	s := holdSeq(target)
	k := s.FixIndex(int(idx))
	v := reflect.ValueOf(s.Value)
	s1 := vm.NewSequence(v.Slice(0, k).Interface(), true, s.Code)
	s2 := vm.NewSequence(v.Slice(k, v.Len()).Interface(), true, s.Code)
	unholdSeq(s.Mutable, target)
	return vm.NewList(s1, s2)
}

// SequenceUnpack is a Sequence method.
//
// unpack reads a packed binary sequence into a List.
func SequenceUnpack(vm *VM, target, locals *Object, msg *Message) *Object {
	f, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	s := holdSeq(target)
	b := s.Bytes()
	unholdSeq(s.Mutable, target)
	l := []*Object{}
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
	return vm.NewList(l...)
}

// SequenceWithStruct is a Sequence method.
//
// withStruct creates a packed binary sequence representing the values in the
// argument list, with list elements alternating between types and values. Note
// that while 64-bit types are valid, not all their values can be represented.
func SequenceWithStruct(vm *VM, target, locals *Object, msg *Message) *Object {
	l, obj, stop := msg.ListArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	b := make([]byte, 0, len(l)*2)
	p := []byte{7: 0}
	obj.Lock()
	defer obj.Unlock()
	for i := 0; i < len(l)/2; i++ {
		vi := l[2*i]
		// Check that the list doesn't contain itself so that we don't try to
		// acquire its lock while already holding it.
		if vi == obj {
			return vm.RaiseExceptionf("types must be strings, not %s", vm.TypeName(vi))
		}
		vi.Lock()
		typ, ok := vi.Value.(Sequence)
		if !ok {
			vi.Unlock()
			return vm.RaiseExceptionf("types must be strings, not %s", vm.TypeName(vi))
		}
		typs := typ.String()
		vi.Unlock()
		vi = l[2*i+1]
		vi.Lock()
		val, ok := vi.Value.(float64)
		vi.Unlock()
		if !ok {
			return vm.RaiseExceptionf("values must be numbers, not %s", vm.TypeName(vi))
		}
		switch strings.ToLower(typs) {
		case "uint8", "int8":
			b = append(b, byte(val))
		case "uint16", "int16":
			binary.LittleEndian.PutUint16(p, uint16(val))
			b = append(b, p[:2]...)
		case "uint32", "int32":
			binary.LittleEndian.PutUint32(p, uint32(val))
			b = append(b, p[:4]...)
		case "uint64", "int64":
			binary.LittleEndian.PutUint64(p, uint64(val))
			b = append(b, p...)
		case "float32":
			binary.LittleEndian.PutUint32(p, math.Float32bits(float32(val)))
			b = append(b, p[:4]...)
		case "float64":
			binary.LittleEndian.PutUint64(p, math.Float64bits(val))
			b = append(b, p...)
		default:
			return vm.RaiseExceptionf("unrecognized struct field type %q", typs)
		}
	}
	return vm.NewSequence(b, true, "latin1")
}
