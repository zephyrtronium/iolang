package iolang

import (
	"fmt"
	"math"
)

// CheckNumeric checks that the sequence is numeric.
func (s *Sequence) CheckNumeric(name string) error {
	if s.Code == "number" {
		return nil
	}
	return fmt.Sprintf("%q not valid on non-number encodings", a)
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

// SequenceCos is a Sequence method.
//
// cos sets each element of the receiver to its cosine.
func SequenceCos(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("cos"); err != nil {
		return vm.IoError(err)
	}
	if err := s.CheckNumeric("cos"); err != nil {
		return vm.IoError(err)
	}
	s.MapUnary(math.Cos)
}
