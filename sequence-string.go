package iolang

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	unichr "unicode"

	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

var (
	encLatin1 = charmap.Windows1252
	encUTF16  = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	encUTF32  = utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)
)

// validEncodings is the list of accepted sequence encodings.
var validEncodings = []string{"ascii", "utf8", "number", "latin1", "utf16", "utf32"}

// NewString creates a new Sequence object representing the given string in
// UTF-8 encoding.
func (vm *VM) NewString(value string) *Object {
	return &Object{
		Protos: vm.CoreProto("String"),
		Value: Sequence{
			Value:   []byte(value),
			Mutable: false,
			Code:    "utf8",
		},
		Tag: SequenceTag,
	}
}

// String returns a string representation of the object. If the sequence is
// mutable, callers should hold its object's lock.
func (s Sequence) String() string {
	if s.Code == "number" {
		return s.NumberString()
	}
	if s.Code == "utf8" {
		if v, ok := s.Value.([]byte); ok {
			// Easy path. The sequence is already a UTF-8 coded string, so we
			// can just convert it and return it.
			return string(v)
		}
		// All other kinds reinterpret their bytes as being already UTF-8.
		return string(s.Bytes())
	}
	if s.Code == "ascii" || s.Code == "latin1" {
		d := encLatin1.NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// Io supports UCS-2, but we support UTF-16. This could conceivably cause
	// some compatibility issues between the two, but the ? operator exists.
	if s.Code == "utf16" {
		if v, ok := s.Value.([]uint16); ok {
			return string(utf16.Decode(v))
		}
		d := encUTF16.NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// Again, we use UTF-32 where Io supports UCS-4. This is probably less of
	// an issue, though, because it's not like anyone uses either. :)
	if s.Code == "utf32" {
		if v, ok := s.Value.([]rune); ok {
			// rune is an alias for int32, so we can convert directly to
			// string.
			return string(v)
		}
		d := encUTF32.NewDecoder()
		b, _ := d.Bytes(s.Bytes()) // bytes :)
		return string(b)
	}
	// TODO: We can really support any encoding in x/text/encoding.
	panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
}

// NumberString returns a string containing the values of the sequence
// interpreted numerically. If the sequence is mutable, callers should hold
// its object's lock.
func (s Sequence) NumberString() string {
	if s.Len() == 0 {
		return ""
	}
	b := strings.Builder{}
	switch v := s.Value.(type) {
	case []byte:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []uint16:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []uint32:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []uint64:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []int8:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []int16:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []int32:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []int64:
		for _, c := range v {
			b.WriteString(fmt.Sprintf("%d, ", c))
		}
	case []float32:
		for _, c := range v {
			b.WriteString(strconv.FormatFloat(float64(c), 'g', -1, 32))
			b.WriteString(", ")
		}
	case []float64:
		for _, c := range v {
			b.WriteString(strconv.FormatFloat(c, 'g', -1, 64))
			b.WriteString(", ")
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	r := b.String()
	return r[:len(r)-2]
}

// Bytes returns a slice of bytes with the same bit pattern as the sequence.
// The result is always a copy. If the sequence is mutable, callers should hold
// its object's lock.
func (s Sequence) Bytes() []byte {
	switch v := s.Value.(type) {
	case []byte:
		return append([]byte{}, v...)
	case []float32:
		// encoding/binary uses reflect for floating point types instead of
		// fast-pathing like it does for integer types, so we do so manually.
		b := make([]byte, 4*len(v))
		for i, f := range v {
			c := math.Float32bits(f)
			binary.LittleEndian.PutUint32(b[4*i:], c)
		}
		return b
	case []float64:
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
// corresponding portion of the sequence. The result is always a copy. If the
// sequence is mutable, callers should hold its object's lock.
func (s Sequence) BytesN(n int) []byte {
	if n > s.Len()*s.ItemSize() {
		n = s.Len() * s.ItemSize()
	}
	// We could create a LimitedWriter type to use binary.Write, but that
	// function creates the entire slice to write first, which defeats the
	// purpose of having this be a separate method.
	b := make([]byte, 0, (n+7)/8*8)
	switch v := s.Value.(type) {
	case []byte:
		b = append(b, v[:n]...)
	case []uint16:
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8))
			if len(b) >= n {
				break
			}
		}
	case []uint32:
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case []uint64:
		for _, c := range v {
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	case []int8:
		for _, c := range v {
			b = append(b, byte(c))
			if len(b) >= n {
				break
			}
		}
	case []int16:
		for _, x := range v {
			c := uint16(x)
			b = append(b, byte(c), byte(c>>8))
			if len(b) >= n {
				break
			}
		}
	case []int32:
		for _, x := range v {
			c := uint32(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case []int64:
		for _, x := range v {
			c := uint64(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	case []float32:
		for _, x := range v {
			c := math.Float32bits(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
			if len(b) >= n {
				break
			}
		}
	case []float64:
		for _, x := range v {
			c := math.Float64bits(x)
			b = append(b, byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
				byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
			if len(b) >= n {
				break
			}
		}
	default:
		panic(fmt.Sprintf("unknown sequence type %T", s.Value))
	}
	return b[:n]
}

// EncodeString creates a new Sequence value encoding sv using the given
// encoding and sequence element type. If the encoding is number, the sequence
// will be each rune of sv converted to the sequence's type; otherwise, the
// value will read the packed binary representation of the encoded string. Any
// unrepresentable runes are converted to replacements. The returned sequence
// is marked mutable, but it may be set to immutable before first use.
func EncodeString(sv string, encoding string, kind SeqKind) Sequence {
	var b []byte
	switch strings.ToLower(encoding) {
	case "number":
		t := kind.kind
		v := reflect.MakeSlice(t, 0, len(sv))
		t = t.Elem()
		for _, r := range sv {
			v = reflect.Append(v, reflect.ValueOf(r).Convert(t))
		}
		return Sequence{Value: v.Interface(), Mutable: true, Code: encoding}
	case "utf8":
		b = []byte(sv)
	case "ascii", "latin1":
		// Encode by hand to make sure we get replacement encodings.
		b = make([]byte, 0, len(sv))
		for _, r := range sv {
			c, _ := encLatin1.EncodeRune(r)
			b = append(b, c)
		}
	case "utf16":
		b, _ = encUTF16.NewEncoder().Bytes([]byte(sv))
	case "utf32":
		b, _ = encUTF32.NewEncoder().Bytes([]byte(sv))
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", encoding))
	}
	if kind == SeqU8 {
		return Sequence{Value: b, Mutable: true, Code: encoding}
	}
	// Round up the length of the buffer to the next multiple of the item size
	// so that we use every rune.
	k := kind.ItemSize()
	if len(b)%k != 0 {
		x := [SeqMaxItemSize]byte{}
		b = append(b, x[:k-len(b)%k]...)
	}
	v := reflect.MakeSlice(kind.kind, len(b)/k, len(b)/k)
	binary.Read(bytes.NewReader(b), binary.LittleEndian, v.Interface())
	return Sequence{Value: v.Interface(), Mutable: true, Code: encoding}
}

// FirstRune decodes the first rune from the sequence and returns its size in
// bytes. If the sequence is empty, the result is (-1, 0). If the sequence is
// mutable, callers should hold its object's lock.
func (s Sequence) FirstRune() (rune, int) {
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
		return encLatin1.DecodeByte(b[0]), 1
	case "utf16":
		d := encUTF16.NewDecoder()
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
		d := encUTF32.NewDecoder()
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

var latin1range = unichr.RangeTable{
	R16: []unichr.Range16{
		{0x0000, 0x007f, 1},
		{0x0081, 0x0081, 1},
		{0x008d, 0x008d, 1},
		{0x008f, 0x0090, 1},
		{0x009d, 0x009d, 1},
		{0x00a0, 0x00ff, 1},
		{0x0152, 0x0153, 1},
		{0x0160, 0x0161, 1},
		{0x0178, 0x0178, 1},
		{0x017d, 0x017e, 1},
		{0x0192, 0x0192, 1},
		{0x02c6, 0x02c6, 1},
		{0x02dc, 0x02dc, 1},
		{0x2013, 0x2014, 1},
		{0x2018, 0x201a, 1},
		{0x201c, 0x201e, 1},
		{0x2020, 0x2022, 1},
		{0x2026, 0x2026, 1},
		{0x2030, 0x2030, 1},
		{0x2039, 0x203a, 1},
		{0x20ac, 0x20ac, 1},
		{0x2122, 0x2122, 1},
	},
	LatinOffset: 229,
}

// MinCode determines the smallest supported encoding which can encode every
// rune in the sequence. If there is any rune which cannot be decoded, or if
// the sequence encoding is already number, the result is number. The order of
// preference is UTF-8, Latin-1, UTF-16, UTF-32, number.
func (s Sequence) MinCode() string {
	code := "utf8"
	l1ok := true
	switch s.Code {
	case "number":
		return "number"
	case "utf8":
		b := s.Bytes()
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if r < 0 {
				return "number"
			}
			if n > 1 {
				if r > 0xffff || 0xd800 <= r && r < 0xe000 {
					code = "utf32"
				} else {
					code = "utf16"
				}
			}
			if l1ok && !unichr.Is(&latin1range, r) {
				l1ok = false
			}
			i += n
		}
	case "ascii", "latin1":
		return s.Code
	case "utf16":
		d := encUTF16.NewDecoder()
		b, err := d.Bytes(s.Bytes()) // bytes :)
		if err != nil {
			return "number"
		}
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if r > 0xffff || 0xd800 <= r && r < 0xe000 {
				code = "utf32"
			} else if r > 0x7f {
				code = "utf16"
			}
			if l1ok && !unichr.Is(&latin1range, r) {
				l1ok = false
			}
			i += n
		}
	case "utf32":
		d := encUTF32.NewDecoder()
		b, err := d.Bytes(s.Bytes()) // bytes :)
		if err != nil {
			return "number"
		}
		i := 0
		for i < len(b) {
			r, n := utf8.DecodeRune(b[i:])
			if r > 0xffff || 0xd800 <= r && r < 0xe000 {
				code = "utf32"
			} else if r > 0x7f {
				code = "utf16"
			}
			if l1ok && !unichr.Is(&latin1range, r) {
				l1ok = false
			}
			i += n
		}
	default:
		// TODO: We can really support any encoding in x/text/encoding.
		panic(fmt.Sprintf("unsupported sequence encoding %q", s.Code))
	}
	if l1ok && code != "utf8" {
		return "ascii"
	}
	return code
}

// CheckEncoding checks whether the given encoding name is a valid encoding
// accepted by the VM.
func (vm *VM) CheckEncoding(encoding string) bool {
	encoding = strings.ToLower(encoding)
	for _, enc := range validEncodings {
		if encoding == enc {
			return true
		}
	}
	return false
}

// SequenceEncoding is a Sequence method.
//
// encoding returns the sequence's encoding.
func SequenceEncoding(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	c := s.Code
	unholdSeq(s.Mutable, target)
	return vm.NewString(c)
}

// SequenceSetEncoding is a Sequence method.
//
// setEncoding sets the sequence's encoding. The sequence must be mutable.
// The requested encoding, converted to lower case, must be in the
// validEncodings list.
func SequenceSetEncoding(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("setEncoding"); err != nil {
		return vm.IoError(err)
	}
	enc, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	enc = strings.ToLower(enc)
	if !vm.CheckEncoding(enc) {
		return vm.RaiseExceptionf("invalid encoding %q", enc)
	}
	s.Code = enc
	target.Value = s
	return target
}

// SequenceValidEncodings is a Sequence method.
//
// validEncodings returns a list of valid encoding names.
func SequenceValidEncodings(vm *VM, target, locals Interface, msg *Message) *Object {
	encs := make([]Interface, len(validEncodings))
	for k, v := range validEncodings {
		encs[k] = vm.NewString(v)
	}
	return vm.NewList(encs...)
}

// SequenceAsLatin1 is a Sequence method.
//
// asLatin1 creates a Sequence encoding the receiver in Latin-1 (Windows-1252).
// Unrepresentable characters will be encoded as a byte with the value 0x1A.
func SequenceAsLatin1(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	if _, ok := s.Value.([]uint8); ok && (s.Code == "ascii" || s.Code == "latin1") {
		unholdSeq(s.Mutable, target)
		return vm.NewSequence(s.Value, s.IsMutable(), "latin1")
	}
	v := s.String()
	unholdSeq(s.Mutable, target)
	// Using the Windows-1252 encoder's Bytes method fails entirely if an
	// invalid rune is encountered, so we have to do the whole thing to ensure
	// we get our replacement bytes.
	r := make([]byte, 0, len(v))
	for _, c := range v {
		ec, _ := encLatin1.EncodeRune(c)
		r = append(r, ec)
	}
	return vm.NewSequence(r, s.IsMutable(), "latin1")
}

// SequenceAsUTF8 is a Sequence method.
//
// asUTF8 creates a Sequence encoding the receiver in UTF-8.
func SequenceAsUTF8(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	if _, ok := s.Value.([]uint8); ok && s.Code == "utf8" {
		unholdSeq(s.Mutable, target)
		return vm.NewSequence(s.Value, s.IsMutable(), "utf8")
	}
	// s.String already does what we want. We could duplicate its logic to
	// avoid extra allocations, but that would make more work if/when we
	// support more encodings.
	v := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewSequence([]byte(v), s.IsMutable(), "utf8")
}

// SequenceAsUTF16 is a Sequence method.
//
// asUTF16 creates a Sequence encoding the receiver in UTF-16.
func SequenceAsUTF16(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	if _, ok := s.Value.([]uint16); ok && s.Code == "utf16" {
		unholdSeq(s.Mutable, target)
		return vm.NewSequence(s.Value, s.IsMutable(), "utf16")
	}
	// Again, we could duplicate s.String to skip extra copies, but :effort:.
	v := []rune(s.String())
	unholdSeq(s.Mutable, target)
	return vm.NewSequence(utf16.Encode(v), s.IsMutable(), "utf16")
}

// SequenceAsUTF32 is a Sequence method.
//
// asUTF32 creates a Sequence encoding the receiver in UTF-32.
func SequenceAsUTF32(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	if _, ok := s.Value.([]rune); ok && s.Code == "utf32" {
		unholdSeq(s.Mutable, target)
		return vm.NewSequence(s.Value, s.IsMutable(), "utf32")
	}
	v := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewSequence([]rune(v), s.IsMutable(), "utf32")
}

// SequenceAppendPathSeq is a Sequence method.
//
// appendPathSeq appends a path element to the sequence, removing extra path
// separators between.
func SequenceAppendPathSeq(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("appendPathSeq"); err != nil {
		return vm.IoError(err)
	}
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	if other.IsMutable() {
		obj.Lock()
		defer obj.Unlock()
	}
	sl, ok := s.At(s.Len() - 1)
	if !ok {
		return vm.NewSequence(other.Value, true, other.Code)
	}
	of, ok := other.At(0)
	if !ok {
		return target
	}
	sis := rune(sl) == filepath.Separator || rune(sl) == '/'
	ois := rune(of) == filepath.Separator || rune(of) == '/'
	if sis && ois {
		s = s.Slice(0, s.Len()-1, 1)
	} else if !sis && !ois {
		s = s.Append(Sequence{Value: []byte(string(filepath.Separator)), Code: "utf8"})
	}
	target.Value = s.Append(other)
	return target
}

// SequenceAsBase64 is a Sequence method.
//
// asBase64 creates a base-64 representation of the bit data of the sequence,
// in accordance with RFC 4648.
func SequenceAsBase64(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	line := 0
	if msg.ArgCount() > 0 {
		n, exc, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			unholdSeq(s.Mutable, target)
			return vm.Stop(exc, stop)
		}
		line = int(n)
	}
	e := base64.StdEncoding.EncodeToString(s.Bytes())
	unholdSeq(s.Mutable, target)
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
// Number encoding is copied directly. If an erroneous encoding is
// encountered, the result will be numeric.
func SequenceAsFixedSizeType(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	code := s.MinCode()
	switch code {
	case "utf8":
		return SequenceAsUTF8(vm, target, locals, msg)
	case "ascii", "latin1":
		return SequenceAsLatin1(vm, target, locals, msg)
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
func SequenceAsIoPath(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewString(filepath.ToSlash(r))
}

// SequenceAsJson is a Sequence method.
//
// asJson creates a JSON representation of the sequence.
func SequenceAsJson(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	var r []byte
	var err error
	if s.Code == "number" {
		// Serialize as an array. We need to avoid the default behavior for
		// []byte, which is to encode as base64.
		if v, ok := s.Value.([]uint8); ok {
			w := make([]uint16, len(v))
			for i, x := range v {
				w[i] = uint16(x)
			}
			r, err = json.Marshal(w)
		} else {
			r, err = json.Marshal(s.Value)
		}
	} else {
		r, err = json.Marshal(s.String())
	}
	unholdSeq(s.Mutable, target)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewSequence(r, false, "utf8")
}

// SequenceAsMessage is a Sequence method.
//
// asMessage compiles the sequence to a Message.
func SequenceAsMessage(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	label := "[asMessage]"
	if msg.ArgCount() > 0 {
		r, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		label = r
	}
	m, err := vm.Parse(strings.NewReader(s.String()), label)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.MessageObject(m)
}

// SequenceAsNumber is a Sequence method.
//
// asNumber parses the sequence as a numeric representation.
func SequenceAsNumber(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	b := strings.TrimSpace(s.String())
	x, err := strconv.ParseFloat(b, 64)
	if err != nil {
		y, err := strconv.ParseInt(b, 0, 64)
		if err != nil {
			unholdSeq(s.Mutable, target)
			return vm.IoError(err)
		}
		x = float64(y)
	}
	unholdSeq(s.Mutable, target)
	return vm.NewNumber(x)
}

// SequenceAsOSPath is a Sequence method.
//
// asOSPath creates a sequence converting the receiver to the host operating
// system's path convention.
func SequenceAsOSPath(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewString(filepath.FromSlash(r))
}

// SequenceCapitalize is a Sequence method.
//
// capitalize replaces the first rune in the sequence with the capitalized
// equivalent. This does not use special (Turkish) casing.
func SequenceCapitalize(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
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
			x := vm.SequenceFromBytes(v, s.Kind())
			s.Value = x.Value
			target.Value = s
		}
	case "ascii", "latin1":
		c, _ := encLatin1.EncodeRune(r)
		v := reflect.ValueOf(s.Value).Index(0)
		// FIXME: This will overwrite if the type isn't 8-bit.
		v.Set(reflect.ValueOf(c).Convert(v.Type()))
	case "utf16":
		e := encUTF16.NewEncoder()
		b, _ = e.Bytes(b)
		v := reflect.ValueOf(s.Value)
		x := reflect.MakeSlice(v.Type(), len(b)/is, len(b)/is)
		binary.Read(bytes.NewReader(b), binary.LittleEndian, x.Interface())
		// FIXME: This will overwrite if the rune is a single code unit and the
		// sequence kind is 32, or if the sequence kind is 64-bit.
		reflect.Copy(v, x)
	case "utf32":
		e := encUTF32.NewEncoder()
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

// SequenceCloneAppendPath is a Sequence method.
//
// cloneAppendPath creates a new Symbol with the receiver's contents and the
// argument sequence's contents appended with redundant path separators
// between them removed.
func SequenceCloneAppendPath(vm *VM, target, locals Interface, msg *Message) *Object {
	other, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(obj, stop)
	}
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	if other.IsMutable() {
		obj.Lock()
		defer obj.Unlock()
	}
	sl, ok := s.At(s.Len() - 1)
	if !ok {
		return vm.NewSequence(other.Value, false, other.Code)
	}
	of, ok := other.At(0)
	if !ok {
		return target
	}
	sis := rune(sl) == filepath.Separator || rune(sl) == '/'
	ois := rune(of) == filepath.Separator || rune(of) == '/'
	v := Sequence{Value: copySeqVal(s.Value), Mutable: true, Code: s.Code}
	if sis && ois {
		v = v.Slice(0, v.Len()-1, 1)
	} else if !sis && !ois {
		v = v.Append(Sequence{Value: []byte(string(filepath.Separator)), Code: "utf8"})
	}
	v = v.Append(other)
	v.Mutable = false
	return vm.SequenceObject(v)
}

// SequenceConvertToFixedSizeType is a Sequence method.
//
// convertToFixedSizeType converts the sequence to be  encoded in the first of
// UTF-8, Latin-1, UTF-16, or UTF-32 that will encode each rune in a single
// word. Number encoding is unchanged. If an erroneous code sequence is
// encountered, the result will be numeric.
func SequenceConvertToFixedSizeType(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("convertToFixedSizeType"); err != nil {
		return vm.IoError(err)
	}
	code := s.MinCode()
	switch code {
	case "utf8":
		if _, ok := s.Value.([]uint8); !ok || s.Code != "utf8" {
			s.Value = []byte(s.String())
			s.Code = "utf8"
		}
	case "ascii", "latin1":
		if _, ok := s.Value.([]uint8); !ok || s.Code != "ascii" && s.Code != "latin1" {
			b := s.String()
			v := make([]byte, 0, len(b))
			for _, r := range b {
				c, _ := encLatin1.EncodeRune(r)
				v = append(v, c)
			}
			s.Value = v
			s.Code = "latin1"
		}
	case "utf16":
		if _, ok := s.Value.([]uint16); !ok || s.Code != "utf16" {
			b := s.String()
			v := make([]uint16, 0, len(b))
			for _, r := range b {
				v = append(v, uint16(r))
			}
			s.Value = v
			s.Code = "utf16"
		}
	case "utf32":
		if _, ok := s.Value.([]rune); !ok || s.Code != "utf32" {
			s.Value = []rune(s.String())
			s.Code = "utf32"
		}
	}
	target.Value = s
	return target
}

// SequenceEscape is a Sequence method.
//
// escape replaces control and non-printable characters with backslash-escaped
// equivalents.
func SequenceEscape(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	if err := s.CheckMutable("escape"); err != nil {
		target.Unlock()
		return vm.IoError(err)
	}
	ss := []byte(strconv.Quote(s.String()))
	ss = ss[1 : len(ss)-1]
	switch s.Value.(type) {
	case []byte:
		s.Value = ss
	case []uint16:
		v := make([]uint16, len(ss))
		for i, x := range ss {
			v[i] = uint16(x)
		}
		s.Value = v
	case []uint32:
		v := make([]uint32, len(ss))
		for i, x := range ss {
			v[i] = uint32(x)
		}
		s.Value = v
	case []uint64:
		v := make([]uint64, len(ss))
		for i, x := range ss {
			v[i] = uint64(x)
		}
		s.Value = v
	case []int8:
		v := make([]int8, len(ss))
		for i, x := range ss {
			v[i] = int8(x)
		}
		s.Value = v
	case []int16:
		v := make([]int16, len(ss))
		for i, x := range ss {
			v[i] = int16(x)
		}
		s.Value = v
	case []int32:
		v := make([]int32, len(ss))
		for i, x := range ss {
			v[i] = int32(x)
		}
		s.Value = v
	case []int64:
		v := make([]int64, len(ss))
		for i, x := range ss {
			v[i] = int64(x)
		}
		s.Value = v
	case []float32:
		v := make([]float32, len(ss))
		for i, x := range ss {
			v[i] = float32(x)
		}
		s.Value = v
	case []float64:
		v := make([]float64, len(ss))
		for i, x := range ss {
			v[i] = float64(x)
		}
		s.Value = v
	}
	target.Value = s
	target.Unlock()
	return target
}

// SequenceFromBase is a Sequence method.
//
// fromBase converts the sequence from a representation of an integer in a
// given radix to the Number it represents.
func SequenceFromBase(vm *VM, target, locals Interface, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	b := int(n)
	s := holdSeq(target)
	sv := strings.TrimSpace(s.String())
	unholdSeq(s.Mutable, target)
	if b == 16 {
		sv = strings.TrimPrefix(sv, "0x")
	}
	x, err := strconv.ParseInt(sv, b, 64)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(x))
}

// SequenceFromBase64 is a Sequence method.
//
// fromBase64 decodes standard (RFC 4648) base64 data from the sequence
// interpreted bytewise.
func SequenceFromBase64(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	v := s.Bytes()
	unholdSeq(s.Mutable, target)
	w := make([]byte, base64.StdEncoding.DecodedLen(len(v)))
	n, err := base64.StdEncoding.Decode(w, v)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewSequence(w[:n], false, "utf8")
}

// SequenceInterpolate is a Sequence method.
//
// interpolate replaces "#{Io code}" in the sequence with the result of
// evaluating the Io code in the current context or the optionally supplied
// one, returning a new sequence with the result.
func SequenceInterpolate(vm *VM, target, locals Interface, msg *Message) *Object {
	// The original implementation is equivalent to
	// method(self asMutable interpolateInPlace asSymbol), but our stronger
	// types actually make it easier to make interpolateInPlace use this.
	ctxt := locals
	if msg.ArgCount() > 0 {
		var stop Stop
		ctxt, stop = msg.EvalArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(ctxt, stop)
		}
	}
	s := holdSeq(target)
	sv := s.String()
	unholdSeq(s.Mutable, target)
	k := 0
	b := strings.Builder{}
	m := vm.IdentMessage("doString", nil)
	m.InsertAfter(vm.IdentMessage("asString"))
	for {
		i := strings.Index(sv[k:], "#{")
		if i < 0 {
			b.WriteString(sv[k:])
			break
		}
		j := strings.Index(sv[k+i+2:], "}")
		if j < 0 {
			b.WriteString(sv[k:])
			break
		}
		b.WriteString(sv[k : k+i])
		code := sv[k+i+2 : k+i+j+2]
		if len(code) != 0 {
			m.Args[0] = vm.StringMessage(code)
			r, stop := m.Eval(vm, ctxt)
			if stop != NoStop {
				return vm.Stop(r, stop)
			}
			if rs, ok := r.Value.(Sequence); ok {
				if rs.IsMutable() {
					r.Lock()
				}
				b.WriteString(rs.String())
				if rs.IsMutable() {
					r.Unlock()
				}
			}
		}
		k += i + j + 3
	}
	return vm.NewString(b.String())
}

// SequenceIsLowercase is a Sequence method.
//
// isLowercase determines whether all characters in the string are equal to
// their lowercase-converted versions using Unicode standard case conversion.
func SequenceIsLowercase(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	sv := s.String()
	unholdSeq(s.Mutable, target)
	for _, r := range sv {
		if r != unichr.ToLower(r) {
			return vm.False
		}
	}
	return vm.True
}

// SequenceIsUppercase is a Sequence method.
//
// isUppercase determines whether all characters in the string are equal to
// their uppercase-converted versions using Unicode standard case conversion.
func SequenceIsUppercase(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	sv := s.String()
	unholdSeq(s.Mutable, target)
	for _, r := range sv {
		if r != unichr.ToUpper(r) {
			return vm.False
		}
	}
	return vm.True
}

// SequenceLastPathComponent is a Sequence method.
//
// lastPathComponent returns the basename of the sequence.
func SequenceLastPathComponent(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	// This is the easy method, but results differ from Io's implementation;
	// the original preserved trailing slashes and made no attempt to interpret
	// the result as an actual path, while this removes trailing slashes,
	// returns "." for the empty string, and returns the system's path
	// separator for all-slash strings.
	return vm.NewString(filepath.Base(r))
}

// SequenceLowercase is a Sequence method.
//
// lowercase converts the values in the sequence to their capitalized
// equivalents. This does not use special (Turkish) casing.
func SequenceLowercase(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	if err := s.CheckMutable("lowercase"); err != nil {
		target.Unlock()
		return vm.IoError(err)
	}
	nv := EncodeString(strings.ToLower(s.String()), s.Code, s.Kind())
	target.Value = nv
	target.Unlock()
	return target
}

// SequenceLstrip is a Sequence method.
//
// lstrip removes all whitespace characters from the beginning of the sequence,
// or all characters in the provided cut set.
func SequenceLstrip(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("lstrip"); err != nil {
		return vm.IoError(err)
	}
	var sv string
	if msg.ArgCount() == 0 {
		sv = strings.TrimLeftFunc(s.String(), unichr.IsSpace)
	} else {
		other, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		sv = strings.TrimLeft(s.String(), other)
	}
	target.Value = EncodeString(sv, s.Code, s.Kind())
	return target
}

// SequenceParseJson is a Sequence method.
//
// parseJson decodes the JSON represented by the receiver.
func SequenceParseJson(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	d := json.NewDecoder(strings.NewReader(s.String()))
	tok, err := d.Token()
	switch err {
	case nil: // do nothing
	case io.EOF:
		return vm.RaiseExceptionf("can't parse empty string")
	default:
		return vm.IoError(err)
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			l := []*Object{}
			l, err = parseJSONList(vm, d, l)
			if err != nil {
				return vm.IoError(err)
			}
			return vm.NewList(l...)
		}
		m := map[string]Interface{}
		err = parseJSONMap(vm, d, m)
		if err != nil {
			return vm.IoError(err)
		}
		return vm.NewMap(m)
	case bool:
		return vm.IoBool(t)
	case float64:
		return vm.NewNumber(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return vm.IoError(err)
		}
		return vm.NewNumber(f)
	case string:
		return vm.NewString(t)
	case nil:
		return vm.Nil
	}
	panic("unreachable")
}

func parseJSONList(vm *VM, d *json.Decoder, v []*Object) ([]*Object, error) {
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return v, err
		}
		switch t := tok.(type) {
		case json.Delim:
			if t == '[' {
				nl := []*Object{}
				nl, err = parseJSONList(vm, d, nl)
				if err != nil {
					return v, err
				}
				v = append(v, vm.NewList(nl...))
			} else {
				// Token guarantees us that delimiters are matched, so we must
				// have {.
				m := map[string]*Object{}
				err = parseJSONMap(vm, d, m)
				if err != nil {
					return v, err
				}
				v = append(v, vm.NewMap(m))
			}
		case bool:
			v = append(v, vm.IoBool(t))
		case float64:
			v = append(v, vm.NewNumber(t))
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				return v, err
			}
			v = append(v, vm.NewNumber(f))
		case string:
			v = append(v, vm.NewString(t))
		case nil:
			v = append(v, vm.Nil)
		}
	}
	// Consume the closing delimiter.
	_, err := d.Token()
	return v, err
}

func parseJSONMap(vm *VM, d *json.Decoder, v map[string]*Object) error {
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		// tok must be the key name.
		k := tok.(string)
		tok, err = d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case json.Delim:
			if t == '[' {
				l := []*Object{}
				l, err = parseJSONList(vm, d, l)
				if err != nil {
					return err
				}
				v[k] = vm.NewList(l...)
			} else {
				nm := map[string]Interface{}
				err = parseJSONMap(vm, d, nm)
				if err != nil {
					return err
				}
				v[k] = vm.NewMap(nm)
			}
		case bool:
			v[k] = vm.IoBool(t)
		case float64:
			v[k] = vm.NewNumber(t)
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				return err
			}
			v[k] = vm.NewNumber(f)
		case string:
			v[k] = vm.NewString(t)
		case nil:
			v[k] = vm.Nil
		}
	}
	// Consume the closing delimiter.
	_, err := d.Token()
	return err
}

// SequencePathComponent is a Sequence method.
//
// pathComponent returns a new Sequence with the receiver up to the last path
// separator. Always converts to slash paths.
func SequencePathComponent(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	// Easy method again. Same differences as lastPathComponent above, but more
	// details also differ: "a/b/c/" pathComponent returns "a/b/c" here, but
	// "a/b" in Io. Should probably fix, but hopefully it's rare.
	return vm.NewString(filepath.ToSlash(filepath.Dir(r)))
}

// SequencePathExtension is a Sequence method.
//
// pathExtension returns a new Sequence with the receiver past the last period.
func SequencePathExtension(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewString(strings.TrimPrefix(filepath.Ext(r), "."))
}

// SequencePercentDecoded is a Sequence method.
//
// percentDecoded unescapes the receiver as a URL path segment.
func SequencePercentDecoded(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	p, err := url.PathUnescape(r)
	if err != nil {
		return vm.NewString("")
	}
	return vm.NewString(p)
}

// SequencePercentEncoded is a Sequence method.
//
// percentEncoded escapes the receiver as a URL path segment.
func SequencePercentEncoded(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewString(url.PathEscape(r))
}

// SequenceRstrip is a Sequence method.
//
// rstrip removes all whitespace characters from the end of the sequence, or
// all characters in the provided cut set.
func SequenceRstrip(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("rstrip"); err != nil {
		return vm.IoError(err)
	}
	var sv string
	if msg.ArgCount() == 0 {
		sv = strings.TrimRightFunc(s.String(), unichr.IsSpace)
	} else {
		other, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		sv = strings.TrimRight(s.String(), other)
	}
	target.Value = EncodeString(sv, s.Code, s.Kind())
	return target
}

// SequenceSplit is a Sequence method.
//
// split returns a list of the portions of the sequence split at each
// occurrence of any of the given separators, or by whitespace if none given.
func SequenceSplit(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	str := s.String()
	unholdSeq(s.Mutable, target)
	l := []Interface{}
	if msg.ArgCount() == 0 {
		// Split at whitespace.
		v := strings.Fields(str)
		for _, x := range v {
			l = append(l, vm.NewString(x))
		}
	} else {
		seps := make([]string, msg.ArgCount())
		for arg := range seps {
			sep, exc, stop := msg.StringArgAt(vm, locals, arg)
			if stop != NoStop {
				return vm.Stop(exc, stop)
			}
			seps[arg] = sep
		}
		v := strings.Builder{}
		ign := 0
	stringloop:
		for k, r := range str {
			if ign > 0 {
				ign--
				continue
			}
			for _, sep := range seps {
				if strings.HasPrefix(str[k:], sep) {
					l = append(l, vm.NewString(v.String()))
					ign = utf8.RuneCountInString(sep) - 1
					v.Reset()
					continue stringloop
				}
			}
			v.WriteRune(r)
		}
		l = append(l, vm.NewString(v.String()))
	}
	return vm.NewList(l...)
}

// SequenceStrip is a Sequence method.
//
// strip removes all whitespace characters from each end of the sequence, or
// all characters in the provided cut set.
func SequenceStrip(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("strip"); err != nil {
		return vm.IoError(err)
	}
	var sv string
	if msg.ArgCount() == 0 {
		sv = strings.TrimFunc(s.String(), unichr.IsSpace)
	} else {
		other, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		sv = strings.Trim(s.String(), other)
	}
	target.Value = EncodeString(sv, s.Code, s.Kind())
	return target
}

// SequenceToBase is a Sequence method.
//
// toBase converts the sequence from a base 10 representation of a number to a
// base 8 or 16 representation of the same number.
func SequenceToBase(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	defer unholdSeq(s.Mutable, target)
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	base := int(n)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("cannot convert to base %d", base)
	}
	x, err := strconv.ParseInt(s.String(), 10, 64)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewString(strconv.FormatInt(x, base))
}

