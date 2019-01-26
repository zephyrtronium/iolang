package iolang

import (
	"encoding/binary"
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

// SequenceAsBinaryNumber is a Sequence method.
//
// asBinaryNumber reinterprets the first eight bytes of the sequence as an
// IEEE-754 binary64 floating-point value and returns the appropriate Number.
func SequenceAsBinaryNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	v := s.Bytes()
	if len(v) < 8 {
		return vm.RaiseExceptionf("need 8 bytes in sequence, have only %d", len(v))
	}
	x := binary.LittleEndian.Uint64(v)
	return vm.NewNumber(math.Float64frombits(x))
}

// SequenceAsBinarySignedInteger is a Sequence method.
//
// asBinarySignedInteger reinterprets the bytes of the sequence as a signed
// integer. The byte size of the sequence must be 1, 2, 4, or 8.
func SequenceAsBinarySignedInteger(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	v := s.Bytes()
	switch len(v) {
	case 1:
		return vm.NewNumber(float64(int8(v[0])))
	case 2:
		return vm.NewNumber(float64(int16(binary.LittleEndian.Uint16(v))))
	case 4:
		return vm.NewNumber(float64(int32(binary.LittleEndian.Uint32(v))))
	case 8:
		return vm.NewNumber(float64(int64(binary.LittleEndian.Uint64(v))))
	}
	return vm.RaiseException("asBinarySignedInteger receiver must be Sequence of 1, 2, 4, or 8 bytes")
}

// SequenceAsBinaryUnsignedInteger is a Sequence method.
//
// asBinaryUnsignedInteger reinterprets the bytes of the sequence as an
// unsigned integer. the byte size of the sequence must be 1, 2, 4, or 8.
func SequenceAsBinaryUnsignedInteger(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	v := s.Bytes()
	switch len(v) {
	case 1:
		return vm.NewNumber(float64(v[0]))
	case 2:
		return vm.NewNumber(float64(binary.LittleEndian.Uint16(v)))
	case 4:
		return vm.NewNumber(float64(binary.LittleEndian.Uint32(v)))
	case 8:
		return vm.NewNumber(float64(binary.LittleEndian.Uint64(v)))
	}
	return vm.RaiseException("asBinaryUnsignedInteger receiver must be Sequence of 1, 2, 4, or 8 bytes")
}

// SequenceAsin is a Sequence method.
//
// asin sets each element of the receiver to its arcsine.
func SequenceAsin(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("asin", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Asin)
	return s
}

// SequenceAtan is a Sequence method.
//
// atan sets each element of the receiver to its arctangent.
func SequenceAtan(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("atan", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Atan)
	return s
}

// SequenceCeil is a Sequence method.
//
// ceil sets each element of the receiver to the smallest integer greater than
// its current value. No-op on integer sequences.
func SequenceCeil(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("ceil", true); err != nil {
		return vm.IoError(err)
	}
	if s.IsFP() {
		s.MapUnary(math.Ceil)
	}
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

// SequenceCosh is a Sequence method.
//
// cosh sets each element of the receiver to its hyperbolic cosine.
func SequenceCosh(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("cosh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Cosh)
	return s
}

// SequenceFloor is a Sequence method.
//
// floor sets each element of the receiver to the largest integer less than its
// current value. No-op on integer sequences.
func SequenceFloor(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("floor", true); err != nil {
		return vm.IoError(err)
	}
	if s.IsFP() {
		s.MapUnary(math.Floor)
	}
	return s
}

// SequenceLog is a Sequence method.
//
// log sets each element of the receiver to its natural logarithm.
func SequenceLog(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("log", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Log)
	return s
}

// SequenceLog10 is a Sequence method.
//
// log10 sets each element of the receiver to its common logarithm.
func SequenceLog10(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("log10", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Log10)
	return s
}

// SequenceNegate is a Sequence method.
//
// negate sets each element of the receiver to its opposite.
func SequenceNegate(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("negate", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(func(x float64) float64 { return -x })
	return s
}

// SequenceSin is a Sequence method.
//
// sin sets each element of the receiver to its sine.
func SequenceSin(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sin", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sin)
	return s
}

// SequenceSinh is a Sequence method.
//
// sinh sets each element of the receiver to its hyperbolic sine.
func SequenceSinh(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sinh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sinh)
	return s
}

// SequenceSqrt is a Sequence method.
//
// sqrt sets each element of the receiver to its square root.
func SequenceSqrt(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sqrt", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sqrt)
	return s
}

// SequenceSquare is a Sequence method.
//
// square sets each element of the receiver to its square.
func SequenceSquare(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("square", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(func(x float64) float64 { return x * x })
	return s
}

// SequenceTan is a Sequence method.
//
// tan sets each element of the receiver to its tangent.
func SequenceTan(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("tan", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Tan)
	return s
}

// SequenceTanh is a Sequence method.
//
// tanh sets each element of the receiver to its hyperbolic tangent.
func SequenceTanh(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckNumeric("tanh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Tanh)
	return s
}
