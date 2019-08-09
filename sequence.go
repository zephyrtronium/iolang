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
	Value   interface{}
	Mutable bool
	Code    string // encoding
}

// SeqKind represents a sequence data type.
type SeqKind struct {
	kind reflect.Type
}

// SeqKind values.
var (
	SeqU8  = SeqKind{seqU8}
	SeqU16 = SeqKind{seqU16}
	SeqU32 = SeqKind{seqU32}
	SeqU64 = SeqKind{seqU64}
	SeqS8  = SeqKind{seqS8}
	SeqS16 = SeqKind{seqS16}
	SeqS32 = SeqKind{seqS32}
	SeqS64 = SeqKind{seqS64}
	SeqF32 = SeqKind{seqF32}
	SeqF64 = SeqKind{seqF64}
)

var (
	seqU8  reflect.Type = reflect.TypeOf([]byte(nil))
	seqU16 reflect.Type = reflect.TypeOf([]uint16(nil))
	seqU32 reflect.Type = reflect.TypeOf([]uint32(nil))
	seqU64 reflect.Type = reflect.TypeOf([]uint64(nil))
	seqS8  reflect.Type = reflect.TypeOf([]int8(nil))
	seqS16 reflect.Type = reflect.TypeOf([]int16(nil))
	seqS32 reflect.Type = reflect.TypeOf([]int32(nil))
	seqS64 reflect.Type = reflect.TypeOf([]int64(nil))
	seqF32 reflect.Type = reflect.TypeOf([]float32(nil))
	seqF64 reflect.Type = reflect.TypeOf([]float64(nil))
)

// SeqMaxItemSize is the maximum size in bytes of a single sequence element.
const SeqMaxItemSize = 8

