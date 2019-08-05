package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
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

// SequenceFromBytes makes a Sequence with the given type having the same bit
// pattern as the given bytes. If the length of b is not a multiple of the item
// size for the given kind, the extra bytes are ignored. The sequence's
// encoding will be number unless the kind is uint8, uint16, or int32, in which
// cases the encoding will be utf8, utf16, or utf32, respectively.
func (vm *VM) SequenceFromBytes(b []byte, kind SeqKind) *Sequence {
	if kind == SeqMU8 || kind == SeqIU8 {
		return vm.NewSequence(b, kind > 0, "utf8")
	}
	if kind == SeqMF32 || kind == SeqIF32 {
		v := make([]float32, 0, len(b)/4)
		for len(b) >= 4 {
			c := binary.LittleEndian.Uint32(b)
			v = append(v, math.Float32frombits(c))
			b = b[4:]
		}
		return vm.NewSequence(v, kind > 0, "number")
	}
	if kind == SeqMF64 || kind == SeqIF64 {
		v := make([]float64, 0, len(b)/8)
		for len(b) >= 8 {
			c := binary.LittleEndian.Uint64(b)
			v = append(v, math.Float64frombits(c))
			b = b[8:]
		}
		return vm.NewSequence(v, kind > 0, "number")
	}
	var v interface{}
	switch kind {
	case SeqMU16, SeqIU16:
		v = make([]uint16, len(b)/2)
	case SeqMU32, SeqIU32:
		v = make([]uint32, len(b)/4)
	case SeqMU64, SeqIU64:
		v = make([]uint64, len(b)/8)
	case SeqMS8, SeqIS8:
		v = make([]int8, len(b))
	case SeqMS16, SeqIS16:
		v = make([]int16, len(b)/2)
	case SeqMS32, SeqIS32:
		v = make([]int32, len(b)/4)
	case SeqMS64, SeqIS64:
		v = make([]int64, len(b)/8)
	case SeqUntyped:
		panic("cannot create untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", kind))
	}
	binary.Read(bytes.NewReader(b), binary.LittleEndian, v)
	return vm.NewSequence(v, kind > 0, kind.Encoding())
}

// StringArgAt evaluates the nth argument and returns it as a Sequence. If a
// stop occurs during evaluation, the Sequence will be nil, and the stop status
// and result will be returned. If the evaluated result is not a Sequence, the
// result will be nil, and an exception will be raised.
func (m *Message) StringArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		if str, ok := v.(*Sequence); ok {
			return str, nil, NoStop
		}
		// Not the expected type, so return an error.
		v, s = vm.RaiseExceptionf("argument %d to %s must be Sequence, not %s", n, m.Text, vm.TypeName(v))
	}
	return nil, v, s
}

// SequenceArgAt is a synonym for StringArgAt with nicer spelling.
func (m *Message) SequenceArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface, Stop) {
	return m.StringArgAt(vm, locals, n)
}

// AsStringArgAt evaluates the nth argument, then activates its asString slot
// for a string representation. If the result is not a string, then the result
// is nil, and an error is returned.
func (m *Message) AsStringArgAt(vm *VM, locals Interface, n int) (*Sequence, Interface, Stop) {
	v, stop := m.EvalArgAt(vm, locals, n)
	if stop != NoStop {
		return nil, v, stop
	}
	r, stop := vm.Perform(v, locals, vm.IdentMessage("asString"))
	if stop == NoStop {
		if s, ok := r.(*Sequence); ok {
			return s, nil, NoStop
		}
		r, stop = vm.RaiseExceptionf("argument %d to %s cannot be converted to string", n, m.Text)
	}
	return nil, r, stop
}

