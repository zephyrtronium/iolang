package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
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
	switch v := s.Value.(type) {
	case []byte:
		for i, c := range v {
			v[i] = byte(op(float64(c)))
		}
	case []uint16:
		for i, c := range v {
			v[i] = uint16(op(float64(c)))
		}
	case []uint32:
		for i, c := range v {
			v[i] = uint32(op(float64(c)))
		}
	case []uint64:
		for i, c := range v {
			v[i] = uint64(op(float64(c)))
		}
	case []int8:
		for i, c := range v {
			v[i] = int8(op(float64(c)))
		}
	case []int16:
		for i, c := range v {
			v[i] = int16(op(float64(c)))
		}
	case []int32:
		for i, c := range v {
			v[i] = int32(op(float64(c)))
		}
	case []int64:
		for i, c := range v {
			v[i] = int64(op(float64(c)))
		}
	case []float32:
		for i, c := range v {
			v[i] = float32(op(float64(c)))
		}
	case []float64:
		for i, c := range v {
			v[i] = op(c)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// MapBinary replaces each value of the sequence with the result of applying op
// with the respective value of t, or with the given default value if past the
// end of t. Values are converted to float64 and back to the appropriate type.
func (s *Sequence) MapBinary(op func(float64, float64) float64, t *Sequence, def float64) {
	if !s.IsMutable() {
		panic("can't modify immutable sequence")
	}
	switch v := s.Value.(type) {
	case []byte:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = byte(op(float64(c), x))
		}
	case []uint16:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint16(op(float64(c), x))
		}
	case []uint32:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint32(op(float64(c), x))
		}
	case []uint64:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = uint64(op(float64(c), x))
		}
	case []int8:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int8(op(float64(c), x))
		}
	case []int16:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int16(op(float64(c), x))
		}
	case []int32:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int32(op(float64(c), x))
		}
	case []int64:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = int64(op(float64(c), x))
		}
	case []float32:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = float32(op(float64(c), x))
		}
	case []float64:
		for i, c := range v {
			x, ok := t.At(i)
			if !ok {
				x = def
			}
			v[i] = op(c, x)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// Reduce evaluates op on each element of the sequence, using the output as the
// first input to the following call. The first input for the first element is
// ic.
func (s *Sequence) Reduce(op func(float64, float64) float64, ic float64) float64 {
	switch v := s.Value.(type) {
	case []byte:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []uint16:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []uint32:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []uint64:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []int8:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []int16:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []int32:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []int64:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []float32:
		for _, c := range v {
			ic = op(ic, float64(c))
		}
	case []float64:
		for _, c := range v {
			ic = op(ic, c)
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return ic
}

// SeqOrNumArgAt evaluates the given argument, then returns it as a Sequence
// or Number, or a raised exception if it is neither, or a return or raised
// exception if one occurs during evaluation.
func (m *Message) SeqOrNumArgAt(vm *VM, locals Interface, n int) (*Sequence, *Number, Interface, Stop) {
	r, stop := m.EvalArgAt(vm, locals, n)
	if stop != NoStop {
		return nil, nil, r, stop
	}
	switch v := r.(type) {
	case *Sequence:
		return v, nil, nil, NoStop
	case *Number:
		return nil, v, nil, NoStop
	}
	return nil, nil, vm.NewExceptionf("argument %d to %s must be Sequence or Number, not %s", n, m.Name(), vm.TypeName(r)), ExceptionStop
}

// SequenceStarStarEq is a Sequence method.
//
// **= sets each element of the receiver to its value raised to the power of the
// respective element of the argument.
func SequenceStarStarEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("**=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if t != nil {
		s.MapBinary(math.Pow, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return math.Pow(x, y) })
	}
	return target, NoStop
}

// SequenceStarEq is a Sequence method.
//
// *= sets each element of the receiver to its value times the respective
// element of the argument.
func SequenceStarEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("*=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x * y }, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x * y })
	}
	return target, NoStop
}

