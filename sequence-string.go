package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"unicode/utf16"
)

// NewString creates a new Sequence object representing the given string in
// UTF-8 encoding.
func (vm *VM) NewString(value string) *Sequence {
	if s, ok := vm.StringMemo[value]; ok {
		return s
	}
	return &Sequence{
		Object: *vm.CoreInstance("String"),
		Value:  []byte(value),
		Kind:   SeqIU8,
		Code:   "utf8",
	}
}

// String returns a string representation of the object.
func (s *Sequence) String() string {
	if s.Code == "number" {
		return s.NumberString()
	}
	if s.Code == "utf8" {
		if s.Kind == SeqMU8 || s.Kind == SeqIU8 {
			// Easy path. The sequence is already a UTF-8 coded string, so we
			// can just convert it and return it.
			return string(s.Value.([]byte))
		}
		// All other kinds reinterpret their bytes as being already UTF-8.
		return string(s.Bytes())
	}
	if s.Code == "ascii" || s.Code == "latin1" {
		d := charmap.Windows1252.NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// Io supports UCS-2, but we support UTF-16. This could conceivably cause
	// some compatibility issues between the two, but the ? operator exists.
	if s.Code == "utf16" {
		if s.Kind == SeqMU16 || s.Kind == SeqIU16 {
			return string(utf16.Decode(s.Value.([]uint16)))
		}
		d := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// Again, we use UTF-32 where Io supports UCS-4. This is probably less of
	// an issue, though, because it's not like anyone uses either. :)
	if s.Code == "utf32" {
		if s.Kind == SeqMS32 || s.Kind == SeqIS32 {
			// rune is an alias for int32, so we can convert directly to
			// string.
			return string(s.Value.([]rune))
		}
		d := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// TODO: We can really support any encoding in x/text/encoding.
	panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
}

// NumberString returns a string containing the values of the sequence
// interpreted numerically.
func (s *Sequence) NumberString() string {
	b := strings.Builder{}
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, c := range v {
			b.WriteString(strconv.FormatFloat(float64(c), 'g', -1, 32))
			b.WriteString(", ")
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, c := range v {
			b.WriteString(strconv.FormatFloat(c, 'g', -1, 64))
			b.WriteString(", ")
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	r := b.String()
	return r[:len(r)-2]
}

// Bytes returns a slice of bytes with the same bit pattern as the sequence.
// The result is always a copy.
func (s *Sequence) Bytes() []byte {
	if s.Kind == SeqMU8 || s.Kind == SeqIU8 {
		return append([]byte{}, s.Value.([]byte)...)
	}
	// encoding/binary uses reflect for floating point types instead of
	// fast-pathing like it does for integer types, so we'll do so manually.
	if s.Kind == SeqMF32 || s.Kind == SeqIF32 {
		v := s.Value.([]float32)
		b := make([]byte, 4*len(v))
		for i, f := range v {
			c := math.Float32bits(f)
			binary.LittleEndian.PutUint32(b[4*i:], c)
		}
		return b
	}
	if s.Kind == SeqMF64 || s.Kind == SeqIF64 {
		v := s.Value.([]float64)
		b := make([]byte, 8*len(v))
		for i, f := range v {
			c := math.Float64bits(f)
			binary.LittleEndian.PutUint64(b[8*i:], c)
		}
		return b
	}
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, s.Value)
	return b.Bytes()
}

// CheckEncoding checks whether the given encoding name is a valid encoding
// accepted by the VM.
func (vm *VM) CheckEncoding(encoding string) bool {
	encoding = strings.ToLower(encoding)
	for _, enc := range vm.ValidEncodings {
		if encoding == enc {
			return true
		}
	}
	return false
}

// SequenceEncoding is a Sequence method.
//
// encoding returns the sequence's encoding.
func SequenceEncoding(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if s.IsMutable() {
		defer MutableMethod(target)()
	}
	return vm.NewString(s.Code)
}

// SequenceSetEncoding is a Sequence method.
//
// setEncoding sets the sequence's encoding. The sequence must be mutable.
// The requested encoding, converted to lower case, must be in the
// validEncodings list.
func SequenceSetEncoding(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("setEncoding"); err != nil {
		return vm.IoError(err)
	}
	defer MutableMethod(target)()
	arg, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	enc := strings.ToLower(arg.String())
	if !vm.CheckEncoding(enc) {
		return vm.RaiseExceptionf("invalid encoding %q", enc)
	}
	s.Code = enc
	return target
}

// SequenceValidEncodings is a Sequence method.
//
// validEncodings returns a list of valid encoding names.
func SequenceValidEncodings(vm *VM, target, locals Interface, msg *Message) Interface {
	encs := make([]Interface, len(vm.ValidEncodings))
	for k, v := range vm.ValidEncodings {
		encs[k] = vm.NewString(v)
	}
	return vm.NewList(encs...)
}

// SequenceAsUTF8 is a Sequence method.
//
// asUTF8 creates a Sequence encoding the receiver in UTF-8.
func SequenceAsUTF8(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if s.IsMutable() {
		defer MutableMethod(target)()
	}
	if s.Code == "utf8" && (s.Kind == SeqMU8 || s.Kind == SeqIU8) {
		return vm.NewSequence(s.Value, s.IsMutable(), "utf8")
	}
	// s.String already does what we want. We could duplicate its logic to
	// avoid extra allocations, but that would make more work if/when we
	// support more encodings.
	v := s.String()
	return vm.NewSequence([]byte(v), s.IsMutable(), "utf8")
}

// SequenceAsUTF16 is a Sequence method.
//
// asUTF16 creates a Sequence encoding the receiver in UTF-16.
func SequenceAsUTF16(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if s.IsMutable() {
		defer MutableMethod(target)()
	}
	if s.Code == "utf16" && (s.Kind == SeqMU16 || s.Kind == SeqIU16) {
		return vm.NewSequence(s.Value, s.IsMutable(), "utf16")
	}
	// Again, we could duplicate s.String to skip extra copies, but :effort:.
	v := []rune(s.String())
	return vm.NewSequence(utf16.Encode(v), s.IsMutable(), "utf16")
}

// SequenceAsUTF32 is a Sequence method.
//
// asUTF32 creates a Sequence encoding the receiver in UTF-32.
func SequenceAsUTF32(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if s.IsMutable() {
		defer MutableMethod(target)()
	}
	if s.Code == "utf32" && (s.Kind == SeqMS32 || s.Kind == SeqIS32) {
		return vm.NewSequence(s.Value, s.IsMutable(), "utf32")
	}
	v := s.String()
	return vm.NewSequence([]rune(v), s.IsMutable(), "utf32")
}
