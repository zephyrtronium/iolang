//go:generate mknumbermemo

package iolang

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Number is the primary numeric type. These should be considered immutable.
type Number struct {
	Object
	Value float64
}

// NewNumber creates a Number object with a given value. If the value is
// memoized by the VM, that object is returned; otherwise, a new object will be
// allocated.
func (vm *VM) NewNumber(value float64) *Number {
	if x := vm.chkIntsMemo(value); x != nil {
		return x
	}
	if x := vm.chkRealsMemo(value); x != nil {
		return x
	}
	return &Number{
		Object: *vm.CoreInstance("Number"),
		Value:  value,
	}
}

// NumberArgAt evaluates the nth argument and returns it as a Number. If a
// stop occurs during evaluation, the Number will be nil, and the stop status
// and result will be returned. If the evaluated result is not a Number, the
// result will be nil and an exception will be raised.
func (m *Message) NumberArgAt(vm *VM, locals Interface, n int) (*Number, Interface, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		if num, ok := v.(*Number); ok {
			return num, nil, NoStop
		}
		// Not the expected type, so return an error.
		v, s = vm.RaiseExceptionf("argument %d to %s must be Number, not %s", n, m.Text, vm.TypeName(v))
	}
	return nil, v, s
}

// Activate returns the number.
func (n *Number) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return n, NoStop
}

// Clone creates a clone of this Number with the same value.
func (n *Number) Clone() Interface {
	return &Number{
		Object{Slots: Slots{}, Protos: []Interface{n}},
		n.Value,
	}
}

// String creates a string representation of this number.
func (n *Number) String() string {
	return strconv.FormatFloat(n.Value, 'g', -1, 64)
}

