package iolang

import (
	"fmt"
)

// SequenceAt is a Sequence method.
//
// at returns a value of the sequence as a number.
func SequenceAt(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	x := s.At(vm, int(arg.Value))
	if x != nil {
		return x
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
