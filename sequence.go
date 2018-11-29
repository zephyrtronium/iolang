package iolang

import (
	"fmt"
)

// There are a *lot* of Sequence methods, and each one needs to be able to
// handle a dozen different data types, so the implementation is spread out
// among a few separate files:
//
//    - sequence.go: The Sequence type itself and Go methods thereof.
//    - sequence-immutable.go: Io methods for non-mutating Sequence methods.
//    - sequence-mutable.go: Io methods for mutating Sequence methods.
//    - sequence-string.go: Implementation of Sequence as a string type,
//        including encodings and representation.
//    - sequence-math.go: Mathematical methods and operations. Eventually,
//        this should have different versions for different arches.

// A Sequence is a collection of data of one fixed-size type.
type Sequence struct {
	Object
	Value interface{}
	Kind  SeqKind
	Code  string // encoding
}

// SeqKind encodes the data type and mutability of a Sequence.
type SeqKind int8

// Sequence data types. M means mutable, I means immutable; U means unsigned,
// S means signed, F means floating point; and the number is the size in bits
// of each datum.
const (
	SeqUntyped SeqKind = 0

	SeqMU8, SeqIU8 SeqKind = iota, -iota
	SeqMU16, SeqIU16
	SeqMU32, SeqIU32
	SeqMU64, SeqIU64
	SeqMS8, SeqIS8
	SeqMS16, SeqIS16
	SeqMS32, SeqIS32
	SeqMS64, SeqIS64
	SeqMF32, SeqIF32
	SeqMF64, SeqIF64
)

// NewSequence creates a new Sequence with the given value and with the given
// encoding. The value must be a slice of a basic fixed-size data type, and it
// is copied. Panics if the encoding is not supported.
func (vm *VM) NewSequence(value interface{}, mutable bool, encoding string) *Sequence {
	if !vm.CheckEncoding(encoding) {
		panic(fmt.Sprintf("unsupported encoding %q", encoding))
	}
	kind := SeqUntyped
	switch v := value.(type) {
	case []byte:
		kind = SeqMU8
		value = append([]byte{}, v...)
	case []uint16:
		kind = SeqMU16
		value = append([]uint16{}, v...)
	case []uint32:
		kind = SeqMU32
		value = append([]uint32{}, v...)
	case []uint64:
		kind = SeqMU64
		value = append([]uint64{}, v...)
	case []int8:
		kind = SeqMS8
		value = append([]int8{}, v...)
	case []int16:
		kind = SeqMS16
		value = append([]int16{}, v...)
	case []int32:
		kind = SeqMS32
		value = append([]int32{}, v...)
	case []int64:
		kind = SeqMS64
		value = append([]int64{}, v...)
	case []float32:
		kind = SeqMF32
		value = append([]float32{}, v...)
	case []float64:
		kind = SeqMF64
		value = append([]float64{}, v...)
	default:
		panic(fmt.Sprintf("unsupported value type %T, must be slice of basic fixed-size data type", value))
	}
	if mutable {
		return &Sequence{
			Object: *vm.CoreInstance("Sequence"),
			Value:  value,
			Kind:   kind,
			Code:   encoding,
		}
	}
	return &Sequence{
		Object: *vm.CoreInstance("ImmutableSequence"),
		Value:  value,
		Kind:   -kind,
		Code:   encoding,
	}
}

// Activate returns the sequence.
func (s *Sequence) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	return s
}

