package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	p := s.Find(other)
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
