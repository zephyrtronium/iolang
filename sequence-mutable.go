package iolang

import (
	"fmt"
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
	other, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	s.Append(other)
	return target
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