// SequencePlusEq is a Sequence method.
//
// += sets each element of the receiver to its value plus the respective
// element of the argument.
func SequencePlusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("+=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x + y }, t, 0)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x + y })
	}
	return target, NoStop
}

// SequenceMinusEq is a Sequence method.
//
// -= sets each element of the receiver to its value minus the respective
// element of the argument.
func SequenceMinusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("-=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x - y }, t, 0)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x - y })
	}
	return target, NoStop
}

// SequenceSlashEq is a Sequence method.
//
// /= sets each element of the receiver to its value divided by the respective
// element of the argument.
func SequenceSlashEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("/=", true); err != nil {
		return vm.IoError(err)
	}
	t, n, err, stop := msg.SeqOrNumArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if t != nil {
		s.MapBinary(func(x, y float64) float64 { return x / y }, t, 1)
	} else {
		y := n.Value
		s.MapUnary(func(x float64) float64 { return x / y })
	}
	return target, NoStop
}

// SequencePairwiseMax is a Sequence method.
//
// Max sets each element of the receiver to the greater of the receiver element
// and the respective argument element.
func SequencePairwiseMax(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("Max", true); err != nil {
		return vm.IoError(err)
	}
	t, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	s.MapBinary(math.Max, t, math.Inf(-1))
	return target, NoStop
}

// SequencePairwiseMin is a Sequence method.
//
// Min sets each element of the receiver to the lesser of the receiver element
// and the respective argument element.
func SequencePairwiseMin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("Min", true); err != nil {
		return vm.IoError(err)
	}
	t, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	s.MapBinary(math.Min, t, math.Inf(0))
	return target, NoStop
}

// SequenceAbs is a Sequence method.
//
// abs sets each element of the receiver to its absolute value.
func SequenceAbs(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("abs", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Abs)
	return s, NoStop
}

// SequenceAcos is a Sequence method.
//
// acos sets each element of the receiver to its arc-cosine.
func SequenceAcos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("acos", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Acos)
	return s, NoStop
}

// SequenceAsBinaryNumber is a Sequence method.
//
// asBinaryNumber reinterprets the first eight bytes of the sequence as an
// IEEE-754 binary64 floating-point value and returns the appropriate Number.
func SequenceAsBinaryNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	v := s.Bytes()
	if len(v) < 8 {
		return vm.RaiseExceptionf("need 8 bytes in sequence, have only %d", len(v))
	}
	x := binary.LittleEndian.Uint64(v)
	return vm.NewNumber(math.Float64frombits(x)), NoStop
}

// SequenceAsBinarySignedInteger is a Sequence method.
//
// asBinarySignedInteger reinterprets the bytes of the sequence as a signed
// integer. The byte size of the sequence must be 1, 2, 4, or 8.
func SequenceAsBinarySignedInteger(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	v := s.Bytes()
	switch len(v) {
	case 1:
		return vm.NewNumber(float64(int8(v[0]))), NoStop
	case 2:
		return vm.NewNumber(float64(int16(binary.LittleEndian.Uint16(v)))), NoStop
	case 4:
		return vm.NewNumber(float64(int32(binary.LittleEndian.Uint32(v)))), NoStop
	case 8:
		return vm.NewNumber(float64(int64(binary.LittleEndian.Uint64(v)))), NoStop
	}
	return vm.RaiseException("asBinarySignedInteger receiver must be Sequence of 1, 2, 4, or 8 bytes")
}