// initNumber initializes Number on this VM.
func (vm *VM) initNumber() {
	var kind *Number
	slots := Slots{
		"*":                  vm.NewCFunction(NumberMul, kind),
		"+":                  vm.NewCFunction(NumberAdd, kind),
		"-":                  vm.NewCFunction(NumberSub, kind),
		"/":                  vm.NewCFunction(NumberDiv, kind),
		"abs":                vm.NewCFunction(NumberAbs, kind),
		"acos":               vm.NewCFunction(NumberAcos, kind),
		"asBuffer":           vm.NewCFunction(NumberAsBuffer, kind),
		"asCharacter":        vm.NewCFunction(NumberAsCharacter, kind),
		"asLowercase":        vm.NewCFunction(NumberAsLowercase, kind),
		"asNumber":           vm.NewCFunction(ObjectThisContext, kind), // hax
		"asString":           vm.NewCFunction(NumberAsString, kind),
		"asUint32Buffer":     vm.NewCFunction(NumberAsUint32Buffer, kind),
		"asUppercase":        vm.NewCFunction(NumberAsUppercase, kind),
		"asin":               vm.NewCFunction(NumberAsin, kind),
		"at":                 vm.NewCFunction(NumberAt, kind),
		"atan":               vm.NewCFunction(NumberAtan, kind),
		"atan2":              vm.NewCFunction(NumberAtan2, kind),
		"between":            vm.NewCFunction(NumberBetween, kind),
		"bitwiseAnd":         vm.NewCFunction(NumberBitwiseAnd, kind),
		"bitwiseComplement":  vm.NewCFunction(NumberBitwiseComplement, kind),
		"bitwiseOr":          vm.NewCFunction(NumberBitwiseOr, kind),
		"bitwiseXor":         vm.NewCFunction(NumberBitwiseXor, kind),
		"ceil":               vm.NewCFunction(NumberCeil, kind),
		"clip":               vm.NewCFunction(NumberClip, kind),
		"compare":            vm.NewCFunction(NumberCompare, kind),
		"cos":                vm.NewCFunction(NumberCos, kind),
		"cubed":              vm.NewCFunction(NumberCubed, kind),
		"exp":                vm.NewCFunction(NumberExp, kind),
		"factorial":          vm.NewCFunction(NumberFactorial, kind),
		"floor":              vm.NewCFunction(NumberFloor, kind),
		"isAlphaNumeric":     vm.NewCFunction(NumberIsAlphaNumeric, kind),
		"isControlCharacter": vm.NewCFunction(NumberIsControlCharacter, kind),
		"isDigit":            vm.NewCFunction(NumberIsDigit, kind),
		"isEven":             vm.NewCFunction(NumberIsEven, kind),
		"isHexDigit":         vm.NewCFunction(NumberIsHexDigit, kind),
		"isLetter":           vm.NewCFunction(NumberIsLetter, kind),
		"isLowercase":        vm.NewCFunction(NumberIsLowercase, kind),
		"isNan":              vm.NewCFunction(NumberIsNan, kind),
		"isOdd":              vm.NewCFunction(NumberIsOdd, kind),
		"isPrint":            vm.NewCFunction(NumberIsPrint, kind),
		"isPunctuation":      vm.NewCFunction(NumberIsPunctuation, kind),
		"isSpace":            vm.NewCFunction(NumberIsSpace, kind),
		"isUppercase":        vm.NewCFunction(NumberIsUppercase, kind),
		"log":                vm.NewCFunction(NumberLog, kind),
		"log10":              vm.NewCFunction(NumberLog10, kind),
		"log2":               vm.NewCFunction(NumberLog2, kind),
		"max":                vm.NewCFunction(NumberMax, kind),
		"min":                vm.NewCFunction(NumberMin, kind),
		"mod":                vm.NewCFunction(NumberMod, kind),
		"negate":             vm.NewCFunction(NumberNegate, kind),
		"pow":                vm.NewCFunction(NumberPow, kind),
		"repeat":             vm.NewCFunction(NumberRepeat, kind),
		"round":              vm.NewCFunction(NumberRound, kind),
		"roundDown":          vm.NewCFunction(NumberRoundDown, kind),
		"shiftLeft":          vm.NewCFunction(NumberShiftLeft, kind),
		"shiftRight":         vm.NewCFunction(NumberShiftRight, kind),
		"sin":                vm.NewCFunction(NumberSin, kind),
		"sqrt":               vm.NewCFunction(NumberSqrt, kind),
		"squared":            vm.NewCFunction(NumberSquared, kind),
		"tan":                vm.NewCFunction(NumberTan, kind),
		"toBase":             vm.NewCFunction(NumberToBase, kind),
		"toBaseWholeBytes":   vm.NewCFunction(NumberToBaseWholeBytes, kind),
		"toggle":             vm.NewCFunction(NumberToggle, kind),
		"type":               vm.NewString("Number"),
	}
	vm.SetSlot(vm.Core, "Number", &Number{Object: *vm.ObjectWith(slots)})

	vm.initIntsMemo()
	vm.initRealsMemo()
	slots["%"] = slots["mod"]
	slots["&"] = slots["bitwiseAnd"]
	slots["|"] = slots["bitwiseOr"]
	slots["^"] = slots["bitwiseXor"]
	slots["**"] = slots["pow"]
	slots["<<"] = slots["shiftLeft"]
	slots[">>"] = slots["shiftRight"]
	slots["asJson"] = slots["asString"]
	slots["asSimpleString"] = slots["asString"]
	slots["minMax"] = slots["clip"]
	slots["floatMin"] = vm.NewNumber(math.SmallestNonzeroFloat64)
	slots["floatMax"] = vm.NewNumber(math.MaxFloat64)
	slots["integerMin"] = vm.NewNumber(math.MinInt64)
	slots["integerMax"] = vm.NewNumber(math.MaxInt64)
	slots["longMin"] = vm.NewNumber(math.MinInt64)
	slots["longMax"] = vm.NewNumber(math.MaxInt64)
	slots["shortMin"] = vm.NewNumber(-32768)
	slots["shortMax"] = vm.NewNumber(32767)
	slots["unsignedIntMax"] = vm.NewNumber(math.MaxUint64)
	slots["unsignedLongMax"] = slots["unsignedIntMax"]
	slots["constants"] = vm.ObjectWith(Slots{
		// Io originally had only e, inf, nan, and pi.
		"e":       vm.NewNumber(math.E),
		"pi":      vm.NewNumber(math.Pi),
		"phi":     vm.NewNumber(math.Phi),
		"sqrt2":   vm.NewNumber(math.Sqrt2),
		"sqrtE":   vm.NewNumber(math.SqrtE),
		"sqrtPi":  vm.NewNumber(math.SqrtPi),
		"sqrtPhi": vm.NewNumber(math.SqrtPhi),
		"ln2":     vm.NewNumber(math.Ln2),
		"log2E":   vm.NewNumber(math.Log2E),
		"ln10":    vm.NewNumber(math.Ln10),
		"log10E":  vm.NewNumber(math.Log10E),
		"inf":     vm.NewNumber(math.Inf(1)),
		"nan":     vm.NewNumber(math.NaN()),
	})
}