// SequenceUnescape is a Sequence method.
//
// unescape interprets backslash-escaped codes in the sequence.
func SequenceUnescape(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("unescape"); err != nil {
		return vm.IoError(err)
	}
	ss, err := strconv.Unquote(`"` + s.String() + `"`)
	if err != nil {
		return vm.IoError(err)
	}
	target.Value = EncodeString(ss, s.Code, s.Kind())
	return target
}

// SequenceUppercase is a Sequence method.
//
// uppercase converts the values in the sequence to their capitalized
// equivalents. This does not use special (Turkish) casing.
func SequenceUppercase(vm *VM, target, locals Interface, msg *Message) *Object {
	s := lockSeq(target)
	defer target.Unlock()
	if err := s.CheckMutable("uppercase"); err != nil {
		return vm.IoError(err)
	}
	r := s.String()
	target.Value = EncodeString(r, s.Code, s.Kind())
	return target
}

// SequenceUrlDecoded is a Sequence method.
//
// urlDecoded unescapes the sequence as a URL query.
func SequenceUrlDecoded(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	p, err := url.QueryUnescape(r)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewString(p)
}

// SequenceUrlEncoded is a Sequence method.
//
// urlEncoded escapes the sequence for safe use in a URL query.
func SequenceUrlEncoded(vm *VM, target, locals Interface, msg *Message) *Object {
	s := holdSeq(target)
	r := s.String()
	unholdSeq(s.Mutable, target)
	return vm.NewString(url.QueryEscape(r))
}