// Activate returns the sequence.
func (s *Sequence) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return s, NoStop
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
	var kind *Sequence
	// We can't use vm.NewString until we create the proto after this.
	slots := Slots{
		// sequence-immutable.go:
		"afterSeq":         vm.NewCFunction(SequenceAfterSeq, kind),
		"asList":           vm.NewCFunction(SequenceAsList, kind),
		"asStruct":         vm.NewCFunction(SequenceAsStruct, kind),
		"asSymbol":         vm.NewCFunction(SequenceAsSymbol, kind),
		"at":               vm.NewCFunction(SequenceAt, kind),
		"beforeSeq":        vm.NewCFunction(SequenceBeforeSeq, kind),
		"beginsWithSeq":    vm.NewCFunction(SequenceBeginsWithSeq, kind),
		"between":          vm.NewCFunction(SequenceBetween, kind),
		"bitAt":            vm.NewCFunction(SequenceBitAt, kind),
		"byteAt":           vm.NewCFunction(SequenceByteAt, kind),
		"cloneAppendSeq":   vm.NewCFunction(SequenceCloneAppendSeq, kind),
		"compare":          vm.NewCFunction(SequenceCompare, kind),
		"contains":         vm.NewCFunction(SequenceContains, kind),
		"containsSeq":      vm.NewCFunction(SequenceContainsSeq, kind),
		"endsWithSeq":      vm.NewCFunction(SequenceEndsWithSeq, kind),
		"exSlice":          vm.NewCFunction(SequenceExSlice, kind),
		"findSeq":          vm.NewCFunction(SequenceFindSeq, kind),
		"findSeqs":         vm.NewCFunction(SequenceFindSeqs, kind),
		"foreach":          vm.NewCFunction(SequenceForeach, kind),
		"hash":             vm.NewCFunction(SequenceHash, kind),
		"inSlice":          vm.NewCFunction(SequenceInSlice, kind),
		"isMutable":        vm.NewCFunction(SequenceIsMutable, kind),
		"isZero":           vm.NewCFunction(SequenceIsZero, kind),
		"itemSize":         vm.NewCFunction(SequenceItemSize, kind),
		"itemType":         vm.NewCFunction(SequenceItemType, kind),
		"occurrencesOfSeq": vm.NewCFunction(SequenceOccurrencesOfSeq, kind),
		"pack":             vm.NewCFunction(SequencePack, nil),
		"reverseFindSeq":   vm.NewCFunction(SequenceReverseFindSeq, kind),
		"size":             vm.NewCFunction(SequenceSize, kind),
		"splitAt":          vm.NewCFunction(SequenceSplitAt, kind),
		"unpack":           vm.NewCFunction(SequenceUnpack, kind),
		"withStruct":       vm.NewCFunction(SequenceWithStruct, nil),

		// sequence-mutable.go:
		"append":              vm.NewCFunction(SequenceAppend, kind),
		"appendSeq":           vm.NewCFunction(SequenceAppendSeq, kind),
		"asMutable":           vm.NewCFunction(SequenceAsMutable, kind),
		"atInsertSeq":         vm.NewCFunction(SequenceAtInsertSeq, kind),
		"atPut":               vm.NewCFunction(SequenceAtPut, kind),
		"clipAfterSeq":        vm.NewCFunction(SequenceClipAfterSeq, kind),
		"clipAfterStartOfSeq": vm.NewCFunction(SequenceClipAfterStartOfSeq, kind),
		"clipBeforeEndOfSeq":  vm.NewCFunction(SequenceClipBeforeEndOfSeq, kind),
		"clipBeforeSeq":       vm.NewCFunction(SequenceClipBeforeSeq, kind),
		"convertToItemType":   vm.NewCFunction(SequenceConvertToItemType, kind),
		"copy":                vm.NewCFunction(SequenceCopy, kind),
		"duplicateIndexes":    vm.NewCFunction(SequenceDuplicateIndexes, kind),
		"empty":               vm.NewCFunction(SequenceEmpty, kind),
		"insertSeqEvery":      vm.NewCFunction(SequenceInsertSeqEvery, kind),
		"leaveThenRemove":     vm.NewCFunction(SequenceLeaveThenRemove, kind),
		"preallocateToSize":   vm.NewCFunction(SequencePreallocateToSize, kind),
		"rangeFill":           vm.NewCFunction(SequenceRangeFill, kind),
		"removeAt":            vm.NewCFunction(SequenceRemoveAt, kind),
		"removeEvenIndexes":   vm.NewCFunction(SequenceRemoveEvenIndexes, kind),
		"removeLast":          vm.NewCFunction(SequenceRemoveLast, kind),
		"removeOddIndexes":    vm.NewCFunction(SequenceRemoveOddIndexes, kind),
		"removePrefix":        vm.NewCFunction(SequenceRemovePrefix, kind),
		"removeSeq":           vm.NewCFunction(SequenceRemoveSeq, kind),
		"removeSlice":         vm.NewCFunction(SequenceRemoveSlice, kind),
		"removeSuffix":        vm.NewCFunction(SequenceRemoveSuffix, kind),
		"replaceFirstSeq":     vm.NewCFunction(SequenceReplaceFirstSeq, kind),
		"replaceSeq":          vm.NewCFunction(SequenceReplaceSeq, kind),
		"reverseInPlace":      vm.NewCFunction(SequenceReverseInPlace, kind),
		"setItemType":         vm.NewCFunction(SequenceSetItemType, kind),
		"setItemsToDouble":    vm.NewCFunction(SequenceSetItemsToDouble, kind),
		"setSize":             vm.NewCFunction(SequenceSetSize, kind),
		"sort":                vm.NewCFunction(SequenceSort, kind),
		"zero":                vm.NewCFunction(SequenceZero, kind),

		// sequence-string.go:
		"appendPathSeq":          vm.NewCFunction(SequenceAppendPathSeq, kind),
		"asBase64":               vm.NewCFunction(SequenceAsBase64, kind),
		"asFixedSizeType":        vm.NewCFunction(SequenceAsFixedSizeType, kind),
		"asIoPath":               vm.NewCFunction(SequenceAsIoPath, kind),
		"asJson":                 vm.NewCFunction(SequenceAsJson, kind),
		"asLatin1":               vm.NewCFunction(SequenceAsLatin1, kind),
		"asMessage":              vm.NewCFunction(SequenceAsMessage, kind),
		"asNumber":               vm.NewCFunction(SequenceAsNumber, kind),
		"asOSPath":               vm.NewCFunction(SequenceAsOSPath, kind),
		"asUTF16":                vm.NewCFunction(SequenceAsUTF16, kind),
		"asUTF32":                vm.NewCFunction(SequenceAsUTF32, kind),
		"asUTF8":                 vm.NewCFunction(SequenceAsUTF8, kind),
		"capitalize":             vm.NewCFunction(SequenceCapitalize, kind),
		"cloneAppendPath":        vm.NewCFunction(SequenceCloneAppendPath, kind),
		"convertToFixedSizeType": vm.NewCFunction(SequenceConvertToFixedSizeType, kind),
		"encoding":               vm.NewCFunction(SequenceEncoding, kind),
		"escape":                 vm.NewCFunction(SequenceEscape, kind),
		"fromBase":               vm.NewCFunction(SequenceFromBase, kind),
		"fromBase64":             vm.NewCFunction(SequenceFromBase64, kind),
		"interpolate":            vm.NewCFunction(SequenceInterpolate, kind),
		"isLowercase":            vm.NewCFunction(SequenceIsLowercase, kind),
		"isUppercase":            vm.NewCFunction(SequenceIsUppercase, kind),
		"lastPathComponent":      vm.NewCFunction(SequenceLastPathComponent, kind),
		"lowercase":              vm.NewCFunction(SequenceLowercase, kind),
		"lstrip":                 vm.NewCFunction(SequenceLstrip, kind),
		"setEncoding":            vm.NewCFunction(SequenceSetEncoding, kind),
		"parseJson":              vm.NewCFunction(SequenceParseJson, kind),
		"pathComponent":          vm.NewCFunction(SequencePathComponent, kind),
		"pathExtension":          vm.NewCFunction(SequencePathExtension, kind),
		"percentDecoded":         vm.NewCFunction(SequencePercentDecoded, kind),
		"percentEncoded":         vm.NewCFunction(SequencePercentEncoded, kind),
		"rstrip":                 vm.NewCFunction(SequenceRstrip, kind),
		"split":                  vm.NewCFunction(SequenceSplit, kind),
		"strip":                  vm.NewCFunction(SequenceStrip, kind),
		"toBase":                 vm.NewCFunction(SequenceToBase, kind),
		"unescape":               vm.NewCFunction(SequenceUnescape, kind),
		"uppercase":              vm.NewCFunction(SequenceUppercase, kind),
		"urlDecoded":             vm.NewCFunction(SequenceUrlDecoded, kind),
		"urlEncoded":             vm.NewCFunction(SequenceUrlEncoded, kind),
		"validEncodings":         vm.NewCFunction(SequenceValidEncodings, nil),

		// sequence-math.go:
		"**=":                     vm.NewCFunction(SequenceStarStarEq, kind),
		"*=":                      vm.NewCFunction(SequenceStarEq, kind),
		"+=":                      vm.NewCFunction(SequencePlusEq, kind),
		"-=":                      vm.NewCFunction(SequenceMinusEq, kind),
		"/=":                      vm.NewCFunction(SequenceSlashEq, kind),
		"Max":                     vm.NewCFunction(SequencePairwiseMax, kind),
		"Min":                     vm.NewCFunction(SequencePairwiseMin, kind),
		"abs":                     vm.NewCFunction(SequenceAbs, kind),
		"acos":                    vm.NewCFunction(SequenceAcos, kind),
		"asBinaryNumber":          vm.NewCFunction(SequenceAsBinaryNumber, kind),
		"asBinarySignedInteger":   vm.NewCFunction(SequenceAsBinarySignedInteger, kind),
		"asBinaryUnsignedInteger": vm.NewCFunction(SequenceAsBinaryUnsignedInteger, kind),
		"asin":                    vm.NewCFunction(SequenceAsin, kind),
		"atan":                    vm.NewCFunction(SequenceAtan, kind),
		"bitCount":                vm.NewCFunction(SequenceBitCount, kind),
		"bitwiseAnd":              vm.NewCFunction(SequenceBitwiseAnd, kind),
		"bitwiseNot":              vm.NewCFunction(SequenceBitwiseNot, kind),
		"bitwiseOr":               vm.NewCFunction(SequenceBitwiseOr, kind),
		"bitwiseXor":              vm.NewCFunction(SequenceBitwiseXor, kind),
		"ceil":                    vm.NewCFunction(SequenceCeil, kind),
		"cos":                     vm.NewCFunction(SequenceCos, kind),
		"cosh":                    vm.NewCFunction(SequenceCosh, kind),
		"distanceTo":              vm.NewCFunction(SequenceDistanceTo, kind),
		"dotProduct":              vm.NewCFunction(SequenceDotProduct, kind),
		"floor":                   vm.NewCFunction(SequenceFloor, kind),
		"log":                     vm.NewCFunction(SequenceLog, kind),
		"log10":                   vm.NewCFunction(SequenceLog10, kind),
		"max":                     vm.NewCFunction(SequenceMax, kind),
		"mean":                    vm.NewCFunction(SequenceMean, kind),
		"meanSquare":              vm.NewCFunction(SequenceMeanSquare, kind),
		"min":                     vm.NewCFunction(SequenceMin, kind),
		"negate":                  vm.NewCFunction(SequenceNegate, kind),
		"normalize":               vm.NewCFunction(SequenceNormalize, kind),
		"product":                 vm.NewCFunction(SequenceProduct, kind),
		"sin":                     vm.NewCFunction(SequenceSin, kind),
		"sinh":                    vm.NewCFunction(SequenceSinh, kind),
		"sqrt":                    vm.NewCFunction(SequenceSqrt, kind),
		"sum":                     vm.NewCFunction(SequenceSum, kind),
		"square":                  vm.NewCFunction(SequenceSquare, kind),
		"tan":                     vm.NewCFunction(SequenceTan, kind),
		"tanh":                    vm.NewCFunction(SequenceTanh, kind),
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
	vm.Core.SetSlots(Slots{
		"Sequence":          ms,
		"ImmutableSequence": is,
		"String":            is,
	})
	// Now that we have the String proto, we can use vm.NewString.
	ms.SetSlot("type", vm.NewString("Sequence"))
	is.SetSlot("type", vm.NewString("ImmutableSequence"))
}
