package iolang

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
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
	if s.Len() == 0 {
		return ""
	}
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
	arg, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
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
	if s.Code == "utf32" && (s.Kind == SeqMS32 || s.Kind == SeqIS32) {
		return vm.NewSequence(s.Value, s.IsMutable(), "utf32")
	}
	v := s.String()
	return vm.NewSequence([]rune(v), s.IsMutable(), "utf32")
}

// SequenceAppendPathSeq is a Sequence method.
//
// appendPathSeq appends a path element to the sequence, removing extra path
// separators between.
func SequenceAppendPathSeq(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("appendPathSeq"); err != nil {
		return vm.IoError(err)
	}
	other, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	sl, ok := s.At(s.Len() - 1)
	if !ok {
		return vm.NewSequence(other.Value, true, other.Code)
	}
	of, ok := other.At(0)
	if !ok {
		return s
	}
	sis := rune(sl) == filepath.Separator || rune(sl) == '/'
	ois := rune(of) == filepath.Separator || rune(of) == '/'
	if sis && ois {
		s.Slice(0, s.Len()-1, 1)
	} else if !sis && !ois {
		s.Append(vm.NewString(string(filepath.Separator)))
	}
	s.Append(other)
	return target
}

// SequenceAsBase64 is a Sequence method.
//
// asBase64 creates a base-64 representation of the bit data of the sequence,
// in accordance with RFC 4648.
func SequenceAsBase64(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	line := 0
	if msg.ArgCount() > 0 {
		n, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != nil {
			return stop
		}
		line = int(n.Value)
	}
	e := base64.StdEncoding.EncodeToString(s.Bytes())
	if line > 0 {
		b := strings.Builder{}
		for len(e) > line {
			b.WriteString(e[:line])
			b.WriteByte('\n')
			e = e[line:]
		}
		b.WriteString(e)
		e = b.String()
	}
	return vm.NewString(e + "\n")
}
