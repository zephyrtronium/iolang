package iolang

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	unichr "unicode"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"unicode/utf16"
	"unicode/utf8"
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

// BytesN returns a slice of up to n bytes with the same bit pattern as the
// corresponding portion of the sequence. The result is always a copy.
func (s *Sequence) BytesN(n int) []byte {
	// We could create a LimitedWriter type to use binary.Write, but that
	// function creates the entire slice to write first, which defeats the
	// purpose of having this be a separate method.
	b := make([]byte, 0, (n+7)/8*8)
	switch s.Kind {
	case SeqMU8, SeqIU8:
		v := s.Value.([]byte)
		b = append(b, v...)
	case SeqMU16, SeqIU16:
		v := s.Value.([]uint16)
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8))
			if len(b) >= n {
				break
			}
		}
	case SeqMU32, SeqIU32:
		v := s.Value.([]uint32)
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case SeqMU64, SeqIU64:
		v := s.Value.([]uint64)
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	case SeqMS8, SeqIS8:
		v := s.Value.([]int8)
		for _, c := range v {
			b = append(b, byte(c))
			if len(b) >= n {
				break
			}
		}
	case SeqMS16, SeqIS16:
		v := s.Value.([]int16)
		for _, x := range v {
			c := uint16(x)
			b = append(b, byte(c), byte(c>>8))
			if len(b) >= n {
				break
			}
		}
	case SeqMS32, SeqIS32:
		v := s.Value.([]int32)
		for _, x := range v {
			c := uint32(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case SeqMS64, SeqIS64:
		v := s.Value.([]int64)
		for _, x := range v {
			c := uint64(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	case SeqMF32, SeqIF32:
		v := s.Value.([]float32)
		for _, x := range v {
			c := math.Float32bits(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case SeqMF64, SeqIF64:
		v := s.Value.([]float64)
		for _, x := range v {
			c := math.Float64bits(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	case SeqUntyped:
		panic("use of untyped sequence")
	default:
		panic(fmt.Sprintf("unknown sequence kind %#v", s.Kind))
	}
	return b
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

// FirstRune decodes the first rune from the sequence and returns its size in
// bytes. If the sequence is empty, the result is (-1, 0).
func (s *Sequence) FirstRune() (rune, int) {
	b := s.BytesN(4)
	if len(b) == 0 {
		return -1, 0
	}
	switch s.Code {
	case "number":
		r, _ := s.At(0)
		return rune(r), utf8.RuneLen(rune(r))
	case "utf8":
		return utf8.DecodeRune(b)
	case "ascii", "latin1":
		return charmap.Windows1252.DecodeByte(b[0]), 1
	case "utf16":
		d := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		// Decoding to UTF-8 will replace an invalid UTF-16 sequence with
		// U+FFFD, which will then decode successfully. This isn't exactly what
		// we want, but it's too much effort to support what should be an
		// uncommon case.
		var err error
		b, err = d.Bytes(b)
		if err != nil {
			return -1, 0
		}
		r, _ := utf8.DecodeRune(b)
		if r < 0x010000 {
			return r, 2
		}
		return r, 4
	case "utf32":
		d := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
		// It would be slightly easier to detect invalid UTF-32, since we could
		// just decode four bytes and use utf8.ValidRune, but this should be an
		// even rarer case than UTF-16.
		var err error
		b, err = d.Bytes(b)
		if err != nil {
			return -1, 0
		}
		r, _ := utf8.DecodeRune(b)
		return r, 4
	}
	// TODO: We can really support any encoding in x/text/encoding.
	panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
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

// SequenceAsLatin1 is a Sequence method.
//
// asLatin1 creates a Sequence encoding the receiver in Latin-1 (Windows-1252).
// Unrepresentable characters will be encoded as a byte with the value 0x1A.
func SequenceAsLatin1(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if (s.Code == "ascii" || s.Code == "latin1") && (s.Kind == SeqMU8 || s.Kind == SeqIU8) {
		return vm.NewSequence(s.Value, s.IsMutable(), "latin1")
	}
	v := s.String()
	// Using the Windows-1252 encoder's Bytes method fails entirely if an
	// invalid rune is encountered, so we have to do the whole thing to ensure
	// we get our replacement bytes.
	r := make([]byte, 0, len(v))
	for _, c := range v {
		ec, _ := charmap.Windows1252.EncodeRune(c)
		r = append(r, ec)
	}
	return vm.NewSequence(r, s.IsMutable(), "latin1")
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

// SequenceAsFixedSizeType is a Sequence method.
//
// asFixedSizeType creates a copy of the sequence encoded in the first of
// UTF-8, UTF-16, or UTF-32 that will encode each rune in a single word.
// Number encoding is copied directly. Latin-1/ASCII is copied to a uint8
// sequence, but is not attempted as a destination encoding. If an erroneous
// encoding is encountered, the result will be numeric.
func SequenceAsFixedSizeType(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	code := "utf8"
	switch s.Code {
	case "number":
		return vm.NewSequence(s.Value, s.IsMutable(), "number")
	case "utf8":
		b := s.Bytes()
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if n > 1 {
				if r > 0xffff || 0xd800 <= r && r < 0xe000 {
					code = "utf32"
					break
				} else {
					code = "utf16"
				}
			}
			if r < 0 {
				code = "number"
				break
			}
			i += n
		}
	case "ascii", "latin1":
		return vm.NewSequence(s.Bytes(), s.IsMutable(), s.Code)
	case "utf16":
		d := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		b, err := d.Bytes(s.Bytes()) // bytes :)
		if err != nil {
			code = "number"
			break
		}
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if r > 0xffff || 0xd800 <= r && r < 0xe000 {
				code = "utf32"
				break
			} else if r > 0x7f {
				code = "utf16"
			}
			i += n
		}
	case "utf32":
		d := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
		b, err := d.Bytes(s.Bytes()) // bytes :)
		if err != nil {
			code = "number"
			break
		}
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if r > 0xffff || 0xd800 <= r && r < 0xe000 {
				code = "utf32"
				break
			} else if r > 0x7f {
				code = "utf16"
			}
			i += n
		}
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
	}
	switch code {
	case "utf8":
		return SequenceAsUTF8(vm, target, locals, msg)
	case "utf16":
		return SequenceAsUTF16(vm, target, locals, msg)
	case "utf32":
		return SequenceAsUTF32(vm, target, locals, msg)
	case "number":
		return vm.NewSequence(s.Value, s.IsMutable(), "number")
	}
	panic("unreachable")
}

// SequenceAsIoPath is a Sequence method.
//
// asIoPath creates a sequence converting the receiver to Io's path convention.
func SequenceAsIoPath(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.NewString(filepath.ToSlash(s.String()))
}

// SequenceAsMessage is a Sequence method.
//
// asMessage compiles the sequence to a Message.
func SequenceAsMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	label := "[asMessage]"
	if msg.ArgCount() > 0 {
		r, stop := msg.StringArgAt(vm, locals, 0)
		if stop != nil {
			return stop
		}
		label = r.String()
	}
	m, err := vm.Parse(strings.NewReader(s.String()), label)
	if err != nil {
		return vm.IoError(err)
	}
	if err := vm.OpShuffle(m); err != nil {
		return err
	}
	return m
}

// SequenceAsOSPath is a Sequence method.
//
// asOSPath creates a sequence converting the receiver to the host operating
// system's path convention.
func SequenceAsOSPath(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	return vm.NewString(filepath.FromSlash(s.String()))
}

// SequenceCapitalize is a Sequence method.
//
// capitalize replaces the first rune in the sequence with the capitalized
// equivalent. This does not use special (Turkish) casing.
func SequenceCapitalize(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("capitalize"); err != nil {
		return vm.IoError(err)
	}
	fr, bn := s.FirstRune()
	if fr < 0 {
		return target
	}
	r := unichr.ToUpper(fr)
	if r == fr {
		return target
	}
	is := s.ItemSize()
	nn := (utf8.RuneLen(r) + is - 1) / is * is
	b := []byte(string(r)) // UTF-8
	switch s.Code {
	case "number":
		v := reflect.ValueOf(s.Value).Index(0)
		v.Set(reflect.ValueOf(r).Convert(v.Type()))
	case "utf8":
		if nn == bn {
			// Straight replacement.
			v := reflect.ValueOf(s.Value)
			x := reflect.MakeSlice(v.Type(), nn/is, nn/is)
			binary.Read(bytes.NewReader(b), binary.LittleEndian, x.Interface())
			// FIXME: This will overwrite if the number of bytes in the rune
			// isn't a multiple of the item size. Please, just use uint8.
			reflect.Copy(v, x)
		} else {
			// Mismatch in encoded sizes. This happens for thirty runes encoded
			// in UTF-8, two of which change from 2 bytes to 1, seventeen from
			// 2 to 3, and eleven from 3 to 2. This is an obnoxious case, so
			// we're going to be obnoxious in handling it.
			v := s.Bytes()
			v = append(b, v[bn:]...)
			for len(v)%is != 0 {
				v = append(v, 0)
			}
			x := vm.SequenceFromBytes(v, s.Kind)
			s.Value = x.Value
		}
	case "ascii", "latin1":
		c, _ := charmap.Windows1252.EncodeRune(r)
		v := reflect.ValueOf(s.Value).Index(0)
		// FIXME: This will overwrite if the type isn't 8-bit.
		v.Set(reflect.ValueOf(c).Convert(v.Type()))
	case "utf16":
		e := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
		b, _ = e.Bytes(b)
		v := reflect.ValueOf(s.Value)
		x := reflect.MakeSlice(v.Type(), len(b)/is, len(b)/is)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, x.Interface())
		// FIXME: This will overwrite if the rune is a single code unit and the
		// sequence kind is 32, or if the sequence kind is 64-bit.
		reflect.Copy(v, x)
	case "utf32":
		e := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewEncoder()
		b, _ = e.Bytes(b)
		v := reflect.ValueOf(s.Value)
		x := reflect.MakeSlice(v.Type(), len(b)/is, len(b)/is)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, x.Interface())
		// FIXME: This will overwrite if the sequence kind is 64-bit.
		reflect.Copy(v, x)
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
	}
	return target
}

// SequenceEscape is a Sequence method.
//
// escape replaces control and non-printable characters with backslash-escaped
// equivalents.
func SequenceEscape(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("escape"); err != nil {
		return vm.IoError(err)
	}
	ss := []byte(strconv.Quote(s.String()))
	ss = ss[1 : len(ss)-1]
	switch s.Kind {
	case SeqMU8:
		s.Value = ss
	case SeqMU16:
		v := make([]uint16, len(ss))
		for i, x := range ss {
			v[i] = uint16(x)
		}
		s.Value = v
	case SeqMU32:
		v := make([]uint32, len(ss))
		for i, x := range ss {
			v[i] = uint32(x)
		}
		s.Value = v
	case SeqMU64:
		v := make([]uint64, len(ss))
		for i, x := range ss {
			v[i] = uint64(x)
		}
		s.Value = v
	case SeqMS8:
		v := make([]int8, len(ss))
		for i, x := range ss {
			v[i] = int8(x)
		}
		s.Value = v
	case SeqMS16:
		v := make([]int16, len(ss))
		for i, x := range ss {
			v[i] = int16(x)
		}
		s.Value = v
	case SeqMS32:
		v := make([]int32, len(ss))
		for i, x := range ss {
			v[i] = int32(x)
		}
		s.Value = v
	case SeqMS64:
		v := make([]int64, len(ss))
		for i, x := range ss {
			v[i] = int64(x)
		}
		s.Value = v
	case SeqMF32:
		v := make([]float32, len(ss))
		for i, x := range ss {
			v[i] = float32(x)
		}
		s.Value = v
	case SeqMF64:
		v := make([]float64, len(ss))
		for i, x := range ss {
			v[i] = float64(x)
		}
		s.Value = v
	}
	return target
}

// SequenceLowercase is a Sequence method.
//
// lowercase converts the values in the sequence to their capitalized
// equivalents. This does not use special (Turkish) casing.
func SequenceLowercase(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("lowercase"); err != nil {
		return vm.IoError(err)
	}
	sv := strings.ToLower(s.String())
	switch s.Code {
	case "number":
		t := reflect.TypeOf(s.Value).Elem()
		v := reflect.MakeSlice(t, 0, s.Len())
		for _, r := range sv {
			v = reflect.Append(v, reflect.ValueOf(r).Convert(t))
		}
		s.Value = v.Interface()
	case "utf8":
		if s.Kind == SeqMU8 {
			s.Value = []byte(sv)
		} else {
			n := (len(sv) + s.ItemSize() - 1) / s.ItemSize()
			v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
			binary.Read(strings.NewReader(sv), binary.LittleEndian, v.Interface())
			s.Value = v.Interface()
		}
	case "ascii", "latin1":
		// This can't err because the original was also Latin-1.
		b, _ := charmap.Windows1252.NewEncoder().Bytes([]byte(sv))
		if s.Kind == SeqMU8 {
			s.Value = b
		} else {
			n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
			v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
			binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
			s.Value = v.Interface()
		}
	case "utf16":
		b, _ := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder().Bytes([]byte(sv))
		n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
		v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
		s.Value = v.Interface()
	case "utf32":
		b, _ := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewEncoder().Bytes([]byte(sv))
		n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
		v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
		s.Value = v.Interface()
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
	}
	return target
}

// SequenceUppercase is a Sequence method.
//
// uppercase converts the values in the sequence to their capitalized
// equivalents. This does not use special (Turkish) casing.
func SequenceUppercase(vm *VM, target, locals Interface, msg *Message) Interface {
	s := target.(*Sequence)
	if err := s.CheckMutable("uppercase"); err != nil {
		return vm.IoError(err)
	}
	sv := strings.ToUpper(s.String())
	switch s.Code {
	case "number":
		t := reflect.TypeOf(s.Value).Elem()
		v := reflect.MakeSlice(t, 0, s.Len())
		for _, r := range sv {
			v = reflect.Append(v, reflect.ValueOf(r).Convert(t))
		}
		s.Value = v.Interface()
	case "utf8":
		if s.Kind == SeqMU8 {
			s.Value = []byte(sv)
		} else {
			n := (len(sv) + s.ItemSize() - 1) / s.ItemSize()
			v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
			binary.Read(strings.NewReader(sv), binary.LittleEndian, v.Interface())
			s.Value = v.Interface()
		}
	case "ascii", "latin1":
		// This can't err because the original was also Latin-1.
		b, _ := charmap.Windows1252.NewEncoder().Bytes([]byte(sv))
		if s.Kind == SeqMU8 {
			s.Value = b
		} else {
			n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
			v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
			binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
			s.Value = v.Interface()
		}
	case "utf16":
		b, _ := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder().Bytes([]byte(sv))
		n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
		v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
		s.Value = v.Interface()
	case "utf32":
		b, _ := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewEncoder().Bytes([]byte(sv))
		n := (len(b) + s.ItemSize() - 1) / s.ItemSize()
		v := reflect.MakeSlice(reflect.TypeOf(s.Value), n, n)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
		s.Value = v.Interface()
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
	}
	return target
}
