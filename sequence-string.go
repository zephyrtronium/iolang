package iolang

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"unicode/utf16"
)

// String returns a string representation of the object.
func (s *Sequence) String() string {
	if s.IsMutable() {
		defer MutableMethod(s)()
	}
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
	// We claim to support UCS-2 because Io does, but it's inferior to UTF-16
	// in probably every way, plus UCS2 would actually be more work to
	// implement.
	if s.Code == "ucs2" || s.Code == "utf16" {
		if s.Kind == SeqMU16 || s.Kind == SeqIU16 {
			return string(utf16.Decode(s.Value.([]uint16)))
		}
		d := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// Again, we claim UCS-4 but do UTF-32. In this case, the benefit is less
	// clear, since UTF-32 is a strict subset of UCS-4, but it's not like
	// anyone uses UCS-4 anyway. :)
	if s.Code == "ucs4" || s.Code == "utf32" {
		if s.Kind == SeqMS32 || s.Kind == SeqIS32 {
			// rune is an alias for int32, so we can convert directly to
			// string.
			return string(s.Value.([]rune))
		}
		d := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
}

func (s *Sequence) NumberString() string {
	return fmt.Sprint(s.Value) // lazy
}

// Bytes returns a slice of bytes with the same bit pattern as the sequence.
// The result is always a copy.
func (s *Sequence) Bytes() []byte {
	if s.IsMutable() {
		defer MutableMethod(s)()
	}
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

// NewString creates a new Sequence object representing the given string in
// UTF-8 encoding.
func (vm *VM) NewString(value string) *Sequence {
	if s, ok := vm.StringMemo[value]; ok {
		return s
	}
	return &Sequence{
		Object: *vm.CoreInstance("String"),
		Value: []byte(value),
		Kind: SeqIU8,
		Code: "utf8",
	}
}

func (vm *VM) initString(slots Slots) {
}
