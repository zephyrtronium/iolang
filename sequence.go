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
//    - sequence_immutable.go: Io methods for non-mutating Sequence methods.
//    - sequence_mutable.go: Io methods for mutating Sequence methods.
//    - sequence_string.go: Implementation of Sequence as a string type,
//        including encodings and representation.
//    - sequence_math.go: Mathematical methods and operations. Eventually,
//        this should have different versions for different arches.

// A Sequence is a collection of data of one fixed-size type.
type Sequence struct {
	Value   interface{}
	Mutable bool
	Code    string // encoding
}

// tagSequence is the Tag type for Sequence values.
type tagSequence struct{}

func (tagSequence) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagSequence) CloneValue(value interface{}) interface{} {
	s := value.(Sequence)
	ns := Sequence{
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
	return ns
}

func (tagSequence) String() string {
	return "Sequence"
}

// SequenceTag is the Tag for Sequence values. Activate returns self.
// CloneValue copies the sequence.
var SequenceTag tagSequence

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

// ItemSize returns the number of bytes required to represent one element of
// the type represented by the SeqKind.
func (kind SeqKind) ItemSize() int {
	switch kind {
	case SeqU8, SeqS8:
		return 1
	case SeqU16, SeqS16:
		return 2
	case SeqU32, SeqS32, SeqF32:
		return 4
	case SeqU64, SeqS64, SeqF64:
		return 8
	default:
		panic(fmt.Sprintf("unrecognized sequence element type %#v", kind.kind))
	}
}

// copySeqVal creates a copy of a Sequence value slice. If the value is that of
// an existing sequence object and that sequence is mutable, then callers
// should hold the object's lock.
func copySeqVal(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		return append([]byte{}, v...)
	case []uint16:
		return append([]uint16{}, v...)
	case []uint32:
		return append([]uint32{}, v...)
	case []uint64:
		return append([]uint64{}, v...)
	case []int8:
		return append([]int8{}, v...)
	case []int16:
		return append([]int16{}, v...)
	case []int32:
		return append([]int32{}, v...)
	case []int64:
		return append([]int64{}, v...)
	case []float32:
		return append([]float32{}, v...)
	case []float64:
		return append([]float64{}, v...)
	default:
		panic(fmt.Sprintf("unsupported value type %T; must be slice of basic fixed-size data type", value))
	}
}

// NewSequence creates a new Sequence object with the given value and with the
// given encoding. The value must be a slice of a basic fixed-size data type,
// and it is copied. Panics if the encoding is not supported.
func (vm *VM) NewSequence(value interface{}, mutable bool, encoding string) *Object {
	if !vm.CheckEncoding(encoding) {
		panic(fmt.Sprintf("unsupported encoding %q", encoding))
	}
	seq := Sequence{
		Value:   copySeqVal(value),
		Mutable: mutable,
		Code:    encoding,
	}
	return vm.ObjectWith(nil, vm.CoreProto("Sequence"), seq, SequenceTag)
}

// SequenceObject creates a new Sequence object with the given value directly.
func (vm *VM) SequenceObject(value Sequence) *Object {
	return vm.ObjectWith(nil, vm.CoreProto("Sequence"), value, SequenceTag)
}