// NumberAbs is a Number method.
//
// abs returns the absolute value of the target.
func NumberAbs(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n := target.(*Number)
	if n.Value < 0 {
		return vm.NewNumber(-n.Value), NoStop
	}
	return target, NoStop
}

// NumberAcos is a Number method.
//
// acos returns the arccosine of the target.
func NumberAcos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Acos(target.(*Number).Value)), NoStop
}

// NumberAdd is a Number method.
//
// + is an operator which sums two numbers.
func NumberAdd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(target.(*Number).Value + arg.Value), NoStop
}

// NumberAsBuffer is a Number method.
//
// asBuffer creates a Latin-1 Sequence with bytes equal to the binary
// representation of the target. An optional byte count for the size of the
// buffer may be supplied, with a default of 8.
func NumberAsBuffer(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n := 8
	if msg.ArgCount() > 0 {
		arg, err, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		n = int(arg.Value)
		if n < 0 {
			return vm.RaiseException("buffer size must be nonnegative")
		}
	}
	x := target.(*Number).Value
	var v []byte
	if n >= 8 {
		v = make([]byte, n)
	} else {
		v = []byte{7: 0}
	}
	binary.LittleEndian.PutUint64(v, math.Float64bits(x))
	return vm.NewSequence(v[:n], true, "latin1"), NoStop
}

// NumberAsCharacter is a Number method.
//
// asCharacter returns a string containing the Unicode character with the
// codepoint corresponding to the integer value of the target.
func NumberAsCharacter(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(string(rune(target.(*Number).Value))), NoStop
}

// NumberAsLowercase is a Number method.
//
// asLowercase returns the Number which is the Unicode codepoint corresponding
// to the lowercase version of the target as a Unicode codepoint.
func NumberAsLowercase(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(unicode.ToLower(rune(target.(*Number).Value)))), NoStop
}

// NumberAsString is a Number method.
//
// asString returns the decimal string representation of the target.
func NumberAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(target.(*Number).String()), NoStop
}

// NumberAsUint32Buffer is a Number method.
//
// asUint32Buffer returns a 4-byte buffer representing the target's value
// converted to a uint32.
func NumberAsUint32Buffer(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := uint32(target.(*Number).Value)
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, x)
	return vm.NewSequence(v, true, "latin1"), NoStop
}

// NumberAsUppercase is a Number method.
//
// asUppercase returns the Number which is the Unicode codepoint corresponding
// to the uppercase version of the target as a Unicode codepoint.
func NumberAsUppercase(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(unicode.ToUpper(rune(target.(*Number).Value)))), NoStop
}

// NumberAsin is a Number method.
//
// asin returns the arcsine of the target.
func NumberAsin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Asin(target.(*Number).Value)), NoStop
}

// NumberAt is a Number method.
//
// at returns 1 if the argument has the nth bit of its integer representation
// set and 0 otherwise.
//
//   io> 3 at(1)
//   1
//   io> 3 at(2)
//   0
func NumberAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) >> uint64(arg.Value) & 1)), NoStop
}

// NumberAtan is a Number method.
//
// atan returns the arctangent of the target.
func NumberAtan(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Atan(target.(*Number).Value)), NoStop
}

// NumberAtan2 is a Number method.
//
// atan2 returns the directional arctangent of the target divided by the
// argument.
func NumberAtan2(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(math.Atan2(target.(*Number).Value, arg.Value)), NoStop
}