// Encoding returns the suggested default encoding for the sequence kind. This
// is utf8 for uint8 kinds, utf16 for uint16, utf32 for int32, and number for
// all other kinds.
func (kind SeqKind) Encoding() string {
	switch kind {
	case SeqU8:
		return "utf8"
	case SeqU16:
		return "utf16"
	case SeqS32:
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
	switch v := value.(type) {
	case []byte:
		value = append([]byte{}, v...)
	case []uint16:
		value = append([]uint16{}, v...)
	case []uint32:
		value = append([]uint32{}, v...)
	case []uint64:
		value = append([]uint64{}, v...)
	case []int8:
		value = append([]int8{}, v...)
	case []int16:
		value = append([]int16{}, v...)
	case []int32:
		value = append([]int32{}, v...)
	case []int64:
		value = append([]int64{}, v...)
	case []float32:
		value = append([]float32{}, v...)
	case []float64:
		value = append([]float64{}, v...)
	default:
		panic(fmt.Sprintf("unsupported value type %T, must be slice of basic fixed-size data type", value))
	}
	return &Sequence{
		Object:  Object{Protos: vm.CoreProto("ImmutableSequence")},
		Value:   value,
		Mutable: mutable,
		Code:    encoding,
	}
}

// SequenceFromBytes makes a mutable Sequence with the given type having the
// same bit pattern as the given bytes. If the length of b is not a multiple of
// the item size for the given kind, the extra bytes are ignored. The
// sequence's encoding will be number unless the kind is SeqU8, SeqU16, or
// SeqS32, in which cases the encoding will be utf8, utf16, or utf32,
// respectively.
func (vm *VM) SequenceFromBytes(b []byte, kind SeqKind) *Sequence {
	if kind == SeqU8 {
		return vm.NewSequence(b, true, "utf8")
	}
	if kind == SeqF32 {
		v := make([]float32, 0, len(b)/4)
		for len(b) >= 4 {
			c := binary.LittleEndian.Uint32(b)
			v = append(v, math.Float32frombits(c))
			b = b[4:]
		}
		return vm.NewSequence(v, true, "number")
	}
	if kind == SeqF64 {
		v := make([]float64, 0, len(b)/8)
		for len(b) >= 8 {
			c := binary.LittleEndian.Uint64(b)
			v = append(v, math.Float64frombits(c))
			b = b[8:]
		}
		return vm.NewSequence(v, true, "number")
	}
	var v interface{}
	switch kind {
	case SeqU16:
		v = make([]uint16, len(b)/2)
	case SeqU32:
		v = make([]uint32, len(b)/4)
	case SeqU64:
		v = make([]uint64, len(b)/8)
	case SeqS8:
		v = make([]int8, len(b))
	case SeqS16:
		v = make([]int16, len(b)/2)
	case SeqS32:
		v = make([]int32, len(b)/4)
	case SeqS64:
		v = make([]int64, len(b)/8)
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", kind))
	}
	binary.Read(bytes.NewReader(b), binary.LittleEndian, v)
	return vm.NewSequence(v, true, kind.Encoding())
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
		Object:  Object{Protos: []Interface{s}},
		Mutable: s.Mutable,
		Code:    s.Code,
	}
	switch v := s.Value.(type) {
	case []byte:
		ns.Value = append([]byte{}, v...)
	case []uint16:
		ns.Value = append([]uint16{}, v...)
	case []uint32:
		ns.Value = append([]uint32{}, v...)
	case []uint64:
		ns.Value = append([]uint64{}, v...)
	case []int8:
		ns.Value = append([]int8{}, v...)
	case []int16:
		ns.Value = append([]int16{}, v...)
	case []int32:
		ns.Value = append([]int32{}, v...)
	case []int64:
		ns.Value = append([]int64{}, v...)
	case []float32:
		ns.Value = append([]float32{}, v...)
	case []float64:
		ns.Value = append([]float64{}, v...)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return &ns
}

// Kind returns the SeqKind appropriate for this sequence.
func (s *Sequence) Kind() SeqKind {
	switch s.Value.(type) {
	case []byte:
		return SeqU8
	case []uint16:
		return SeqU16
	case []uint32:
		return SeqU32
	case []uint64:
		return SeqU64
	case []int8:
		return SeqS8
	case []int16:
		return SeqS16
	case []int32:
		return SeqS32
	case []int64:
		return SeqS64
	case []float32:
		return SeqF32
	case []float64:
		return SeqF64
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// IsMutable returns whether the sequence can be modified safely.
func (s *Sequence) IsMutable() bool {
	return s.Mutable
}

// IsFP returns whether the sequence has a float32 or float64 data type.
func (s *Sequence) IsFP() bool {
	switch s.Value.(type) {
	case []float32, []float64:
		return true
	}
	return false
}

// SameType returns whether the sequence has the same data type as another,
// regardless of mutability.
func (s *Sequence) SameType(as *Sequence) bool {
	return reflect.TypeOf(s.Value) == reflect.TypeOf(as.Value)
}

// Len returns the length of the sequence.
func (s *Sequence) Len() int {
	switch v := s.Value.(type) {
	case []byte:
		return len(v)
	case []uint16:
		return len(v)
	case []uint32:
		return len(v)
	case []uint64:
		return len(v)
	case []int8:
		return len(v)
	case []int16:
		return len(v)
	case []int32:
		return len(v)
	case []int64:
		return len(v)
	case []float32:
		return len(v)
	case []float64:
		return len(v)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
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
	switch s.Value.(type) {
	case []byte, []int8:
		return 1
	case []uint16, []int16:
		return 2
	case []uint32, []int32, []float32:
		return 4
	case []uint64, []int64, []float64:
		return 8
	}
	panic(fmt.Sprintf("unknown sequencet type %T", s.Value))
}

// At returns the value of an item in the sequence as a float64. If the index
// is out of bounds, the second return value is false.
func (s *Sequence) At(i int) (float64, bool) {
	if i < 0 {
		return 0, false
	}
	switch v := s.Value.(type) {
	case []byte:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []uint16:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []uint32:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []uint64:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []int8:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []int16:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []int32:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []int64:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []float32:
		if i >= len(v) {
			return 0, false
		}
		return float64(v[i]), true
	case []float64:
		if i >= len(v) {
			return 0, false
		}
		return v[i], true
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// Convert changes the item type of the sequence. The conversion is such that
// the result keeps the same number of items.
func (s *Sequence) Convert(vm *VM, kind SeqKind) *Sequence {
	u := reflect.ValueOf(s.Value)
	if u.Type() == kind.kind {
		return vm.NewSequence(s.Value, s.Mutable, s.Code)
	}
	n := s.Len()
	v := reflect.MakeSlice(kind.kind, n, n)
	nt := kind.kind.Elem()
	for i := 0; i < n; i++ {
		v.Index(i).Set(u.Index(i).Convert(nt))
	}
	return vm.NewSequence(v.Interface(), s.Mutable, s.Code)
}

// Append appends other's items to this sequence. If other has a larger item
// size than this sequence, then this sequence will be converted to the item
// type of other. Panics if this sequence is not mutable.
func (s *Sequence) Append(other *Sequence) {
	if err := s.CheckMutable("*Sequence.Append"); err != nil {
		panic(err)
	}
	if s.SameType(other) {
		s.appendSameKind(other)
	} else if s.ItemSize() >= other.ItemSize() {
		s.appendConvert(other)
	} else {
		s.appendGrow(other)
	}
}

func (s *Sequence) appendSameKind(other *Sequence) {
	switch v := s.Value.(type) {
	case []byte:
		s.Value = append(v, other.Value.([]byte)...)
	case []uint16:
		s.Value = append(v, other.Value.([]uint16)...)
	case []uint32:
		s.Value = append(v, other.Value.([]uint32)...)
	case []uint64:
		s.Value = append(v, other.Value.([]uint64)...)
	case []int8:
		s.Value = append(v, other.Value.([]int8)...)
	case []int16:
		s.Value = append(v, other.Value.([]int16)...)
	case []int32:
		s.Value = append(v, other.Value.([]int32)...)
	case []int64:
		s.Value = append(v, other.Value.([]int64)...)
	case []float32:
		s.Value = append(v, other.Value.([]float32)...)
	case []float64:
		s.Value = append(v, other.Value.([]float64)...)
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
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
	switch v := s.Value.(type) {
	case []byte:
		if len(v) < k {
			v = append(v, make([]byte, k-len(v))...)
		}
		s.Value = v
	case []uint16:
		if len(v) < k {
			v = append(v, make([]uint16, k-len(v))...)
		}
		s.Value = v
	case []uint32:
		if len(v) < k {
			v = append(v, make([]uint32, k-len(v))...)
		}
		s.Value = v
	case []uint64:
		if len(v) < k {
			v = append(v, make([]uint64, k-len(v))...)
		}
		s.Value = v
	case []int8:
		if len(v) < k {
			v = append(v, make([]int8, k-len(v))...)
		}
		s.Value = v
	case []int16:
		if len(v) < k {
			v = append(v, make([]int16, k-len(v))...)
		}
		s.Value = v
	case []int32:
		if len(v) < k {
			v = append(v, make([]int32, k-len(v))...)
		}
		s.Value = v
	case []int64:
		if len(v) < k {
			v = append(v, make([]int64, k-len(v))...)
		}
		s.Value = v
	case []float32:
		if len(v) < k {
			v = append(v, make([]float32, k-len(v))...)
		}
		s.Value = v
	case []float64:
		if len(v) < k {
			v = append(v, make([]float64, k-len(v))...)
		}
		s.Value = v
	default:
		panic(fmt.Sprintf("unknown sequence kind %T", s.Value))
	}
}

func (s *Sequence) insertSameKind(other *Sequence, k int) {
	switch v := s.Value.(type) {
	case []byte:
		w := other.Value.([]byte)
		v = append(v, make([]byte, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []uint16:
		w := other.Value.([]uint16)
		v = append(v, make([]uint16, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []uint32:
		w := other.Value.([]uint32)
		v = append(v, make([]uint32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []uint64:
		w := other.Value.([]uint64)
		v = append(v, make([]uint64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []int8:
		w := other.Value.([]int8)
		v = append(v, make([]int8, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []int16:
		w := other.Value.([]int16)
		v = append(v, make([]int16, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []int32:
		w := other.Value.([]int32)
		v = append(v, make([]int32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []int64:
		w := other.Value.([]int64)
		v = append(v, make([]int64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []float32:
		w := other.Value.([]float32)
		v = append(v, make([]float32, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	case []float64:
		w := other.Value.([]float64)
		v = append(v, make([]float64, len(w))...)
		copy(v[k+len(w):], v[k:])
		copy(v[k:], w)
		s.Value = v
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
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
// Comparison is done following conversion to the type of s. If there is no
// match, the result is -1.
func (s *Sequence) Find(other *Sequence, start int) int {
	ol := other.Len()
	if ol == 0 {
		return start
	}
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	checks := s.Len() - ol + 1
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		for i := start; i < checks; i++ {
			if findUMatch(m, o, i, ol, mt) {
				return i
			}
		}
	case []int8, []int16, []int32, []int64:
		for i := start; i < checks; i++ {
			if findIMatch(m, o, i, ol, mt) {
				return i
			}
		}
	case []float32, []float64:
		for i := start; i < checks; i++ {
			if findFMatch(m, o, i, ol, mt) {
				return i
			}
		}
	}
	return -1
}

// RFind locates the last instance of other in the sequence ending before end.
// Comparison is done following conversion to the type of s. If there is no
// match, the result is -1.
func (s *Sequence) RFind(other *Sequence, end int) int {
	ol := other.Len()
	if ol == 0 {
		return end
	}
	if end > s.Len() {
		end = s.Len()
	}
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		for i := end - ol; i >= 0; i-- {
			if findUMatch(m, o, i, ol, mt) {
				return i
			}
		}
	case []int8, []int16, []int32, []int64:
		for i := end - ol; i >= 0; i-- {
			if findIMatch(m, o, i, ol, mt) {
				return i
			}
		}
	case []float32, []float64:
		for i := end - ol; i >= 0; i-- {
			if findFMatch(m, o, i, ol, mt) {
				return i
			}
		}
	}
	return -1
}

func findUMatch(m, o reflect.Value, i, ol int, mt reflect.Type) bool {
	for k := 0; k < ol; k++ {
		x := m.Index(i + k).Uint()
		y := o.Index(k).Convert(mt).Uint()
		if x != y {
			return false
		}
	}
	return true
}

func findIMatch(m, o reflect.Value, i, ol int, mt reflect.Type) bool {
	for k := 0; k < ol; k++ {
		x := m.Index(i + k).Int()
		y := o.Index(k).Convert(mt).Int()
		if x != y {
			return false
		}
	}
	return true
}

func findFMatch(m, o reflect.Value, i, ol int, mt reflect.Type) bool {
	for k := 0; k < ol; k++ {
		x := m.Index(i + k).Float()
		y := o.Index(k).Convert(mt).Float()
		if x != y {
			return false
		}
	}
	return true
}

// Compare finds the lexicographical ordering between s and other in the
// element-wise sense, returning -1 if s < other, 1 if s > other, and 0 if
// s == other.
func (s *Sequence) Compare(other *Sequence) int {
	sl := s.Len()
	ol := other.Len()
	n := sl
	if sl > ol {
		n = ol
	}
	m := reflect.ValueOf(s.Value)
	o := reflect.ValueOf(other.Value)
	mt := m.Type().Elem()
	switch s.Value.(type) {
	case []byte, []uint16, []uint32, []uint64:
		for i := 0; i < n; i++ {
			x := m.Index(i).Uint()
			y := o.Index(i).Convert(mt).Uint()
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
		}
	case []int8, []int16, []int32, []int64:
		for i := 0; i < n; i++ {
			x := m.Index(i).Int()
			y := o.Index(i).Convert(mt).Int()
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
		}
	case []float32, []float64:
		for i := 0; i < n; i++ {
			x := m.Index(i).Float()
			y := o.Index(i).Convert(mt).Float()
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
		}
	}
	if sl < ol {
		return -1
	}
	if sl > ol {
		return 1
	}
	return 0
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
	switch v := s.Value.(type) {
	case []byte:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []uint16:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []uint32:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []uint64:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []int8:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []int16:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []int32:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []int64:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []float32:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	case []float64:
		for start < stop {
			v[j] = v[start]
			j++
			start += step
		}
		s.Value = v[:j]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

func (s *Sequence) sliceBackward(start, stop, step int) {
	i, j := start, 0
	switch v := s.Value.(type) {
	case []byte:
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
	case []uint16:
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
	case []uint32:
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
	case []uint64:
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
	case []int8:
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
	case []int16:
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
	case []int32:
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
	case []int64:
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
	case []float32:
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
	case []float64:
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
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
}

// Remove deletes a range of elements from the sequence. Panics if the sequence
// is immutable.
func (s *Sequence) Remove(i, j int) {
	if err := s.CheckMutable("*Sequence.Remove"); err != nil {
		panic(err)
	}
	switch v := s.Value.(type) {
	case []byte:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []uint16:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []uint32:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []uint64:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []int8:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []int16:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []int32:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []int64:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []float32:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	case []float64:
		copy(v[i:], v[j:])
		s.Value = v[:len(v)-(j-i)]
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
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
		Object:  *vm.ObjectWith(slots),
		Value:   []byte(nil),
		Mutable: true,
		Code:    "utf8",
	}
	is := ms.Clone().(*Sequence)
	is.Mutable = false
	vm.SetSlots(vm.Core, Slots{
		"Sequence":          ms,
		"ImmutableSequence": is,
		"String":            is,
	})
	// Now that we have the String proto, we can use vm.NewString.
	vm.SetSlot(ms, "type", vm.NewString("Sequence"))
	vm.SetSlot(is, "type", vm.NewString("ImmutableSequence"))
}
