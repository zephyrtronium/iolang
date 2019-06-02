package iolang

import (
	"fmt"
	"reflect"
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

var seqItemSizes = [...]int{0, 1, 2, 4, 8, 1, 2, 4, 8, 4, 8}

// SeqMaxItemSize is the maximum size in bytes of a single sequence element.
const SeqMaxItemSize = 8

// ItemSize returns the size in bytes of each element of the sequence.
func (kind SeqKind) ItemSize() int {
	if kind >= 0 {
		return seqItemSizes[kind]
	}
	return seqItemSizes[-kind]
}

// Encoding returns the suggested default encoding for the sequence kind. This
// is utf8 for uint8 kinds, utf16 for uint16, utf32 for int32, and number for
// all other kinds.
func (kind SeqKind) Encoding() string {
	switch kind {
	case SeqMU8, SeqIU8:
		return "utf8"
	case SeqMU16, SeqIU16:
		return "utf16"
	case SeqMS32, SeqIS32:
		return "utf32"
	}
	return "number"
}

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
func (s *Sequence) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
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

// IsFP returns whether the sequence has a float32 or float64 kind.
func (s *Sequence) IsFP() bool {
	switch s.Kind {
	case SeqMF32, SeqIF32, SeqMF64, SeqIF64:
		return true
	}
	return false
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

// FixIndex wraps an index into the sequence's size.
func (s *Sequence) FixIndex(i int) int {
	n := s.Len()
	if i >= n {
		return n
	}
	if i < 0 {
		i += n
		if i < 0 {
			return 0
		}
	}
	return i
}

// ItemSize is a proxy to s.Kind.ItemSize().
func (s *Sequence) ItemSize() int {
	return s.Kind.ItemSize()
}

// At returns the value of an item in the sequence as a float64. If the index
// is out of bounds, the second return value is false.
func (s *Sequence) At(i int) (float64, bool) {
	if i < 0 {
		return 0, false
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		if i >= len(v) {
			return 0, false
		}
		return v[i], true
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
			x, _ := s.At(i)
			v[i] = byte(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU16, SeqIU16:
		v := make([]uint16, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = uint16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU32, SeqIU32:
		v := make([]uint32, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = uint32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMU64, SeqIU64:
		v := make([]uint64, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = uint64(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS8, SeqIS8:
		v := make([]int8, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = int8(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS16, SeqIS16:
		v := make([]int16, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = int16(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS32, SeqIS32:
		v := make([]int32, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = int32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMS64, SeqIS64:
		v := make([]int64, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = int64(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMF32, SeqIF32:
		v := make([]float32, s.Len())
		for i := range v {
			x, _ := s.At(i)
			v[i] = float32(x)
		}
		return vm.NewSequence(v, kind > 0, s.Code)
	case SeqMF64, SeqIF64:
		v := make([]float64, s.Len())
		for i := range v {
			v[i], _ = s.At(i)
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
	} else if s.ItemSize() >= other.ItemSize() {
		s.appendConvert(other)
	} else {
		s.appendGrow(other)
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

func (s *Sequence) appendConvert(other *Sequence) {
	a := reflect.ValueOf(s.Value)
	b := reflect.ValueOf(other.Value)
	t := a.Type().Elem()
	for i := 0; i < b.Len(); i++ {
		a = reflect.Append(a, b.Index(i).Convert(t))
	}
	s.Value = a.Interface()
}

func (s *Sequence) appendGrow(other *Sequence) {
	old := reflect.ValueOf(s.Value)
	b := reflect.ValueOf(other.Value)
	t := b.Type().Elem()
	a := reflect.MakeSlice(b.Type(), 0, old.Len()+b.Len())
	for i := 0; i < old.Len(); i++ {
		a = reflect.Append(a, old.Index(i).Convert(t))
	}
	a = reflect.AppendSlice(a, b)
	s.Value = a.Interface()
	s.Kind = other.Kind
}

// Insert inserts the elements of another sequence, converted to this
// sequence's type, at a given index. If the index is beyond the length of the
// sequence, then zeros are inserted as needed. Panics if k < 0 or if s is
// immutable.
func (s *Sequence) Insert(other *Sequence, k int) {
	if err := s.CheckMutable("*Sequence.Insert"); err != nil {
		panic(err)
	}
	if sl := s.Len(); k > sl {
		s.extend(k)
	}
	if s.SameType(other) {
		s.insertSameKind(other, k)
	} else {
		s.insertConvert(other, k)
	}
}

func (s *Sequence) extend(k int) {
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		if len(v) < k {
			v = append(v, make([]byte, k-len(v))...)
		}
		s.Value = v
	case SeqMU16:
		v := s.Value.([]uint16)
		if len(v) < k {
			v = append(v, make([]uint16, k-len(v))...)
		}
		s.Value = v
	case SeqMU32:
		v := s.Value.([]uint32)
		if len(v) < k {
			v = append(v, make([]uint32, k-len(v))...)
		}
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		if len(v) < k {
			v = append(v, make([]uint64, k-len(v))...)
		}
		s.Value = v
	case SeqMS8:
		v := s.Value.([]int8)
		if len(v) < k {
			v = append(v, make([]int8, k-len(v))...)
		}
		s.Value = v
	case SeqMS16:
		v := s.Value.([]int16)
		if len(v) < k {
			v = append(v, make([]int16, k-len(v))...)
		}
		s.Value = v
	case SeqMS32:
		v := s.Value.([]int32)
		if len(v) < k {
			v = append(v, make([]int32, k-len(v))...)
		}
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		if len(v) < k {
			v = append(v, make([]int64, k-len(v))...)
		}
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		if len(v) < k {
			v = append(v, make([]float32, k-len(v))...)
		}
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		if len(v) < k {
			v = append(v, make([]float64, k-len(v))...)
		}
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) insertSameKind(other *Sequence, k int) {
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		w := other.Value.([]byte)
		v = append(v, make([]byte, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMU16:
		v := s.Value.([]uint16)
		w := other.Value.([]uint16)
		v = append(v, make([]uint16, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMU32:
		v := s.Value.([]uint32)
		w := other.Value.([]uint32)
		v = append(v, make([]uint32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMU64:
		v := s.Value.([]uint64)
		w := other.Value.([]uint64)
		v = append(v, make([]uint64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMS8:
		v := s.Value.([]int8)
		w := other.Value.([]int8)
		v = append(v, make([]int8, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMS16:
		v := s.Value.([]int16)
		w := other.Value.([]int16)
		v = append(v, make([]int16, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMS32:
		v := s.Value.([]int32)
		w := other.Value.([]int32)
		v = append(v, make([]int32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMS64:
		v := s.Value.([]int64)
		w := other.Value.([]int64)
		v = append(v, make([]int64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMF32:
		v := s.Value.([]float32)
		w := other.Value.([]float32)
		v = append(v, make([]float32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqMF64:
		v := s.Value.([]float64)
		w := other.Value.([]float64)
		v = append(v, make([]float64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) insertConvert(other *Sequence, k int) {
	a := reflect.ValueOf(s.Value)
	b := reflect.ValueOf(other.Value)
	al := a.Len()
	bl := b.Len()
	z := reflect.MakeSlice(a.Type(), bl, bl)
	a = reflect.AppendSlice(a, z)
	reflect.Copy(a.Slice(k+bl, al), a.Slice(k, al))
	at := a.Type().Elem()
	for i := 0; i < bl; i++ {
		a.Index(k + i).Set(b.Index(i).Convert(at))
	}
	s.Value = a.Interface()
}

// Find locates the first instance of other in the sequence following start.
// Comparison is done following conversion to the same type. If there is no
// match, the result is -1.
func (s *Sequence) Find(other *Sequence, start int) int {
	ol := other.Len()
	if ol == 0 {
		return start
	}
	checks := s.Len() - ol + 1
	for i := start; i < checks; i++ {
		if s.findMatch(other, i, ol) {
			return i
		}
	}
	return -1
}

// RFind locates the last instance of other in the sequence ending before end.
// Comparison is done following conversion to the same type. If there is no
// match, the result is -1.
func (s *Sequence) RFind(other *Sequence, end int) int {
	ol := other.Len()
	if ol == 0 {
		return end
	}
	if end > s.Len() {
		end = s.Len()
	}
	for i := end - ol; i >= 0; i-- {
		if s.findMatch(other, i, ol) {
			return i
		}
	}
	return -1
}

func (s *Sequence) findMatch(other *Sequence, i, ol int) bool {
	// TODO: this method is slow and imprecise for 64-bit types.
	for k := 0; k < ol; k++ {
		x, _ := s.At(i + k)
		y, _ := other.At(k)
		if x != y {
			return false
		}
	}
	return true
}

// Slice reduces the sequence to a selected linear portion.
func (s *Sequence) Slice(start, stop, step int) {
	if !s.IsMutable() {
		panic("cannot slice immutable sequence")
	}
	l := s.Len()
	start = fixSliceIndex(start, step, l)
	stop = fixSliceIndex(stop, step, l)
	if step > 0 {
		s.sliceForward(start, stop, step)
	} else if step < 0 {
		s.sliceBackward(start, stop, step)
	} else {
		panic("cannot slice with zero step")
	}
}

func (s *Sequence) sliceForward(start, stop, step int) {
	j := 0
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMU16:
		v := s.Value.([]uint16)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMU32:
		v := s.Value.([]uint32)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMU64:
		v := s.Value.([]uint64)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMS8:
		v := s.Value.([]int8)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMS16:
		v := s.Value.([]int16)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMS32:
		v := s.Value.([]int32)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMS64:
		v := s.Value.([]int64)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMF32:
		v := s.Value.([]float32)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqMF64:
		v := s.Value.([]float64)
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

func (s *Sequence) sliceBackward(start, stop, step int) {
	i, j := start, 0
	switch s.Kind {
	case SeqMU8:
		v := s.Value.([]byte)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMU16:
		v := s.Value.([]uint16)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMU32:
		v := s.Value.([]uint32)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMU64:
		v := s.Value.([]uint64)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMS8:
		v := s.Value.([]int8)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMS16:
		v := s.Value.([]int16)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMS32:
		v := s.Value.([]int32)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMS64:
		v := s.Value.([]int64)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMF32:
		v := s.Value.([]float32)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqMF64:
		v := s.Value.([]float64)
		for i > j && i > stop {
			v[j], v[i] = v[i], v[j]
			i += step
			j++
		}
		for i > stop {
			v[j] = v[start+i*step]
			i += step
			j++
		}
		s.Value = v[:j]
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
}

// Remove deletes a range of elements from the sequence. Panics if the sequence
// is immutable.
func (s *Sequence) Remove(i, j int) {
	if err := s.CheckMutable("*Sequence.Remove"); err != nil {
		panic(err)
	}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
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
		"afterSeq":         vm.NewTypedCFunction(SequenceAfterSeq, exemplar),
		"asList":           vm.NewTypedCFunction(SequenceAsList, exemplar),
		"asStruct":         vm.NewTypedCFunction(SequenceAsStruct, exemplar),
		"asSymbol":         vm.NewTypedCFunction(SequenceAsSymbol, exemplar),
		"at":               vm.NewTypedCFunction(SequenceAt, exemplar),
		"beforeSeq":        vm.NewTypedCFunction(SequenceBeforeSeq, exemplar),
		"beginsWithSeq":    vm.NewTypedCFunction(SequenceBeginsWithSeq, exemplar),
		"between":          vm.NewTypedCFunction(SequenceBetween, exemplar),
		"bitAt":            vm.NewTypedCFunction(SequenceBitAt, exemplar),
		"byteAt":           vm.NewTypedCFunction(SequenceByteAt, exemplar),
		"cloneAppendSeq":   vm.NewTypedCFunction(SequenceCloneAppendSeq, exemplar),
		"compare":          vm.NewTypedCFunction(SequenceCompare, exemplar),
		"contains":         vm.NewTypedCFunction(SequenceContains, exemplar),
		"containsSeq":      vm.NewTypedCFunction(SequenceContainsSeq, exemplar),
		"endsWithSeq":      vm.NewTypedCFunction(SequenceEndsWithSeq, exemplar),
		"exSlice":          vm.NewTypedCFunction(SequenceExSlice, exemplar),
		"findSeq":          vm.NewTypedCFunction(SequenceFindSeq, exemplar),
		"findSeqs":         vm.NewTypedCFunction(SequenceFindSeqs, exemplar),
		"foreach":          vm.NewTypedCFunction(SequenceForeach, exemplar),
		"hash":             vm.NewTypedCFunction(SequenceHash, exemplar),
		"inSlice":          vm.NewTypedCFunction(SequenceInSlice, exemplar),
		"isMutable":        vm.NewTypedCFunction(SequenceIsMutable, exemplar),
		"isZero":           vm.NewTypedCFunction(SequenceIsZero, exemplar),
		"itemSize":         vm.NewTypedCFunction(SequenceItemSize, exemplar),
		"itemType":         vm.NewTypedCFunction(SequenceItemType, exemplar),
		"occurrencesOfSeq": vm.NewTypedCFunction(SequenceOccurrencesOfSeq, exemplar),
		"pack":             vm.NewCFunction(SequencePack),
		"reverseFindSeq":   vm.NewTypedCFunction(SequenceReverseFindSeq, exemplar),
		"size":             vm.NewTypedCFunction(SequenceSize, exemplar),
		"splitAt":          vm.NewTypedCFunction(SequenceSplitAt, exemplar),
		"unpack":           vm.NewTypedCFunction(SequenceUnpack, exemplar),
		"withStruct":       vm.NewCFunction(SequenceWithStruct),

		// sequence-mutable.go:
		"append":              vm.NewTypedCFunction(SequenceAppend, exemplar),
		"appendSeq":           vm.NewTypedCFunction(SequenceAppendSeq, exemplar),
		"asMutable":           vm.NewTypedCFunction(SequenceAsMutable, exemplar),
		"atInsertSeq":         vm.NewTypedCFunction(SequenceAtInsertSeq, exemplar),
		"atPut":               vm.NewTypedCFunction(SequenceAtPut, exemplar),
		"clipAfterSeq":        vm.NewTypedCFunction(SequenceClipAfterSeq, exemplar),
		"clipAfterStartOfSeq": vm.NewTypedCFunction(SequenceClipAfterStartOfSeq, exemplar),
		"clipBeforeEndOfSeq":  vm.NewTypedCFunction(SequenceClipBeforeEndOfSeq, exemplar),
		"clipBeforeSeq":       vm.NewTypedCFunction(SequenceClipBeforeSeq, exemplar),
		"convertToItemType":   vm.NewTypedCFunction(SequenceConvertToItemType, exemplar),
		"copy":                vm.NewTypedCFunction(SequenceCopy, exemplar),
		"duplicateIndexes":    vm.NewTypedCFunction(SequenceDuplicateIndexes, exemplar),
		"empty":               vm.NewTypedCFunction(SequenceEmpty, exemplar),
		"insertSeqEvery":      vm.NewTypedCFunction(SequenceInsertSeqEvery, exemplar),
		"leaveThenRemove":     vm.NewTypedCFunction(SequenceLeaveThenRemove, exemplar),
		"preallocateToSize":   vm.NewTypedCFunction(SequencePreallocateToSize, exemplar),
		"rangeFill":           vm.NewTypedCFunction(SequenceRangeFill, exemplar),
		"removeAt":            vm.NewTypedCFunction(SequenceRemoveAt, exemplar),
		"removeEvenIndexes":   vm.NewTypedCFunction(SequenceRemoveEvenIndexes, exemplar),
		"removeLast":          vm.NewTypedCFunction(SequenceRemoveLast, exemplar),
		"removeOddIndexes":    vm.NewTypedCFunction(SequenceRemoveOddIndexes, exemplar),
		"removePrefix":        vm.NewTypedCFunction(SequenceRemovePrefix, exemplar),
		"removeSeq":           vm.NewTypedCFunction(SequenceRemoveSeq, exemplar),
		"removeSlice":         vm.NewTypedCFunction(SequenceRemoveSlice, exemplar),
		"removeSuffix":        vm.NewTypedCFunction(SequenceRemoveSuffix, exemplar),
		"replaceFirstSeq":     vm.NewTypedCFunction(SequenceReplaceFirstSeq, exemplar),
		"replaceSeq":          vm.NewTypedCFunction(SequenceReplaceSeq, exemplar),
		"reverseInPlace":      vm.NewTypedCFunction(SequenceReverseInPlace, exemplar),
		"setItemType":         vm.NewTypedCFunction(SequenceSetItemType, exemplar),
		"setItemsToDouble":    vm.NewTypedCFunction(SequenceSetItemsToDouble, exemplar),
		"setSize":             vm.NewTypedCFunction(SequenceSetSize, exemplar),
		"sort":                vm.NewTypedCFunction(SequenceSort, exemplar),

		// sequence-string.go:
		"appendPathSeq":          vm.NewTypedCFunction(SequenceAppendPathSeq, exemplar),
		"asBase64":               vm.NewTypedCFunction(SequenceAsBase64, exemplar),
		"asFixedSizeType":        vm.NewTypedCFunction(SequenceAsFixedSizeType, exemplar),
		"asIoPath":               vm.NewTypedCFunction(SequenceAsIoPath, exemplar),
		"asLatin1":               vm.NewTypedCFunction(SequenceAsLatin1, exemplar),
		"asMessage":              vm.NewTypedCFunction(SequenceAsMessage, exemplar),
		"asNumber":               vm.NewTypedCFunction(SequenceAsNumber, exemplar),
		"asOSPath":               vm.NewTypedCFunction(SequenceAsOSPath, exemplar),
		"asUTF16":                vm.NewTypedCFunction(SequenceAsUTF16, exemplar),
		"asUTF32":                vm.NewTypedCFunction(SequenceAsUTF32, exemplar),
		"asUTF8":                 vm.NewTypedCFunction(SequenceAsUTF8, exemplar),
		"capitalize":             vm.NewTypedCFunction(SequenceCapitalize, exemplar),
		"cloneAppendPath":        vm.NewTypedCFunction(SequenceCloneAppendPath, exemplar),
		"convertToFixedSizeType": vm.NewTypedCFunction(SequenceConvertToFixedSizeType, exemplar),
		"encoding":               vm.NewTypedCFunction(SequenceEncoding, exemplar),
		"escape":                 vm.NewTypedCFunction(SequenceEscape, exemplar),
		"fromBase":               vm.NewTypedCFunction(SequenceFromBase, exemplar),
		"fromBase64":             vm.NewTypedCFunction(SequenceFromBase64, exemplar),
		"interpolate":            vm.NewTypedCFunction(SequenceInterpolate, exemplar),
		"isLowercase":            vm.NewTypedCFunction(SequenceIsLowercase, exemplar),
		"isUppercase":            vm.NewTypedCFunction(SequenceIsUppercase, exemplar),
		"lastPathComponent":      vm.NewTypedCFunction(SequenceLastPathComponent, exemplar),
		"lowercase":              vm.NewTypedCFunction(SequenceLowercase, exemplar),
		"lstrip":                 vm.NewTypedCFunction(SequenceLstrip, exemplar),
		"setEncoding":            vm.NewTypedCFunction(SequenceSetEncoding, exemplar),
		"parseJson":              vm.NewTypedCFunction(SequenceParseJson, exemplar),
		"pathComponent":          vm.NewTypedCFunction(SequencePathComponent, exemplar),
		"pathExtension":          vm.NewTypedCFunction(SequencePathExtension, exemplar),
		"percentDecoded":         vm.NewTypedCFunction(SequencePercentDecoded, exemplar),
		"percentEncoded":         vm.NewTypedCFunction(SequencePercentEncoded, exemplar),
		"rstrip":                 vm.NewTypedCFunction(SequenceRstrip, exemplar),
		"split":                  vm.NewTypedCFunction(SequenceSplit, exemplar),
		"strip":                  vm.NewTypedCFunction(SequenceStrip, exemplar),
		"toBase":                 vm.NewTypedCFunction(SequenceToBase, exemplar),
		"unescape":               vm.NewTypedCFunction(SequenceUnescape, exemplar),
		"uppercase":              vm.NewTypedCFunction(SequenceUppercase, exemplar),
		"urlDecoded":             vm.NewTypedCFunction(SequenceUrlDecoded, exemplar),
		"urlEncoded":             vm.NewTypedCFunction(SequenceUrlEncoded, exemplar),
		"validEncodings":         vm.NewCFunction(SequenceValidEncodings),

		// sequence-math.go:
		"**=":                     vm.NewTypedCFunction(SequenceStarStarEq, exemplar),
		"*=":                      vm.NewTypedCFunction(SequenceStarEq, exemplar),
		"+=":                      vm.NewTypedCFunction(SequencePlusEq, exemplar),
		"-=":                      vm.NewTypedCFunction(SequenceMinusEq, exemplar),
		"/=":                      vm.NewTypedCFunction(SequenceSlashEq, exemplar),
		"Max":                     vm.NewTypedCFunction(SequencePairwiseMax, exemplar),
		"Min":                     vm.NewTypedCFunction(SequencePairwiseMin, exemplar),
		"abs":                     vm.NewTypedCFunction(SequenceAbs, exemplar),
		"acos":                    vm.NewTypedCFunction(SequenceAcos, exemplar),
		"asBinaryNumber":          vm.NewTypedCFunction(SequenceAsBinaryNumber, exemplar),
		"asBinarySignedInteger":   vm.NewTypedCFunction(SequenceAsBinarySignedInteger, exemplar),
		"asBinaryUnsignedInteger": vm.NewTypedCFunction(SequenceAsBinaryUnsignedInteger, exemplar),
		"asin":                    vm.NewTypedCFunction(SequenceAsin, exemplar),
		"atan":                    vm.NewTypedCFunction(SequenceAtan, exemplar),
		"bitCount":                vm.NewTypedCFunction(SequenceBitCount, exemplar),
		"bitwiseAnd":              vm.NewTypedCFunction(SequenceBitwiseAnd, exemplar),
		"bitwiseNot":              vm.NewTypedCFunction(SequenceBitwiseNot, exemplar),
		"bitwiseOr":               vm.NewTypedCFunction(SequenceBitwiseOr, exemplar),
		"bitwiseXor":              vm.NewTypedCFunction(SequenceBitwiseXor, exemplar),
		"ceil":                    vm.NewTypedCFunction(SequenceCeil, exemplar),
		"cos":                     vm.NewTypedCFunction(SequenceCos, exemplar),
		"cosh":                    vm.NewTypedCFunction(SequenceCosh, exemplar),
		"distanceTo":              vm.NewTypedCFunction(SequenceDistanceTo, exemplar),
		"dotProduct":              vm.NewTypedCFunction(SequenceDotProduct, exemplar),
		"floor":                   vm.NewTypedCFunction(SequenceFloor, exemplar),
		"log":                     vm.NewTypedCFunction(SequenceLog, exemplar),
		"log10":                   vm.NewTypedCFunction(SequenceLog10, exemplar),
		"max":                     vm.NewTypedCFunction(SequenceMax, exemplar),
		"mean":                    vm.NewTypedCFunction(SequenceMean, exemplar),
		"meanSquare":              vm.NewTypedCFunction(SequenceMeanSquare, exemplar),
		"min":                     vm.NewTypedCFunction(SequenceMin, exemplar),
		"negate":                  vm.NewTypedCFunction(SequenceNegate, exemplar),
		"normalize":               vm.NewTypedCFunction(SequenceNormalize, exemplar),
		"product":                 vm.NewTypedCFunction(SequenceProduct, exemplar),
		"sin":                     vm.NewTypedCFunction(SequenceSin, exemplar),
		"sinh":                    vm.NewTypedCFunction(SequenceSinh, exemplar),
		"sqrt":                    vm.NewTypedCFunction(SequenceSqrt, exemplar),
		"sum":                     vm.NewTypedCFunction(SequenceSum, exemplar),
		"square":                  vm.NewTypedCFunction(SequenceSquare, exemplar),
		"tan":                     vm.NewTypedCFunction(SequenceTan, exemplar),
		"tanh":                    vm.NewTypedCFunction(SequenceTanh, exemplar),
	}
	slots["addEquals"] = slots["+="]
	slots["asBuffer"] = slots["asMutable"]
	slots["asString"] = slots["asSymbol"]
	slots["betweenSeq"] = slots["between"]
	slots["exclusiveSlice"] = slots["exSlice"]
	slots["inclusiveSlice"] = slots["inSlice"]
	slots["slice"] = slots["exSlice"]
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