// NumberBetween is a Number method.
//
// between is true if the target is greater than or equal to the first argument
// and less than or equal to the second.
func NumberBetween(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg1, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	arg2, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	v := target.(*Number).Value
	return vm.IoBool(arg1.Value <= v && v <= arg2.Value), NoStop
}

// NumberBitwiseAnd is a Number method.
//
// bitwiseAnd returns the bitwise intersection of the target and the argument,
// with each converted to 64-bit integers.
func NumberBitwiseAnd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) & int64(arg.Value))), NoStop
}

// NumberBitwiseComplement is a Number method.
//
// bitwiseComplement returns the bitwise complement of the 64-bit integer value
// of the target.
func NumberBitwiseComplement(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(^int64(target.(*Number).Value))), NoStop
}

// NumberBitwiseOr is a Number method.
//
// bitwiseOr returns the bitwise union of the target and the argument, with
// each converted to 64-bit integers.
func NumberBitwiseOr(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) | int64(arg.Value))), NoStop
}

// NumberBitwiseXor is a Number method.
//
// bitwiseXor returns the bitwise symmetric difference of the target and the
// argument, with each converted to 64-bit integers.
func NumberBitwiseXor(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) ^ int64(arg.Value))), NoStop
}

// NumberCeil is a Number method.
//
// ceil returns the smallest integer larger than or equal to the target.
func NumberCeil(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Ceil(target.(*Number).Value)), NoStop
}

// NumberClip is a Number method.
//
// clip returns the target if it is between the given bounds or else the
// exceeded bound.
func NumberClip(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg1, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	arg2, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	v := target.(*Number).Value
	if v > arg2.Value {
		return arg2, NoStop
	}
	if v < arg1.Value {
		return arg1, NoStop
	}
	return target, NoStop
}

// NumberCompare is a Number method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func NumberCompare(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// Io doesn't actually define a Number compare, but I'm doing it anyway.
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	a, b := target.(*Number).Value, arg.Value
	if a < b {
		return vm.NewNumber(-1), NoStop
	}
	if a > b {
		return vm.NewNumber(1), NoStop
	}
	return vm.NewNumber(0), NoStop
}

// NumberCos is a Number method.
//
// cos returns the cosine of the target.
func NumberCos(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Cos(target.(*Number).Value)), NoStop
}

// NumberCubed is a Number method.
//
// cubed returns the target raised to the third power.
func NumberCubed(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := target.(*Number).Value
	return vm.NewNumber(x * x * x), NoStop
}

// NumberDiv is a Number method.
//
// / is an operator which divides the left value by the right.
func NumberDiv(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(target.(*Number).Value / arg.Value), NoStop
}

// NumberExp is a Number method.
//
// exp returns e (the base of the natural logarithm) raised to the power of
// the target.
func NumberExp(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Exp(target.(*Number).Value)), NoStop
}

// NumberFactorial is a Number method.
//
// factorial computes the product of each integer between 1 and the target.
func NumberFactorial(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := int64(target.(*Number).Value)
	if x < 0 {
		return vm.NewNumber(math.NaN()), NoStop
	}
	v := 1.0
	for x > 0 {
		v *= float64(x)
		x--
	}
	return vm.NewNumber(v), NoStop
}

// NumberFloor is a Number method.
//
// floor returns the largest integer smaller than or equal to the target.
func NumberFloor(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Floor(target.(*Number).Value)), NoStop
}

// NumberIsAlphaNumeric is a Number method.
//
// isAlphaNumeric is true if the target is a Unicode codepoint corresponding to
// a letter (category L) or number (category N).
func NumberIsAlphaNumeric(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := rune(target.(*Number).Value)
	return vm.IoBool(unicode.IsLetter(x) || unicode.IsNumber(x)), NoStop
}

// NumberIsControlCharacter is a Number method.
//
// isControlCharacter is true if the target is a Unicode codepoint
// corresponding to a control character.
func NumberIsControlCharacter(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsControl(rune(target.(*Number).Value))), NoStop
}

// NumberIsDigit is a Number method.
//
// isDigit is true if the target is a Unicode codepoint corresponding to a
// decimal digit.
func NumberIsDigit(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsDigit(rune(target.(*Number).Value))), NoStop
}