// SequenceAsBinaryUnsignedInteger is a Sequence method.
//
// asBinaryUnsignedInteger reinterprets the bytes of the sequence as an
// unsigned integer. the byte size of the sequence must be 1, 2, 4, or 8.
func SequenceAsBinaryUnsignedInteger(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	v := s.Bytes()
	switch len(v) {
	case 1:
		return vm.NewNumber(float64(v[0])), NoStop
	case 2:
		return vm.NewNumber(float64(binary.LittleEndian.Uint16(v))), NoStop
	case 4:
		return vm.NewNumber(float64(binary.LittleEndian.Uint32(v))), NoStop
	case 8:
		return vm.NewNumber(float64(binary.LittleEndian.Uint64(v))), NoStop
	}
	return vm.RaiseException("asBinaryUnsignedInteger receiver must be Sequence of 1, 2, 4, or 8 bytes")
}

// SequenceAsin is a Sequence method.
//
// asin sets each element of the receiver to its arcsine.
func SequenceAsin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("asin", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Asin)
	return s, NoStop
}

// SequenceAtan is a Sequence method.
//
// atan sets each element of the receiver to its arctangent.
func SequenceAtan(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("atan", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Atan)
	return s, NoStop
}

// SequenceBitCount is a Sequence method.
//
// bitCount returns the number of 1 bits in the sequence.
func SequenceBitCount(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	n := 0
	switch v := s.Value.(type) {
	case []byte:
		for _, c := range v {
			n += bits.OnesCount8(c)
		}
	case []uint16:
		for _, c := range v {
			n += bits.OnesCount16(c)
		}
	case []uint32:
		for _, c := range v {
			n += bits.OnesCount32(c)
		}
	case []uint64:
		for _, c := range v {
			n += bits.OnesCount64(c)
		}
	case []int8:
		for _, c := range v {
			n += bits.OnesCount8(byte(c))
		}
	case []int16:
		for _, c := range v {
			n += bits.OnesCount16(uint16(c))
		}
	case []int32:
		for _, c := range v {
			n += bits.OnesCount32(uint32(c))
		}
	case []int64:
		for _, c := range v {
			n += bits.OnesCount64(uint64(c))
		}
	case []float32:
		for _, c := range v {
			n += bits.OnesCount32(math.Float32bits(c))
		}
	case []float64:
		for _, c := range v {
			n += bits.OnesCount64(math.Float64bits(c))
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return vm.NewNumber(float64(n)), NoStop
}

// SequenceBitwiseAnd is a Sequence method.
//
// bitwiseAnd sets the receiver to the bitwise AND of its binary representation
// and that of the argument sequence.
func SequenceBitwiseAnd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("bitwiseAnd"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v := s.Bytes()
	w := other.BytesN(len(v))
	var i int
	for i = 0; i < len(w)/8; i++ {
		x := binary.LittleEndian.Uint64(v[8*i:])
		y := binary.LittleEndian.Uint64(w[8*i:])
		x &= y
		binary.LittleEndian.PutUint64(v[8*i:], x)
	}
	for i *= 8; i < len(w); i++ {
		v[i] &= w[i]
	}
	binary.Read(bytes.NewReader(v), binary.LittleEndian, s.Value)
	return target, NoStop
}

// SequenceBitwiseNot is a Sequence method.
//
// bitwiseNot sets the receiver to the bitwise NOT of its binary representation
// and that of the argument sequence.
func SequenceBitwiseNot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("bitwiseNot"); err != nil {
		return vm.IoError(err)
	}
	v := s.Bytes()
	var i int
	for i = 0; i < len(v)/8; i++ {
		x := binary.LittleEndian.Uint64(v[8*i:])
		binary.LittleEndian.PutUint64(v[8*i:], ^x)
	}
	for i *= 8; i < len(v); i++ {
		v[i] = ^v[i]
	}
	binary.Read(bytes.NewReader(v), binary.LittleEndian, s.Value)
	return target, NoStop
}

// SequenceBitwiseOr is a Sequence method.
//
// bitwiseOr sets the receiver to the bitwise OR of its binary representation
// and that of the argument sequence.
func SequenceBitwiseOr(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("bitwiseOr"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v := s.Bytes()
	w := other.BytesN(len(v))
	var i int
	for i = 0; i < len(w)/8; i++ {
		x := binary.LittleEndian.Uint64(v[8*i:])
		y := binary.LittleEndian.Uint64(w[8*i:])
		x |= y
		binary.LittleEndian.PutUint64(v[8*i:], x)
	}
	for i *= 8; i < len(w); i++ {
		v[i] |= w[i]
	}
	binary.Read(bytes.NewReader(v), binary.LittleEndian, s.Value)
	return target, NoStop
}

// SequenceBitwiseXor is a Sequence method.
//
// bitwiseXor sets the receiver to the bitwise XOR of its binary representation
// and that of the argument sequence.
func SequenceBitwiseXor(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckMutable("bitwiseXor"); err != nil {
		return vm.IoError(err)
	}
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	v := s.Bytes()
	w := other.BytesN(len(v))
	var i int
	for i = 0; i < len(w)/8; i++ {
		x := binary.LittleEndian.Uint64(v[8*i:])
		y := binary.LittleEndian.Uint64(w[8*i:])
		x ^= y
		binary.LittleEndian.PutUint64(v[8*i:], x)
	}
	for i *= 8; i < len(w); i++ {
		v[i] ^= w[i]
	}
	binary.Read(bytes.NewReader(v), binary.LittleEndian, s.Value)
	return target, NoStop
}

// SequenceCeil is a Sequence method.
//
// ceil sets each element of the receiver to the smallest integer greater than
// its current value. No-op on integer sequences.
func SequenceCeil(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("ceil", true); err != nil {
		return vm.IoError(err)
	}
	if s.IsFP() {
		s.MapUnary(math.Ceil)
	}
	return s, NoStop
}

// SequenceCos is a Sequence method.
//
// cos sets each element of the receiver to its cosine.
func SequenceCos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("cos", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Cos)
	return s, NoStop
}

// SequenceCosh is a Sequence method.
//
// cosh sets each element of the receiver to its hyperbolic cosine.
func SequenceCosh(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("cosh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Cosh)
	return s, NoStop
}

// SequenceDistanceTo is a Sequence method.
//
// distanceTo computes the L2-norm of the vector pointing between the receiver
// and the argument sequence. Both sequences must be of the same floating-point
// type and of equal size; otherwise, the result will be 0.
func SequenceDistanceTo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := target.(*Sequence)
	y, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if !x.SameType(y) {
		return vm.NewNumber(0), NoStop
	}
	switch v := x.Value.(type) {
	case []float32:
		w := y.Value.([]float32)
		if len(v) != len(w) {
			break
		}
		var sum float32
		for i, a := range v {
			b := a - w[i]
			sum += b * b
		}
		return vm.NewNumber(math.Sqrt(float64(sum))), NoStop
	case []float64:
		w := y.Value.([]float64)
		if len(v) != len(w) {
			break
		}
		var sum float64
		for i, a := range v {
			b := a - w[i]
			sum += b * b
		}
		return vm.NewNumber(math.Sqrt(sum)), NoStop
	}
	return vm.NewNumber(0), NoStop
}

// SequenceDotProduct is a Sequence method.
//
// dotProduct computes the sum of pairwise products between the receiver and
// argument sequence, up to the length of the shorter of the two.
func SequenceDotProduct(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	other, err, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	// The original required the receiver to be mutable for no reason, but we
	// don't. It /would/ be reasonable to require number encoding, but the
	// original doesn't, and I'm sufficiently comfortable with that.
	var sum float64
	i := 0
	for {
		x, ok := s.At(i)
		if !ok {
			break
		}
		y, ok := other.At(i)
		if !ok {
			break
		}
		sum += x * y
		i++
	}
	return vm.NewNumber(sum), NoStop
}

// SequenceFloor is a Sequence method.
//
// floor sets each element of the receiver to the largest integer less than its
// current value. No-op on integer sequences.
func SequenceFloor(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("floor", true); err != nil {
		return vm.IoError(err)
	}
	if s.IsFP() {
		s.MapUnary(math.Floor)
	}
	return s, NoStop
}

// SequenceLog is a Sequence method.
//
// log sets each element of the receiver to its natural logarithm.
func SequenceLog(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("log", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Log)
	return s, NoStop
}

// SequenceLog10 is a Sequence method.
//
// log10 sets each element of the receiver to its common logarithm.
func SequenceLog10(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("log10", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Log10)
	return s, NoStop
}

// SequenceMax is a Sequence method.
//
// max returns the maximum element in the sequence.
func SequenceMax(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(s.Reduce(math.Max, math.Inf(-1))), NoStop
}

// SequenceMean is a Sequence method.
//
// mean computes the arithmetic mean of the elements in the sequence.
func SequenceMean(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	r := s.Reduce(func(x, y float64) float64 { return x + y }, 0)
	return vm.NewNumber(r / float64(s.Len())), NoStop
}

// SequenceMeanSquare is a Sequence method.
//
// meanSquare computes the arithmetic mean of the squares of the elements in
// the sequence.
func SequenceMeanSquare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	// This disagrees with Io's meanSquare, which performs the squaring in the
	// receiver's type.
	r := s.Reduce(func(x, y float64) float64 { return x + y*y }, 0)
	return vm.NewNumber(r / float64(s.Len())), NoStop
}