// SequenceFromBytes makes a mutable Sequence with the given type having the
// same bit pattern as the given bytes. If the length of b is not a multiple of
// the item size for the given kind, the extra bytes are ignored. The
// sequence's encoding will be number unless the kind is SeqU8, SeqU16, or
// SeqS32, in which cases the encoding will be utf8, utf16, or utf32,
// respectively.
func (vm *VM) SequenceFromBytes(b []byte, kind SeqKind) Sequence {
	if kind == SeqU8 {
		return Sequence{
			Value:   b,
			Mutable: true,
			Code:    "utf8",
		}
	}
	if kind == SeqF32 {
		v := make([]float32, 0, len(b)/4)
		for len(b) >= 4 {
			c := binary.LittleEndian.Uint32(b)
			v = append(v, math.Float32frombits(c))
			b = b[4:]
		}
		return Sequence{
			Value:   v,
			Mutable: true,
			Code:    "number",
		}
	}
	if kind == SeqF64 {
		v := make([]float64, 0, len(b)/8)
		for len(b) >= 8 {
			c := binary.LittleEndian.Uint64(b)
			v = append(v, math.Float64frombits(c))
			b = b[8:]
		}
		return Sequence{
			Value:   v,
			Mutable: true,
			Code:    "number",
		}
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
	return Sequence{
		Value:   v,
		Mutable: true,
		Code:    kind.Encoding(),
	}
}

// SequenceArgAt evaluates the nth argument and returns its value as a
// Sequence with its object. If a stop occurs during evaluation, the returned
// Sequence has nil value, and the stop result and status are returned. If the
// evaluated result is not a Sequence, the result has nil value, and an
// exception is returned with an ExceptionStop.
func (m *Message) SequenceArgAt(vm *VM, locals *Object, n int) (Sequence, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		seq, ok := v.Value.(Sequence)
		v.Unlock()
		if ok {
			return seq, v, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Sequence, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return Sequence{}, v, s
}

// StringArgAt evaluates the nth argument, asserts that it is a Sequence, and
// returns its value as a string. If a stop occurs during evaluation, the
// returned string is empty, and the stop result and status are returned. If
// the evaluated result is not a Sequence, the result has nil value, and an
// exception is returned with an ExceptionStop.
func (m *Message) StringArgAt(vm *VM, locals *Object, n int) (string, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		str, ok := v.Value.(Sequence)
		if ok {
			r := str.String()
			v.Unlock()
			return r, nil, NoStop
		}
		v.Unlock()
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Sequence, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return "", v, s
}

// AsStringArgAt evaluates the nth argument, then activates its asString slot
// for a string representation. If the result is not a string, then the result
// has nil value, and an exception object is returned with an ExceptionStop.
func (m *Message) AsStringArgAt(vm *VM, locals *Object, n int) (string, *Object, Stop) {
	v, stop := m.EvalArgAt(vm, locals, n)
	if stop != NoStop {
		return "", v, stop
	}
	v, stop = vm.Perform(v, locals, vm.IdentMessage("asString"))
	if stop == NoStop {
		v.Lock()
		str, ok := v.Value.(Sequence)
		v.Unlock()
		if ok {
			return str.String(), nil, NoStop
		}
		v, stop = vm.NewExceptionf("argument %d to %s cannot be converted to string", n, m.Text), ExceptionStop
	}
	return "", v, stop
}

// Kind returns the SeqKind appropriate for this sequence. If the sequence is
// mutable, callers should hold its object's lock.
func (s Sequence) Kind() SeqKind {
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

// IsMutable returns whether the sequence can be modified safely. Callers do
// not need to hold the object's lock, as mutability of a sequence should never
// change.
func (s Sequence) IsMutable() bool {
	return s.Mutable
}

// IsFP returns whether the sequence has a float32 or float64 data type. If the
// sequence is mutable, callers should hold its object's lock.
func (s Sequence) IsFP() bool {
	switch s.Value.(type) {
	case []float32, []float64:
		return true
	}
	return false
}

// SameType returns whether the sequence has the same data type as another. If
// either sequence is mutable, callers should hold their objects' locks.
func (s Sequence) SameType(as Sequence) bool {
	return reflect.TypeOf(s.Value) == reflect.TypeOf(as.Value)
}

// Len returns the length of the sequence. If the sequence is mutable, callers
// should hold its object's lock.
func (s Sequence) Len() int {
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

// FixIndex wraps an index into the sequence's size. If the sequence is
// mutable, callers should hold its object's lock.
func (s Sequence) FixIndex(i int) int {
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

// ItemSize returns the number of bytes required to represent a single element
// of the sequence. If the sequence is mutable, callers should hold its
// object's lock.
func (s Sequence) ItemSize() int {
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
// is out of bounds, the second return value is false. Callers should hold the
// sequence object's lock if the sequence is mutable.
func (s Sequence) At(i int) (float64, bool) {
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
// the result keeps the same number of items. If the sequence is mutable,
// callers should hold its object's lock.
func (s Sequence) Convert(kind SeqKind) Sequence {
	u := reflect.ValueOf(s.Value)
	if u.Type() == kind.kind {
		return Sequence{Value: copySeqVal(s.Value), Mutable: s.Mutable, Code: s.Code}
	}
	n := s.Len()
	v := reflect.MakeSlice(kind.kind, n, n)
	nt := kind.kind.Elem()
	for i := 0; i < n; i++ {
		v.Index(i).Set(u.Index(i).Convert(nt))
	}
	return Sequence{Value: v.Interface(), Mutable: s.Mutable, Code: s.Code}
}

// Append appends other's items to this sequence. If other has a larger item
// size than this sequence, then the result is converted to the item type of
// other. Callers should hold the sequence object's lock. Panics if this
// sequence is not mutable.
func (s Sequence) Append(other Sequence) Sequence {
	if err := s.CheckMutable("Sequence.Append"); err != nil {
		panic(err)
	}
	if s.SameType(other) {
		return s.appendSameKind(other)
	} else if s.ItemSize() >= other.ItemSize() {
		return s.appendConvert(other)
	} else {
		return s.appendGrow(other)
	}
}

func (s Sequence) appendSameKind(other Sequence) Sequence {
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
	return s
}

func (s Sequence) appendConvert(other Sequence) Sequence {
	a := reflect.ValueOf(s.Value)
	b := reflect.ValueOf(other.Value)
	t := a.Type().Elem()
	for i := 0; i < b.Len(); i++ {
		a = reflect.Append(a, b.Index(i).Convert(t))
	}
	s.Value = a.Interface()
	return s
}

func (s Sequence) appendGrow(other Sequence) Sequence {
	old := reflect.ValueOf(s.Value)
	b := reflect.ValueOf(other.Value)
	t := b.Type().Elem()
	a := reflect.MakeSlice(b.Type(), 0, old.Len()+b.Len())
	for i := 0; i < old.Len(); i++ {
		a = reflect.Append(a, old.Index(i).Convert(t))
	}
	a = reflect.AppendSlice(a, b)
	s.Value = a.Interface()
	return s
}

// Insert inserts the elements of another sequence, converted to this
// sequence's type, at a given index. If the index is beyond the length of the
// sequence, then zeros are inserted as needed. Callers should hold the
// sequence object's lock. Panics if k < 0 or if s is immutable.
func (s Sequence) Insert(other Sequence, k int) Sequence {
	if err := s.CheckMutable("Sequence.Insert"); err != nil {
		panic(err)
	}
	if sl := s.Len(); k > sl {
		s = s.extend(k)
	}
	if s.SameType(other) {
		return s.insertSameKind(other, k)
	}
	return s.insertConvert(other, k)
}

func (s Sequence) extend(k int) Sequence {
	s.Value = copySeqVal(s.Value)
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
	return s
}

func (s Sequence) insertSameKind(other Sequence, k int) Sequence {
	s.Value = copySeqVal(s.Value)
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
	return s
}

func (s Sequence) insertConvert(other Sequence, k int) Sequence {
	s.Value = copySeqVal(s.Value)
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
	return s
}

// Find locates the first instance of other in the sequence following start.
// Comparison is done following conversion to the type of s. If there is no
// match, the result is -1. If s or other are mutable, then callers should hold
// their objects' locks.
func (s Sequence) Find(other Sequence, start int) int {
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
// match, the result is -1. If s or other are mutable, then callers should hold
// their objects' locks.
func (s Sequence) RFind(other Sequence, end int) int {
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
// s == other. If s or other are mutable, then callers should hold their
// objects' locks.
func (s Sequence) Compare(other Sequence) int {
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

// Slice selects a linear portion of the sequence. Callers should hold the
// sequence object's lock if it is mutable.
func (s Sequence) Slice(start, stop, step int) Sequence {
	if !s.IsMutable() {
		panic("cannot slice immutable sequence")
	}
	l := s.Len()
	start = fixSliceIndex(start, step, l)
	stop = fixSliceIndex(stop, step, l)
	if step > 0 {
		return s.sliceForward(start, stop, step)
	} else if step < 0 {
		return s.sliceBackward(start, stop, step)
	} else {
		panic("cannot slice with zero step")
	}
}

func (s Sequence) sliceForward(start, stop, step int) Sequence {
	switch v := s.Value.(type) {
	case []byte:
		w := []byte{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint16:
		w := []uint16{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint32:
		w := []uint32{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint64:
		w := []uint64{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int8:
		w := []int8{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int16:
		w := []int16{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int32:
		w := []int32{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int64:
		w := []int64{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []float32:
		w := []float32{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []float64:
		w := []float64{}
		for start < stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return s
}

func (s Sequence) sliceBackward(start, stop, step int) Sequence {
	switch v := s.Value.(type) {
	case []byte:
		w := []byte{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint16:
		w := []uint16{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint32:
		w := []uint32{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []uint64:
		w := []uint64{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int8:
		w := []int8{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int16:
		w := []int16{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int32:
		w := []int32{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []int64:
		w := []int64{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []float32:
		w := []float32{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	case []float64:
		w := []float64{}
		for start > stop {
			w = append(w, v[start])
			start += step
		}
		s.Value = w
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return s
}

// Remove deletes a range of elements from the sequence and returns the new
// sequence value. Callers should hold the sequence object's lock. Panics if
// the sequence is immutable.
func (s Sequence) Remove(i, j int) Sequence {
	if err := s.CheckMutable("Sequence.Remove"); err != nil {
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
	return s
}

// holdSeq obtains the sequence value and acquires the object's lock if it is
// mutable.
func holdSeq(seq *Object) Sequence {
	seq.Lock()
	s := seq.Value.(Sequence)
	if !s.Mutable {
		seq.Unlock()
	}
	return s
}

// unholdSeq releases the object's lock if mutable is true.
func unholdSeq(mutable bool, seq *Object) {
	if mutable {
		seq.Unlock()
	}
}

// lockSeq acquires the object's lock and returns its sequence value. Like
// holdSeq, but for use in mutable methods.
func lockSeq(seq *Object) Sequence {
	seq.Lock()
	return seq.Value.(Sequence)
}

func (vm *VM) initSequence() {
	// We can't use vm.NewString until we create the proto after this.
	slots := Slots{
		// sequence_immutable.go:
		"afterSeq":         vm.NewCFunction(SequenceAfterSeq, SequenceTag),
		"asList":           vm.NewCFunction(SequenceAsList, SequenceTag),
		"asStruct":         vm.NewCFunction(SequenceAsStruct, SequenceTag),
		"asSymbol":         vm.NewCFunction(SequenceAsSymbol, SequenceTag),
		"at":               vm.NewCFunction(SequenceAt, SequenceTag),
		"beforeSeq":        vm.NewCFunction(SequenceBeforeSeq, SequenceTag),
		"beginsWithSeq":    vm.NewCFunction(SequenceBeginsWithSeq, SequenceTag),
		"between":          vm.NewCFunction(SequenceBetween, SequenceTag),
		"bitAt":            vm.NewCFunction(SequenceBitAt, SequenceTag),
		"byteAt":           vm.NewCFunction(SequenceByteAt, SequenceTag),
		"cloneAppendSeq":   vm.NewCFunction(SequenceCloneAppendSeq, SequenceTag),
		"compare":          vm.NewCFunction(SequenceCompare, SequenceTag),
		"contains":         vm.NewCFunction(SequenceContains, SequenceTag),
		"containsSeq":      vm.NewCFunction(SequenceContainsSeq, SequenceTag),
		"endsWithSeq":      vm.NewCFunction(SequenceEndsWithSeq, SequenceTag),
		"exSlice":          vm.NewCFunction(SequenceExSlice, SequenceTag),
		"findSeq":          vm.NewCFunction(SequenceFindSeq, SequenceTag),
		"findSeqs":         vm.NewCFunction(SequenceFindSeqs, SequenceTag),
		"foreach":          vm.NewCFunction(SequenceForeach, SequenceTag),
		"hash":             vm.NewCFunction(SequenceHash, SequenceTag),
		"inSlice":          vm.NewCFunction(SequenceInSlice, SequenceTag),
		"isMutable":        vm.NewCFunction(SequenceIsMutable, SequenceTag),
		"isZero":           vm.NewCFunction(SequenceIsZero, SequenceTag),
		"itemSize":         vm.NewCFunction(SequenceItemSize, SequenceTag),
		"itemType":         vm.NewCFunction(SequenceItemType, SequenceTag),
		"occurrencesOfSeq": vm.NewCFunction(SequenceOccurrencesOfSeq, SequenceTag),
		"pack":             vm.NewCFunction(SequencePack, nil),
		"reverseFindSeq":   vm.NewCFunction(SequenceReverseFindSeq, SequenceTag),
		"size":             vm.NewCFunction(SequenceSize, SequenceTag),
		"splitAt":          vm.NewCFunction(SequenceSplitAt, SequenceTag),
		"unpack":           vm.NewCFunction(SequenceUnpack, SequenceTag),
		"withStruct":       vm.NewCFunction(SequenceWithStruct, nil),

		// sequence_mutable.go:
		"append":              vm.NewCFunction(SequenceAppend, SequenceTag),
		"appendSeq":           vm.NewCFunction(SequenceAppendSeq, SequenceTag),
		"asMutable":           vm.NewCFunction(SequenceAsMutable, SequenceTag),
		"atInsertSeq":         vm.NewCFunction(SequenceAtInsertSeq, SequenceTag),
		"atPut":               vm.NewCFunction(SequenceAtPut, SequenceTag),
		"clipAfterSeq":        vm.NewCFunction(SequenceClipAfterSeq, SequenceTag),
		"clipAfterStartOfSeq": vm.NewCFunction(SequenceClipAfterStartOfSeq, SequenceTag),
		"clipBeforeEndOfSeq":  vm.NewCFunction(SequenceClipBeforeEndOfSeq, SequenceTag),
		"clipBeforeSeq":       vm.NewCFunction(SequenceClipBeforeSeq, SequenceTag),
		"convertToItemType":   vm.NewCFunction(SequenceConvertToItemType, SequenceTag),
		"copy":                vm.NewCFunction(SequenceCopy, SequenceTag),
		"duplicateIndexes":    vm.NewCFunction(SequenceDuplicateIndexes, SequenceTag),
		"empty":               vm.NewCFunction(SequenceEmpty, SequenceTag),
		"insertSeqEvery":      vm.NewCFunction(SequenceInsertSeqEvery, SequenceTag),
		"leaveThenRemove":     vm.NewCFunction(SequenceLeaveThenRemove, SequenceTag),
		"preallocateToSize":   vm.NewCFunction(SequencePreallocateToSize, SequenceTag),
		"rangeFill":           vm.NewCFunction(SequenceRangeFill, SequenceTag),
		"removeAt":            vm.NewCFunction(SequenceRemoveAt, SequenceTag),
		"removeEvenIndexes":   vm.NewCFunction(SequenceRemoveEvenIndexes, SequenceTag),
		"removeLast":          vm.NewCFunction(SequenceRemoveLast, SequenceTag),
		"removeOddIndexes":    vm.NewCFunction(SequenceRemoveOddIndexes, SequenceTag),
		"removePrefix":        vm.NewCFunction(SequenceRemovePrefix, SequenceTag),
		"removeSeq":           vm.NewCFunction(SequenceRemoveSeq, SequenceTag),
		"removeSlice":         vm.NewCFunction(SequenceRemoveSlice, SequenceTag),
		"removeSuffix":        vm.NewCFunction(SequenceRemoveSuffix, SequenceTag),
		"replaceFirstSeq":     vm.NewCFunction(SequenceReplaceFirstSeq, SequenceTag),
		"replaceSeq":          vm.NewCFunction(SequenceReplaceSeq, SequenceTag),
		"reverseInPlace":      vm.NewCFunction(SequenceReverseInPlace, SequenceTag),
		"setItemType":         vm.NewCFunction(SequenceSetItemType, SequenceTag),
		"setItemsToDouble":    vm.NewCFunction(SequenceSetItemsToDouble, SequenceTag),
		"setSize":             vm.NewCFunction(SequenceSetSize, SequenceTag),
		"sort":                vm.NewCFunction(SequenceSort, SequenceTag),
		"zero":                vm.NewCFunction(SequenceZero, SequenceTag),

		// sequence_string.go:
		"appendPathSeq":          vm.NewCFunction(SequenceAppendPathSeq, SequenceTag),
		"asBase64":               vm.NewCFunction(SequenceAsBase64, SequenceTag),
		"asFixedSizeType":        vm.NewCFunction(SequenceAsFixedSizeType, SequenceTag),
		"asIoPath":               vm.NewCFunction(SequenceAsIoPath, SequenceTag),
		"asJson":                 vm.NewCFunction(SequenceAsJSON, SequenceTag),
		"asLatin1":               vm.NewCFunction(SequenceAsLatin1, SequenceTag),
		"asMessage":              vm.NewCFunction(SequenceAsMessage, SequenceTag),
		"asNumber":               vm.NewCFunction(SequenceAsNumber, SequenceTag),
		"asOSPath":               vm.NewCFunction(SequenceAsOSPath, SequenceTag),
		"asUTF16":                vm.NewCFunction(SequenceAsUTF16, SequenceTag),
		"asUTF32":                vm.NewCFunction(SequenceAsUTF32, SequenceTag),
		"asUTF8":                 vm.NewCFunction(SequenceAsUTF8, SequenceTag),
		"capitalize":             vm.NewCFunction(SequenceCapitalize, SequenceTag),
		"cloneAppendPath":        vm.NewCFunction(SequenceCloneAppendPath, SequenceTag),
		"convertToFixedSizeType": vm.NewCFunction(SequenceConvertToFixedSizeType, SequenceTag),
		"encoding":               vm.NewCFunction(SequenceEncoding, SequenceTag),
		"escape":                 vm.NewCFunction(SequenceEscape, SequenceTag),
		"fromBase":               vm.NewCFunction(SequenceFromBase, SequenceTag),
		"fromBase64":             vm.NewCFunction(SequenceFromBase64, SequenceTag),
		"interpolate":            vm.NewCFunction(SequenceInterpolate, SequenceTag),
		"isLowercase":            vm.NewCFunction(SequenceIsLowercase, SequenceTag),
		"isUppercase":            vm.NewCFunction(SequenceIsUppercase, SequenceTag),
		"lastPathComponent":      vm.NewCFunction(SequenceLastPathComponent, SequenceTag),
		"lowercase":              vm.NewCFunction(SequenceLowercase, SequenceTag),
		"lstrip":                 vm.NewCFunction(SequenceLstrip, SequenceTag),
		"setEncoding":            vm.NewCFunction(SequenceSetEncoding, SequenceTag),
		"parseJson":              vm.NewCFunction(SequenceParseJSON, SequenceTag),
		"pathComponent":          vm.NewCFunction(SequencePathComponent, SequenceTag),
		"pathExtension":          vm.NewCFunction(SequencePathExtension, SequenceTag),
		"percentDecoded":         vm.NewCFunction(SequencePercentDecoded, SequenceTag),
		"percentEncoded":         vm.NewCFunction(SequencePercentEncoded, SequenceTag),
		"rstrip":                 vm.NewCFunction(SequenceRstrip, SequenceTag),
		"split":                  vm.NewCFunction(SequenceSplit, SequenceTag),
		"strip":                  vm.NewCFunction(SequenceStrip, SequenceTag),
		"toBase":                 vm.NewCFunction(SequenceToBase, SequenceTag),
		"unescape":               vm.NewCFunction(SequenceUnescape, SequenceTag),
		"uppercase":              vm.NewCFunction(SequenceUppercase, SequenceTag),
		"urlDecoded":             vm.NewCFunction(SequenceURLDecoded, SequenceTag),
		"urlEncoded":             vm.NewCFunction(SequenceURLEncoded, SequenceTag),
		"validEncodings":         vm.NewCFunction(SequenceValidEncodings, nil),

		// sequence_math.go:
		"**=":                     vm.NewCFunction(SequenceStarStarEq, SequenceTag),
		"*=":                      vm.NewCFunction(SequenceStarEq, SequenceTag),
		"+=":                      vm.NewCFunction(SequencePlusEq, SequenceTag),
		"-=":                      vm.NewCFunction(SequenceMinusEq, SequenceTag),
		"/=":                      vm.NewCFunction(SequenceSlashEq, SequenceTag),
		"Max":                     vm.NewCFunction(SequencePairwiseMax, SequenceTag),
		"Min":                     vm.NewCFunction(SequencePairwiseMin, SequenceTag),
		"abs":                     vm.NewCFunction(SequenceAbs, SequenceTag),
		"acos":                    vm.NewCFunction(SequenceAcos, SequenceTag),
		"asBinaryNumber":          vm.NewCFunction(SequenceAsBinaryNumber, SequenceTag),
		"asBinarySignedInteger":   vm.NewCFunction(SequenceAsBinarySignedInteger, SequenceTag),
		"asBinaryUnsignedInteger": vm.NewCFunction(SequenceAsBinaryUnsignedInteger, SequenceTag),
		"asin":                    vm.NewCFunction(SequenceAsin, SequenceTag),
		"atan":                    vm.NewCFunction(SequenceAtan, SequenceTag),
		"bitCount":                vm.NewCFunction(SequenceBitCount, SequenceTag),
		"bitwiseAnd":              vm.NewCFunction(SequenceBitwiseAnd, SequenceTag),
		"bitwiseNot":              vm.NewCFunction(SequenceBitwiseNot, SequenceTag),
		"bitwiseOr":               vm.NewCFunction(SequenceBitwiseOr, SequenceTag),
		"bitwiseXor":              vm.NewCFunction(SequenceBitwiseXor, SequenceTag),
		"ceil":                    vm.NewCFunction(SequenceCeil, SequenceTag),
		"cos":                     vm.NewCFunction(SequenceCos, SequenceTag),
		"cosh":                    vm.NewCFunction(SequenceCosh, SequenceTag),
		"distanceTo":              vm.NewCFunction(SequenceDistanceTo, SequenceTag),
		"dotProduct":              vm.NewCFunction(SequenceDotProduct, SequenceTag),
		"floor":                   vm.NewCFunction(SequenceFloor, SequenceTag),
		"log":                     vm.NewCFunction(SequenceLog, SequenceTag),
		"log10":                   vm.NewCFunction(SequenceLog10, SequenceTag),
		"max":                     vm.NewCFunction(SequenceMax, SequenceTag),
		"mean":                    vm.NewCFunction(SequenceMean, SequenceTag),
		"meanSquare":              vm.NewCFunction(SequenceMeanSquare, SequenceTag),
		"min":                     vm.NewCFunction(SequenceMin, SequenceTag),
		"negate":                  vm.NewCFunction(SequenceNegate, SequenceTag),
		"normalize":               vm.NewCFunction(SequenceNormalize, SequenceTag),
		"product":                 vm.NewCFunction(SequenceProduct, SequenceTag),
		"sin":                     vm.NewCFunction(SequenceSin, SequenceTag),
		"sinh":                    vm.NewCFunction(SequenceSinh, SequenceTag),
		"sqrt":                    vm.NewCFunction(SequenceSqrt, SequenceTag),
		"sum":                     vm.NewCFunction(SequenceSum, SequenceTag),
		"square":                  vm.NewCFunction(SequenceSquare, SequenceTag),
		"tan":                     vm.NewCFunction(SequenceTan, SequenceTag),
		"tanh":                    vm.NewCFunction(SequenceTanh, SequenceTag),
	}
	slots["addEquals"] = slots["+="]
	slots["asBuffer"] = slots["asMutable"]
	slots["asString"] = slots["asSymbol"]
	slots["betweenSeq"] = slots["between"]
	slots["exclusiveSlice"] = slots["exSlice"]
	slots["inclusiveSlice"] = slots["inSlice"]
	slots["slice"] = slots["exSlice"]
	value := Sequence{
		Value:   []byte(nil),
		Mutable: true,
		Code:    "utf8",
	}
	ms := vm.ObjectWith(slots, []*Object{vm.BaseObject}, value, SequenceTag)
	value.Mutable = false
	is := vm.ObjectWith(nil, []*Object{ms}, value, SequenceTag)
	vm.SetSlots(vm.Core, Slots{
		"Sequence":          ms,
		"ImmutableSequence": is,
		"String":            is,
	})
	// Now that we have the String proto, we can use vm.NewString.
	vm.SetSlot(ms, "type", vm.NewString("Sequence"))
	vm.SetSlot(is, "type", vm.NewString("ImmutableSequence"))
}