// NumberIsEven is a Number method.
//
// isEven is true if the integer value of the target is divisible by 2.
func NumberIsEven(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(int64(target.(*Number).Value)&1 == 0), NoStop
}

// NumberIsGraph is a Number method.
//
// isGraph is true if the target is a Unicode codepoint corresponding to a
// graphic character (categories L, M, N, P, S, Zs).
func NumberIsGraph(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsGraphic(rune(target.(*Number).Value))), NoStop
}

// NumberIsHexDigit is a Number method.
//
// isHexDigit is true if the target is a Unicode codepoint corresponding to the
// characters 0 through 9, A through F, or a through f.
func NumberIsHexDigit(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := rune(target.(*Number).Value)
	return vm.IoBool(('0' <= x && x <= '9') || ('A' <= x && x <= 'F') || ('a' <= x && x <= 'f')), NoStop
}

// NumberIsLetter is a Number method.
//
// isLetter is true if the target is a Unicode codepoint corresponding to a
// letter (category L).
func NumberIsLetter(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsLetter(rune(target.(*Number).Value))), NoStop
}

// NumberIsLowercase is a Number method.
//
// isLowercase is true if the target is a Unicode codepoint corresponding to a
// lowercase letter (category Ll).
func NumberIsLowercase(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsLower(rune(target.(*Number).Value))), NoStop
}

// NumberIsNan is a Number method.
//
// isNan is true if the target is an IEEE-754 Not a Number.
func NumberIsNan(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(math.IsNaN(target.(*Number).Value)), NoStop
}

// NumberIsOdd is a Number method.
//
// isOdd is true if the integer value of the target is not divisible by 2.
func NumberIsOdd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(int64(target.(*Number).Value)&1 == 1), NoStop
}

// NumberIsPrint is a Number method.
//
// isPrint is true if the target is a Unicode codepoint corresponding to a
// printable character (categories L, M, N, P, S, and ASCII space, U+0020).
func NumberIsPrint(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsPrint(rune(target.(*Number).Value))), NoStop
}

// NumberIsPunctuation is a Number method.
//
// isPunctuation is true if the target is a Unicode codepoint corresponding to
// a punctuation character (category P).
func NumberIsPunctuation(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsPunct(rune(target.(*Number).Value))), NoStop
}

// NumberIsSpace is a Number method.
//
// isSpace is true if the target is a Unicode codepoint corresponding to a
// space character.
func NumberIsSpace(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsSpace(rune(target.(*Number).Value))), NoStop
}

// NumberIsUppercase is a Number method.
//
// isUppercase is true if the target is a Unicode codepoint corresponding to an
// uppercase letter (category Lu).
func NumberIsUppercase(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(unicode.IsUpper(rune(target.(*Number).Value))), NoStop
}

// NumberLog is a Number method.
//
// log returns the natural logarithm of the target.
func NumberLog(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	if msg.ArgCount() > 0 {
		b, err, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		return vm.NewNumber(math.Log(target.(*Number).Value) / math.Log(b.Value)), NoStop
	}
	return vm.NewNumber(math.Log(target.(*Number).Value)), NoStop
}

// NumberLog10 is a Number method.
//
// log10 returns the base-10 logarithm of the target.
func NumberLog10(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Log10(target.(*Number).Value)), NoStop
}

// NumberLog2 is a Number method.
//
// log2 returns the base-2 logarithm of the target.
func NumberLog2(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Log2(target.(*Number).Value)), NoStop
}

// NumberMax is a Number method.
//
// max returns the larger of the target and the argument.
func NumberMax(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if target.(*Number).Value >= arg.Value {
		return target, NoStop
	}
	return arg, NoStop
}

// NumberMin is a Number method.
//
// min returns the smaller of the target and the argument.
func NumberMin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if target.(*Number).Value <= arg.Value {
		return target, NoStop
	}
	return arg, NoStop
}

// NumberMod is a Number method.
//
// mod returns the remainder of division of the target by the argument.
func NumberMod(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(math.Remainder(target.(*Number).Value, arg.Value)), NoStop
}

// NumberNegate is a Number method.
//
// negate returns the opposite of the target.
func NumberNegate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(-target.(*Number).Value), NoStop
}