// SequenceMin is a Sequence method.
//
// min returns the minimum element in the sequence.
func SequenceMin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(s.Reduce(math.Min, math.Inf(0))), NoStop
}

// SequenceNegate is a Sequence method.
//
// negate sets each element of the receiver to its opposite.
func SequenceNegate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("negate", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(func(x float64) float64 { return -x })
	return s, NoStop
}

// SequenceNormalize is a Sequence method.
//
// normalize divides each element of the receiver by the sequence's L2 norm.
func SequenceNormalize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	// The original only checks for mutability, not numeric.
	if err := s.CheckNumeric("normalize", true); err != nil {
		return vm.IoError(err)
	}
	l2 := math.Sqrt(s.Reduce(func(x, y float64) float64 { return x + y*y }, 0))
	s.MapUnary(func(x float64) float64 { return x / l2 })
	return target, NoStop
}

// SequenceProduct is a Sequence method.
//
// product returns the product of the elements of the sequence.
func SequenceProduct(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(s.Reduce(func(x, y float64) float64 { return x * y }, 1)), NoStop
}

// SequenceSin is a Sequence method.
//
// sin sets each element of the receiver to its sine.
func SequenceSin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sin", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sin)
	return s, NoStop
}

// SequenceSinh is a Sequence method.
//
// sinh sets each element of the receiver to its hyperbolic sine.
func SequenceSinh(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sinh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sinh)
	return s, NoStop
}

// SequenceSqrt is a Sequence method.
//
// sqrt sets each element of the receiver to its square root.
func SequenceSqrt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("sqrt", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Sqrt)
	return s, NoStop
}

// SequenceSquare is a Sequence method.
//
// square sets each element of the receiver to its square.
func SequenceSquare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("square", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(func(x float64) float64 { return x * x })
	return s, NoStop
}

// SequenceSum is a Sequence method.
//
// sum returns the sum of the elements of the sequence.
func SequenceSum(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	return vm.NewNumber(s.Reduce(func(x, y float64) float64 { return x + y }, 0)), NoStop
}

// SequenceTan is a Sequence method.
//
// tan sets each element of the receiver to its tangent.
func SequenceTan(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("tan", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Tan)
	return s, NoStop
}

// SequenceTanh is a Sequence method.
//
// tanh sets each element of the receiver to its hyperbolic tangent.
func SequenceTanh(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s := target.(*Sequence)
	if err := s.CheckNumeric("tanh", true); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Tanh)
	return s, NoStop
}
