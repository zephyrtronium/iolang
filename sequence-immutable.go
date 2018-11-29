package iolang

import (
	"fmt"
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