// NumberMul is a Number method.
//
// * is an operator which multiplies its operands.
func NumberMul(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(target.(*Number).Value * arg.Value), NoStop
}

// NumberPow is a Number method.
//
// pow returns the target raised to the power of the argument. The ** operator
// is equivalent.
func NumberPow(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(math.Pow(target.(*Number).Value, arg.Value)), NoStop
}

// NumberRepeat is a Number method.
//
// repeat performs a loop the given number of times.
func NumberRepeat(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	if len(msg.Args) < 1 {
		return vm.RaiseException("Number repeat requires 1 or 2 arguments")
	}
	counter, eval := msg.ArgAt(0), msg.ArgAt(1)
	c := counter.Name()
	if eval == nil {
		// One argument was supplied.
		counter, eval = nil, counter
	}
	max := int(math.Ceil(target.(*Number).Value))
	for i := 0; i < max; i++ {
		if counter != nil {
			vm.SetSlot(locals, c, vm.NewNumber(float64(i)))
		}
		result, control = eval.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, control
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
	return result, NoStop
}

// NumberRound is a Number method.
//
// round returns the integer nearest the target, with halfway cases rounding
// away from zero.
func NumberRound(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := target.(*Number).Value
	if x < 0 {
		x = math.Ceil(x - 0.5)
	} else {
		x = math.Floor(x + 0.5)
	}
	return vm.NewNumber(x), NoStop
}

// NumberRoundDown is a Number method.
//
// roundDown returns the integer nearest the target, with halfway cases
// rounding toward positive infinity.
func NumberRoundDown(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Floor(target.(*Number).Value + 0.5)), NoStop
}

// NumberShiftLeft is a Number method.
//
// shiftLeft returns the target as a 64-bit integer shifted left by the
// argument as a 64-bit unsigned integer.
func NumberShiftLeft(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) << uint64(arg.Value))), NoStop
}

// NumberShiftRight is a Number method.
//
// shiftRight returns the target as a 64-bit integer shifted right by the
// argument as a 64-bit unsigned integer.
func NumberShiftRight(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) >> uint64(arg.Value))), NoStop
}

// NumberSin is a Number method.
//
// sin returns the sine of the target.
func NumberSin(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Sin(target.(*Number).Value)), NoStop
}

// NumberSqrt is a Number method.
//
// sqrt returns the square root of the target.
func NumberSqrt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Sqrt(target.(*Number).Value)), NoStop
}

// NumberSquared is a Number method.
//
// squared returns the target raised to the second power.
func NumberSquared(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x := target.(*Number).Value
	return vm.NewNumber(x * x), NoStop
}

// NumberSub is a Number method.
//
// - is an operator which subtracts the right value from the left.
func NumberSub(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.NewNumber(target.(*Number).Value - arg.Value), NoStop
}

// NumberTan is a Number method.
//
// tan returns the tangent of the target.
func NumberTan(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(math.Tan(target.(*Number).Value)), NoStop
}

// NumberToBase is a Number method.
//
// toBase returns the string representation of the target in the radix given in
// the argument. Bases less than 2 and greater than 36 are not supported.
func NumberToBase(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	base := int(arg.Value)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), base)), NoStop
}

var toBaseWholeBytesCols = [...]int8{
	8, 6, 4, 4, 4, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
}

// NumberToBaseWholeBytes is a Number method.
//
// toBaseWholeBytes returns the string representation of the target in the
// radix given in the argument, zero-padded on the left to a multiple of the
// equivalent of eight bits. Bases less than 2 and greater than 36 are not
// supported.
func NumberToBaseWholeBytes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	arg, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	base := int(arg.Value)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	cols := int(toBaseWholeBytesCols[base-2])
	s := strconv.FormatInt(int64(target.(*Number).Value), base)
	w := (len(s) + cols - 1) / cols
	return vm.NewString(strings.Repeat("0", w*cols-len(s)) + s), NoStop
}

// NumberToggle is a Number method.
//
// toggle returns 1 if the target is 0, otherwise 0.
func NumberToggle(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	if target.(*Number).Value == 0 {
		return vm.NewNumber(1), NoStop
	}
	return vm.NewNumber(0), NoStop
}