// Clone returns a new Sequence whose value is a copy of this one's.
func (s *Sequence) Clone() Interface {
	ns := Sequence{
		Object: Object{Slots: Slots{}, Protos: []Interface{s}},
		Kind:   s.Kind,
		Code:   s.Code,
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		ns.Value = append([]byte{}, s.Value.([]byte)...)
	case SeqMU16, SeqIU16:
		ns.Value = append([]uint16{}, s.Value.([]uint16)...)
	case SeqMU32, SeqIU32:
		ns.Value = append([]uint32{}, s.Value.([]uint32)...)
	case SeqMU64, SeqIU64:
		ns.Value = append([]uint64{}, s.Value.([]uint64)...)
	case SeqMS8, SeqIS8:
		ns.Value = append([]int8{}, s.Value.([]int8)...)
	case SeqMS16, SeqIS16:
		ns.Value = append([]int16{}, s.Value.([]int16)...)
	case SeqMS32, SeqIS32:
		ns.Value = append([]int32{}, s.Value.([]int32)...)
	case SeqMS64, SeqIS64:
		ns.Value = append([]int64{}, s.Value.([]int64)...)
	case SeqMF32, SeqIF32:
		ns.Value = append([]float32{}, s.Value.([]float32)...)
	case SeqMF64, SeqIF64:
		ns.Value = append([]float64{}, s.Value.([]float64)...)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return &ns
}

// IsMutable returns whether the sequence can be modified safely.
func (s *Sequence) IsMutable() bool {
	return s.Kind > 0
}

// SameType returns whether the sequence has the same data type as another,
// regardless of mutability.
func (s *Sequence) SameType(as *Sequence) bool {
	return s.Kind == as.Kind || s.Kind == -as.Kind
}

// Len returns the length of the sequence.
func (s *Sequence) Len() int {
	switch s.Kind {
	case SeqMU8, SeqIU8:
		return len(s.Value.([]byte))
	case SeqMU16, SeqIU16:
		return len(s.Value.([]uint16))
	case SeqMU32, SeqIU32:
		return len(s.Value.([]uint32))
	case SeqMU64, SeqIU64:
		return len(s.Value.([]uint64))
	case SeqMS8, SeqIS8:
		return len(s.Value.([]int8))
	case SeqMS16, SeqIS16:
		return len(s.Value.([]int16))
	case SeqMS32, SeqIS32:
		return len(s.Value.([]int32))
	case SeqMS64, SeqIS64:
		return len(s.Value.([]int64))
	case SeqMF32, SeqIF32:
		return len(s.Value.([]float32))
	case SeqMF64, SeqIF64:
		return len(s.Value.([]float64))
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

var seqItemSizes = [...]int{0, 1, 2, 4, 8, 1, 2, 4, 8, 4, 8}

// ItemSize returns the size in bytes of each element of the sequence.
func (s *Sequence) ItemSize() int {
	if s.Kind >= 0 {
		return seqItemSizes[s.Kind]
	}
	return seqItemSizes[-s.Kind]
}

// At returns the value of an item in the sequence as a Number. If the index is
// out of bounds, the result will be nil.
func (s *Sequence) At(vm *VM, i int) *Number {
	if i < 0 {
		return nil
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(float64(v[i]))
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		if i >= len(v) {
			return nil
		}
		return vm.NewNumber(v[i])
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// Convert changes the item type of the sequence. The conversion is such that
// the result keeps the same number of items.
func (s *Sequence) Convert(vm *VM, kind SeqKind) *Sequence {
	if kind == s.Kind || kind == -s.Kind {
		return vm.NewSequence(s.Value, kind > 0, s.Code)
	}
	if kind == 0 {
		panic("conversion to untyped sequence")
	}
	if kind != SeqMF32 && kind != SeqIF32 && kind != SeqMF64 && kind != SeqIF64 {
		if s.Kind == SeqMU64 || s.Kind == SeqIU64 {
			return s.convertU64(vm, kind)
		}
		if s.Kind == SeqMS64 || s.Kind == SeqIS64 {
			return s.convertS64(vm, kind)
		}
	}
	switch kind {
	case SeqMU8, SeqIU8:
		v := make([]byte, s.Len())
		for i := range v {
			v[i] = byte(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU16, SeqIU16:
		v := make([]uint16, s.Len())
		for i := range v {
			v[i] = uint16(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU32, SeqIU32:
		v := make([]uint32, s.Len())
		for i := range v {
			v[i] = uint32(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU64, SeqIU64:
		v := make([]uint64, s.Len())
		for i := range v {
			v[i] = uint64(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS8, SeqIS8:
		v := make([]int8, s.Len())
		for i := range v {
			v[i] = int8(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS16, SeqIS16:
		v := make([]int16, s.Len())
		for i := range v {
			v[i] = int16(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS32, SeqIS32:
		v := make([]int32, s.Len())
		for i := range v {
			v[i] = int32(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS64, SeqIS64:
		v := make([]int64, s.Len())
		for i := range v {
			v[i] = int64(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMF32, SeqIF32:
		v := make([]float32, s.Len())
		for i := range v {
			v[i] = float32(s.At(vm, i).Value)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMF64, SeqIF64:
		v := make([]float64, s.Len())
		for i := range v {
			v[i] = s.At(vm, i).Value
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	}
	panic(fmt.Sprintf("unknown sequence kind %#v", kind))
}

// convertU64 converts a uint64 sequence to an integer type without loss of
// precision.
func (s *Sequence) convertU64(vm *VM, kind SeqKind) *Sequence {
	sv := s.Value.([]uint64)
	switch kind {
	case SeqMU8, SeqIU8:
		v := make([]byte, len(sv))
		for i, x := range sv {
			v[i] = byte(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU16, SeqIU16:
		v := make([]uint16, len(sv))
		for i, x := range sv {
			v[i] = uint16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU32, SeqIU32:
		v := make([]uint32, len(sv))
		for i, x := range sv {
			v[i] = uint32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
		// U64 is handled by Convert's same-type case.
	case SeqMS8, SeqIS8:
		v := make([]int8, len(sv))
		for i, x := range sv {
			v[i] = int8(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS16, SeqIS16:
		v := make([]int16, len(sv))
		for i, x := range sv {
			v[i] = int16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS32, SeqIS32:
		v := make([]int32, len(sv))
		for i, x := range sv {
			v[i] = int32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS64, SeqIS64:
		v := make([]int64, len(sv))
		for i, x := range sv {
			v[i] = int64(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	}
	panic(fmt.Sprintf("unknown sequence kind %#v", kind))
}

// convertS64 converts an int64 sequence to an integer type without loss of
// precision.
func (s *Sequence) convertS64(vm *VM, kind SeqKind) *Sequence {
	sv := s.Value.([]int64)
	switch kind {
	case SeqMU8, SeqIU8:
		v := make([]byte, len(sv))
		for i, x := range sv {
			v[i] = byte(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU16, SeqIU16:
		v := make([]uint16, len(sv))
		for i, x := range sv {
			v[i] = uint16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU32, SeqIU32:
		v := make([]uint32, len(sv))
		for i, x := range sv {
			v[i] = uint32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU64, SeqIU64:
		v := make([]uint64, len(sv))
		for i, x := range sv {
			v[i] = uint64(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS8, SeqIS8:
		v := make([]int8, len(sv))
		for i, x := range sv {
			v[i] = int8(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS16, SeqIS16:
		v := make([]int16, len(sv))
		for i, x := range sv {
			v[i] = int16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS32, SeqIS32:
		v := make([]int32, len(sv))
		for i, x := range sv {
			v[i] = int32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
		// S64 is handled by Convert's same-type case.
	}
	panic(fmt.Sprintf("unknown sequence kind %#v", kind))
}

// Append appends other's items to this sequence. If other has a larger item
// size than this sequence, then this sequence will be converted to the item
// type of other. Panics if this sequence is not mutable.
func (s *Sequence) Append(other *Sequence) {
	if err := s.CheckMutable("*Sequence.Append"); err != nil {
		panic(err)
	}
	if s.Kind == other.Kind || s.Kind == -other.Kind {
		s.appendSameKind(other)
		return
	}
	// This implementation is about 800 lines and took me about twelve hours
	// to write, which makes me wonder whether there is not a better
	// solution within Go's type system. Even if we have to use reflection,
	// I don't think this method is viable.
	switch other.Kind {
	case SeqMU8, SeqIU8:
		s.appendU8(other)
	case SeqMU16, SeqIU16:
		s.appendU16(other)
	case SeqMU32, SeqIU32:
		s.appendU32(other)
	case SeqMU64, SeqIU64:
		s.appendU64(other)
	case SeqMS8, SeqIS8:
		s.appendS8(other)
	case SeqMS16, SeqIS16:
		s.appendS16(other)
	case SeqMS32, SeqIS32:
		s.appendS32(other)
	case SeqMS64, SeqIS64:
		s.appendS64(other)
	case SeqMF32, SeqIF32:
		s.appendF32(other)
	case SeqMF64, SeqIF64:
		s.appendF64(other)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", other.Kind))
	}
}

func (s *Sequence) appendSameKind(other *Sequence) {
	switch s.Kind {
	case SeqMU8:
		s.Value = append(s.Value.([]byte), other.Value.([]byte)...)
	case SeqMU16:
		s.Value = append(s.Value.([]uint16), other.Value.([]uint16)...)
	case SeqMU32:
		s.Value = append(s.Value.([]uint32), other.Value.([]uint32)...)
	case SeqMU64:
		s.Value = append(s.Value.([]uint64), other.Value.([]uint64)...)
	case SeqMS8:
		s.Value = append(s.Value.([]int8), other.Value.([]int8)...)
	case SeqMS16:
		s.Value = append(s.Value.([]int16), other.Value.([]int16)...)
	case SeqMS32:
		s.Value = append(s.Value.([]int32), other.Value.([]int32)...)
	case SeqMS64:
		s.Value = append(s.Value.([]int64), other.Value.([]int64)...)
	case SeqMF32:
		s.Value = append(s.Value.([]float32), other.Value.([]float32)...)
	case SeqMF64:
		s.Value = append(s.Value.([]float64), other.Value.([]float64)...)
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendU8(other *Sequence) {
	ov := other.Value.([]byte)
	switch s.Kind {
	case SeqMU16:
		v := s.Value.([]uint16)
		for _, x := range ov {
			v = append(v, uint16(x))
		}
		s.Value = v
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		v := s.Value.([]int8)
		for _, x := range ov {
			v = append(v, int8(x))
		}
		s.Value = v
	case SeqMS16:
		v := s.Value.([]int16)
		for _, x := range ov {
			v = append(v, int16(x))
		}
		s.Value = v
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendU16(other *Sequence) {
	ov := other.Value.([]uint16)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]uint16, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint16(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU16
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]uint16, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint16(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU16
	case SeqMS16:
		v := s.Value.([]int16)
		for _, x := range ov {
			v = append(v, int16(x))
		}
		s.Value = v
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendU32(other *Sequence) {
	ov := other.Value.([]uint32)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]uint32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU32
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]uint32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU32
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]uint32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU32
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]uint32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU32
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendU64(other *Sequence) {
	ov := other.Value.([]uint64)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMU32:
		old := s.Value.([]uint32)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMS32:
		old := s.Value.([]int32)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		old := s.Value.([]float32)
		v := make([]uint64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = uint64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMU64
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendS8(other *Sequence) {
	ov := other.Value.([]int8)
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		for _, x := range ov {
			v = append(v, byte(x))
		}
		s.Value = v
	case SeqMU16:
		v := s.Value.([]uint16)
		for _, x := range ov {
			v = append(v, uint16(x))
		}
		s.Value = v
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS16:
		v := s.Value.([]int16)
		for _, x := range ov {
			v = append(v, int16(x))
		}
		s.Value = v
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendS16(other *Sequence) {
	ov := other.Value.([]int16)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]int16, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int16(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS16
	case SeqMU16:
		v := s.Value.([]uint16)
		for _, x := range ov {
			v = append(v, uint16(x))
		}
		s.Value = v
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]int16, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int16(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS16
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendS32(other *Sequence) {
	ov := other.Value.([]int32)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]int32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS32
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]int32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS32
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]int32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS32
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]int32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS32
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		for _, x := range ov {
			v = append(v, float32(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendS64(other *Sequence) {
	ov := other.Value.([]int64)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMU32:
		old := s.Value.([]uint32)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMS32:
		old := s.Value.([]int32)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMF32:
		old := s.Value.([]float32)
		v := make([]int64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = int64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMS64
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendF32(other *Sequence) {
	ov := other.Value.([]float32)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]float32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF32
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]float32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF32
	case SeqMU32:
		v := s.Value.([]uint32)
		for _, x := range ov {
			v = append(v, uint32(x))
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]float32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF32
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]float32, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float32(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF32
	case SeqMS32:
		v := s.Value.([]int32)
		for _, x := range ov {
			v = append(v, int32(x))
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		for _, x := range ov {
			v = append(v, float64(x))
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) appendF64(other *Sequence) {
	ov := other.Value.([]float64)
	switch s.Kind {
	case SeqMU8:
		old := s.Value.([]byte)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMU16:
		old := s.Value.([]uint16)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMU32:
		old := s.Value.([]uint32)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMU64:
		v := s.Value.([]uint64)
		for _, x := range ov {
			v = append(v, uint64(x))
		}
		s.Value = v
	case SeqMS8:
		old := s.Value.([]int8)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMS16:
		old := s.Value.([]int16)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMS32:
		old := s.Value.([]int32)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqMS64:
		v := s.Value.([]int64)
		for _, x := range ov {
			v = append(v, int64(x))
		}
		s.Value = v
	case SeqMF32:
		old := s.Value.([]float32)
		v := make([]float64, len(old), len(old)+len(ov))
		for i, x := range old {
			v[i] = float64(x)
		}
		v = append(v, ov...)
		s.Value = v
		s.Kind = SeqMF64
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (vm *VM) initSequence() {
	var exemplar *Sequence
	// We can't use vm.NewString until we create the proto after this.
	slots := Slots{
		// sequence-immutable.go:
		"at":        vm.NewTypedCFunction(SequenceAt, exemplar),
		"compare":   vm.NewTypedCFunction(SequenceCompare, exemplar),
		"isMutable": vm.NewTypedCFunction(SequenceIsMutable, exemplar),
		"itemSize":  vm.NewTypedCFunction(SequenceItemSize, exemplar),
		"itemType":  vm.NewTypedCFunction(SequenceItemType, exemplar),
		"size":      vm.NewTypedCFunction(SequenceSize, exemplar),

		// sequence-mutable.go:
		"append": vm.NewTypedCFunction(SequenceAppend, exemplar),
		"appendSeq": vm.NewTypedCFunction(SequenceAppendSeq, exemplar),
		"asMutable": vm.NewTypedCFunction(SequenceAsMutable, exemplar),

		// sequence-string.go:
		"encoding":       vm.NewTypedCFunction(SequenceEncoding, exemplar),
		"setEncoding":    vm.NewTypedCFunction(SequenceSetEncoding, exemplar),
		"validEncodings": vm.NewCFunction(SequenceValidEncodings),
		"asUTF8":         vm.NewTypedCFunction(SequenceAsUTF8, exemplar),
		"asUTF16":        vm.NewTypedCFunction(SequenceAsUTF16, exemplar),
		"asUTF32":        vm.NewTypedCFunction(SequenceAsUTF32, exemplar),

		// sequence-math.go:
		"cos": vm.NewTypedCFunction(SequenceCos, exemplar),
	}
	ms := &Sequence{
		Object: *vm.ObjectWith(slots),
		Value:  []byte(nil),
		Kind:   SeqMU8,
		Code:   "utf8",
	}
	is := ms.Clone().(*Sequence)
	is.Kind = SeqIU8
	SetSlot(vm.Core, "Sequence", ms)
	SetSlot(vm.Core, "ImmutableSequence", is)
	SetSlot(vm.Core, "String", is)
	// Now that we have the String proto, we can use vm.NewString.
	SetSlot(ms, "type", vm.NewString("Sequence"))
	SetSlot(is, "type", vm.NewString("ImmutableSequence"))
}

// SequenceArgAt is a synonym for StringArgAt with nicer spelling.
func (m *Message) SequenceArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface) {
	return m.StringArgAt(vm, locals, n)
}
