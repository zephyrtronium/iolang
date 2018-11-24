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
