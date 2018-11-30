package iolang

import (
	"fmt"
	"math"
)

// CheckNumeric checks that the sequence is numeric, optionally requiring the
// sequence to be mutable as well.
func (s *Sequence) CheckNumeric(name string, mutable bool) error {
	if s.Code == "number" {
		if mutable {
			return s.CheckMutable(name)
		}
		return nil
	}
	return fmt.Errorf("%q not valid on non-number encodings", name)
}

// MapUnary replaces each value of the sequence with the result of applying op.
// Values are converted to float64 and back to the appropriate type.
func (s *Sequence) MapUnary(op func(float64) float64) {
	if !s.IsMutable() {
		panic("can't modify immutable sequence")
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i, c := range v {
			v[i] = byte(op(float64(c)))
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i, c := range v {
			v[i] = uint16(op(float64(c)))
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i, c := range v {
			v[i] = uint32(op(float64(c)))
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i, c := range v {
			v[i] = uint64(op(float64(c)))
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i, c := range v {
			v[i] = int8(op(float64(c)))
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i, c := range v {
			v[i] = int16(op(float64(c)))
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i, c := range v {
			v[i] = int32(op(float64(c)))
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i, c := range v {
			v[i] = int64(op(float64(c)))
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i, c := range v {
			v[i] = float32(op(float64(c)))
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i, c := range v {
			v[i] = op(c)
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// MapBinary replaces each value of the sequence with the result of applying op
// with the respective value of t, or with the given default value if past the
// end of t. Values are converted to float64 and back to the appropriate type.
func (s *Sequence) MapBinary(op func(float64, float64) float64, t *Sequence, def float64) {
	if !s.IsMutable() {
		panic("can't modify immutable sequence")
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = byte(op(float64(c), x))
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint16(op(float64(c), x))
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint32(op(float64(c), x))
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint64(op(float64(c), x))
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int8(op(float64(c), x))
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int16(op(float64(c), x))
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int32(op(float64(c), x))
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int64(op(float64(c), x))
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = float32(op(float64(c), x))
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = op(c, x)
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// SeqOrNumArgAt evaluates the given argument, then returns it as a Sequence
// or Number, or a raised exception if it is neither, or a return or raised
// exception if one occurs during evaluation.
func (m *Message) SeqOrNumArgAt(vm *VM, locals Interface, n int) (*Sequence, *Number, Interface) {
	r, ok := CheckStop(m.EvalArgAt(vm, locals, n), LoopStops)
	if !ok {
		return nil, nil, r
	}
	switch v := r.(type) {
	case *Sequence:
		return v, nil, nil
	case *Number:
		return nil, v, nil
	}
	return nil, nil, vm.RaiseExceptionf("argument %d to %s must be Sequence or Number, not %s", n, m.Name(), vm.TypeName(r))
}

// SequenceStarStarEq is a Sequence method.
//
// **= sets each element of the receiver to its value raised to the power of the
// respective element of the argument.
func SequenceStarStarEq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("**=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if t != nil {
		s.MapBinary(math.Pow, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return math.Pow(x, y) })
	}
	return target
}

// SequenceStarEq is a Sequence method.
//
// *= sets each element of the receiver to its value times the respective
// element of the argument.
func SequenceStarEq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("*=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x * y }, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x * y })
	}
	return target
}

// SequencePlusEq is a Sequence method.
//
// += sets each element of the receiver to its value plus the respective
// element of the argument.
func SequencePlusEq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("+=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x + y }, t, 0)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x + y })
	}
	return target
}

// SequenceMinusEq is a Sequence method.
//
// -= sets each element of the receiver to its value minus the respective
// element of the argument.
func SequenceMinusEq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("-=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x - y }, t, 0)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x - y })
	}
	return target
}

// SequenceSlashEq is a Sequence method.
//
// /= sets each element of the receiver to its value divided by the respective
// element of the argument.
func SequenceSlashEq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("/=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x / y }, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x / y })
	}
	return target
}

// SequencePairwiseMax is a Sequence method.
//
// Max sets each element of the receiver to the greater of the receiver element
// and the respective argument element.
func SequencePairwiseMax(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("Max", true); err != nil {
		return vm.IoError(err)
	}
	t, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	s.MapBinary(math.Max, t, math.Inf(-1))
	return target
}

// SequencePairwiseMin is a Sequence method.
//
// Min sets each element of the receiver to the lesser of the receiver element
// and the respective argument element.
func SequencePairwiseMin(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("Min", true); err != nil {
		return vm.IoError(err)
	}
	t, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	s.MapBinary(math.Min, t, math.Inf(0))
	return target
}

// SequenceAbs is a Sequence method.
//
// abs sets each element of the receiver to its absolute value.
func SequenceAbs(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("abs", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Abs)
	return s
}

// SequenceAcos is a Sequence method.
//
// acos sets each element of the receiver to its arc-cosine.
func SequenceAcos(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("acos", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Acos)
	return s
}

// SequenceCos is a Sequence method.
//
// cos sets each element of the receiver to its cosine.
func SequenceCos(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("cos", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Cos)
	return s
}
