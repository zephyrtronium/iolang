package iolang

import (
	"encoding/binary"
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
	if x, ok := vm.NumberMemo[value]; ok {
		return x
	}
	return &Number{
		*vm.CoreInstance("Number"),
		value,
	}
}

// Activate returns the number.
func (n *Number) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return n
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
	var exemplar *Number
	slots := Slots{
		"*":                  vm.NewTypedCFunction(NumberMul, exemplar),
		"+":                  vm.NewTypedCFunction(NumberAdd, exemplar),
		"-":                  vm.NewTypedCFunction(NumberSub, exemplar),
		"/":                  vm.NewTypedCFunction(NumberDiv, exemplar),
		"abs":                vm.NewTypedCFunction(NumberAbs, exemplar),
		"acos":               vm.NewTypedCFunction(NumberAcos, exemplar),
		"asBuffer":           vm.NewTypedCFunction(NumberAsBuffer, exemplar),
		"asCharacter":        vm.NewTypedCFunction(NumberAsCharacter, exemplar),
		"asLowercase":        vm.NewTypedCFunction(NumberAsLowercase, exemplar),
		"asNumber":           vm.NewTypedCFunction(ObjectThisContext, exemplar), // hax
		"asString":           vm.NewTypedCFunction(NumberAsString, exemplar),
		"asUint32Buffer":     vm.NewTypedCFunction(NumberAsUint32Buffer, exemplar),
		"asUppercase":        vm.NewTypedCFunction(NumberAsUppercase, exemplar),
		"asin":               vm.NewTypedCFunction(NumberAsin, exemplar),
		"at":                 vm.NewTypedCFunction(NumberAt, exemplar),
		"atan":               vm.NewTypedCFunction(NumberAtan, exemplar),
		"atan2":              vm.NewTypedCFunction(NumberAtan2, exemplar),
		"between":            vm.NewTypedCFunction(NumberBetween, exemplar),
		"bitwiseAnd":         vm.NewTypedCFunction(NumberBitwiseAnd, exemplar),
		"bitwiseComplement":  vm.NewTypedCFunction(NumberBitwiseComplement, exemplar),
		"bitwiseOr":          vm.NewTypedCFunction(NumberBitwiseOr, exemplar),
		"bitwiseXor":         vm.NewTypedCFunction(NumberBitwiseXor, exemplar),
		"ceil":               vm.NewTypedCFunction(NumberCeil, exemplar),
		"clip":               vm.NewTypedCFunction(NumberClip, exemplar),
		"compare":            vm.NewTypedCFunction(NumberCompare, exemplar),
		"cos":                vm.NewTypedCFunction(NumberCos, exemplar),
		"cubed":              vm.NewTypedCFunction(NumberCubed, exemplar),
		"exp":                vm.NewTypedCFunction(NumberExp, exemplar),
		"factorial":          vm.NewTypedCFunction(NumberFactorial, exemplar),
		"floor":              vm.NewTypedCFunction(NumberFloor, exemplar),
		"isAlphaNumeric":     vm.NewTypedCFunction(NumberIsAlphaNumeric, exemplar),
		"isControlCharacter": vm.NewTypedCFunction(NumberIsControlCharacter, exemplar),
		"isDigit":            vm.NewTypedCFunction(NumberIsDigit, exemplar),
		"isEven":             vm.NewTypedCFunction(NumberIsEven, exemplar),
		"isHexDigit":         vm.NewTypedCFunction(NumberIsHexDigit, exemplar),
		"isLetter":           vm.NewTypedCFunction(NumberIsLetter, exemplar),
		"isLowercase":        vm.NewTypedCFunction(NumberIsLowercase, exemplar),
		"isNan":              vm.NewTypedCFunction(NumberIsNan, exemplar),
		"isOdd":              vm.NewTypedCFunction(NumberIsOdd, exemplar),
		"isPrint":            vm.NewTypedCFunction(NumberIsPrint, exemplar),
		"isPunctuation":      vm.NewTypedCFunction(NumberIsPunctuation, exemplar),
		"isSpace":            vm.NewTypedCFunction(NumberIsSpace, exemplar),
		"isUppercase":        vm.NewTypedCFunction(NumberIsUppercase, exemplar),
		"log":                vm.NewTypedCFunction(NumberLog, exemplar),
		"log10":              vm.NewTypedCFunction(NumberLog10, exemplar),
		"log2":               vm.NewTypedCFunction(NumberLog2, exemplar),
		"max":                vm.NewTypedCFunction(NumberMax, exemplar),
		"min":                vm.NewTypedCFunction(NumberMin, exemplar),
		"mod":                vm.NewTypedCFunction(NumberMod, exemplar),
		"negate":             vm.NewTypedCFunction(NumberNegate, exemplar),
		"pow":                vm.NewTypedCFunction(NumberPow, exemplar),
		"repeat":             vm.NewTypedCFunction(NumberRepeat, exemplar),
		"round":              vm.NewTypedCFunction(NumberRound, exemplar),
		"roundDown":          vm.NewTypedCFunction(NumberRoundDown, exemplar),
		"shiftLeft":          vm.NewTypedCFunction(NumberShiftLeft, exemplar),
		"shiftRight":         vm.NewTypedCFunction(NumberShiftRight, exemplar),
		"sin":                vm.NewTypedCFunction(NumberSin, exemplar),
		"sqrt":               vm.NewTypedCFunction(NumberSqrt, exemplar),
		"squared":            vm.NewTypedCFunction(NumberSquared, exemplar),
		"tan":                vm.NewTypedCFunction(NumberTan, exemplar),
		"toBase":             vm.NewTypedCFunction(NumberToBase, exemplar),
		"toBaseWholeBytes":   vm.NewTypedCFunction(NumberToBaseWholeBytes, exemplar),
		"toggle":             vm.NewTypedCFunction(NumberToggle, exemplar),
		"type":               vm.NewString("Number"),
	}
	SetSlot(vm.Core, "Number", &Number{Object: *vm.ObjectWith(slots)})

	for i := -1; i <= 255; i++ {
		vm.MemoizeNumber(float64(i))
	}
	vm.MemoizeNumber(0.5)
	vm.MemoizeNumber(0.33333333333333333)
	vm.MemoizeNumber(0.25)
	vm.MemoizeNumber(math.E)
	vm.MemoizeNumber(math.Pi)
	vm.MemoizeNumber(math.Phi)
	vm.MemoizeNumber(math.Sqrt2)
	vm.MemoizeNumber(math.SqrtE)
	vm.MemoizeNumber(math.SqrtPi)
	vm.MemoizeNumber(math.SqrtPhi)
	vm.MemoizeNumber(math.Ln2)
	vm.MemoizeNumber(math.Log2E)
	vm.MemoizeNumber(math.Ln10)
	vm.MemoizeNumber(math.Log10E)
	vm.MemoizeNumber(math.SmallestNonzeroFloat64)
	vm.MemoizeNumber(math.MaxFloat64)
	vm.MemoizeNumber(math.MinInt64)
	vm.MemoizeNumber(math.MaxInt64)
	vm.MemoizeNumber(math.Inf(1))
	vm.MemoizeNumber(math.Inf(-1))
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
	slots["floatMin"] = vm.NumberMemo[math.SmallestNonzeroFloat64]
	slots["floatMax"] = vm.NumberMemo[math.MaxFloat64]
	slots["integerMin"] = vm.NumberMemo[math.MinInt64]
	slots["integerMax"] = vm.NumberMemo[math.MaxInt64]
	slots["longMin"] = vm.NumberMemo[math.MinInt64]
	slots["longMax"] = vm.NumberMemo[math.MaxInt64]
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
func NumberAbs(vm *VM, target, locals Interface, msg *Message) Interface {
	n := target.(*Number)
	if n.Value < 0 {
		return vm.NewNumber(-n.Value)
	}
	return target
}

// NumberAcos is a Number method.
//
// acos returns the arccosine of the target.
func NumberAcos(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Acos(target.(*Number).Value))
}

// NumberAdd is a Number method.
//
// + is an operator which sums two numbers.
func NumberAdd(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(target.(*Number).Value + arg.Value)
}

// NumberAsBuffer is a Number method.
//
// asBuffer creates a Latin-1 Sequence with bytes equal to the binary
// representation of the target. An optional byte count for the size of the
// buffer may be supplied, with a default of 8.
func NumberAsBuffer(vm *VM, target, locals Interface, msg *Message) Interface {
	n := 8
	if msg.ArgCount() > 0 {
		arg, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != nil {
			return stop
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
	return vm.NewSequence(v[:n], true, "latin1")
}

// NumberAsCharacter is a Number method.
//
// asCharacter returns a string containing the Unicode character with the
// codepoint corresponding to the integer value of the target.
func NumberAsCharacter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(string(rune(target.(*Number).Value)))
}

// NumberAsLowercase is a Number method.
//
// asLowercase returns the Number which is the Unicode codepoint corresponding
// to the lowercase version of the target as a Unicode codepoint.
func NumberAsLowercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(unicode.ToLower(rune(target.(*Number).Value))))
}

// NumberAsString is a Number method.
//
// asString returns the decimal string representation of the target.
func NumberAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(target.(*Number).String())
}

// NumberAsUint32Buffer is a Number method.
//
// asUint32Buffer returns a 4-byte buffer representing the target's value
// converted to a uint32.
func NumberAsUint32Buffer(vm *VM, target, locals Interface, msg *Message) Interface {
	x := uint32(target.(*Number).Value)
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, x)
	return vm.NewSequence(v, true, "latin1")
}

// NumberAsUppercase is a Number method.
//
// asUppercase returns the Number which is the Unicode codepoint corresponding
// to the uppercase version of the target as a Unicode codepoint.
func NumberAsUppercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(unicode.ToUpper(rune(target.(*Number).Value))))
}

// NumberAsin is a Number method.
//
// asin returns the arcsine of the target.
func NumberAsin(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Asin(target.(*Number).Value))
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
func NumberAt(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) >> uint64(arg.Value) & 1))
}

// NumberAtan is a Number method.
//
// atan returns the arctangent of the target.
func NumberAtan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Atan(target.(*Number).Value))
}

// NumberAtan2 is a Number method.
//
// atan2 returns the directional arctangent of the target divided by the
// argument.
func NumberAtan2(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(math.Atan2(target.(*Number).Value, arg.Value))
}

// NumberBetween is a Number method.
//
// between is true if the target is greater than or equal to the first argument
// and less than or equal to the second.
func NumberBetween(vm *VM, target, locals Interface, msg *Message) Interface {
	arg1, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	arg2, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != nil {
		return stop
	}
	v := target.(*Number).Value
	return vm.IoBool(arg1.Value <= v && v <= arg2.Value)
}

// NumberBitwiseAnd is a Number method.
//
// bitwiseAnd returns the bitwise intersection of the target and the argument,
// with each converted to 64-bit integers.
func NumberBitwiseAnd(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) & int64(arg.Value)))
}

// NumberBitwiseComplement is a Number method.
//
// bitwiseComplement returns the bitwise complement of the 64-bit integer value
// of the target.
func NumberBitwiseComplement(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(^int64(target.(*Number).Value)))
}

// NumberBitwiseOr is a Number method.
//
// bitwiseOr returns the bitwise union of the target and the argument, with
// each converted to 64-bit integers.
func NumberBitwiseOr(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) | int64(arg.Value)))
}

// NumberBitwiseXor is a Number method.
//
// bitwiseXor returns the bitwise symmetric difference of the target and the
// argument, with each converted to 64-bit integers.
func NumberBitwiseXor(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) ^ int64(arg.Value)))
}

// NumberCeil is a Number method.
//
// ceil returns the smallest integer larger than or equal to the target.
func NumberCeil(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Ceil(target.(*Number).Value))
}

// NumberClip is a Number method.
//
// clip returns the target if it is between the given bounds or else the
// exceeded bound.
func NumberClip(vm *VM, target, locals Interface, msg *Message) Interface {
	arg1, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	arg2, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != nil {
		return stop
	}
	v := target.(*Number).Value
	if v > arg2.Value {
		return arg2
	}
	if v < arg1.Value {
		return arg1
	}
	return target
}

// NumberCompare is a Number method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func NumberCompare(vm *VM, target, locals Interface, msg *Message) Interface {
	// Io doesn't actually define a Number compare, but I'm doing it anyway.
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	a, b := target.(*Number).Value, arg.Value
	if a < b {
		return vm.NewNumber(-1)
	}
	if a > b {
		return vm.NewNumber(1)
	}
	return vm.NewNumber(0)
}

// NumberCos is a Number method.
//
// cos returns the cosine of the target.
func NumberCos(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Cos(target.(*Number).Value))
}

// NumberCubed is a Number method.
//
// cubed returns the target raised to the third power.
func NumberCubed(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	return vm.NewNumber(x * x * x)
}

// NumberDiv is a Number method.
//
// / is an operator which divides the left value by the right.
func NumberDiv(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(target.(*Number).Value / arg.Value)
}

// NumberExp is a Number method.
//
// exp returns e (the base of the natural logarithm) raised to the power of
// the target.
func NumberExp(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Exp(target.(*Number).Value))
}

// NumberFactorial is a Number method.
//
// factorial computes the product of each integer between 1 and the target.
func NumberFactorial(vm *VM, target, locals Interface, msg *Message) Interface {
	x := int64(target.(*Number).Value)
	if x < 0 {
		return vm.NewNumber(math.NaN())
	}
	v := 1.0
	for x > 0 {
		v *= float64(x)
		x--
	}
	return vm.NewNumber(v)
}

// NumberFloor is a Number method.
//
// floor returns the largest integer smaller than or equal to the target.
func NumberFloor(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Floor(target.(*Number).Value))
}

// NumberIsAlphaNumeric is a Number method.
//
// isAlphaNumeric is true if the target is a Unicode codepoint corresponding to
// a letter (category L) or number (category N).
func NumberIsAlphaNumeric(vm *VM, target, locals Interface, msg *Message) Interface {
	x := rune(target.(*Number).Value)
	return vm.IoBool(unicode.IsLetter(x) || unicode.IsNumber(x))
}

// NumberIsControlCharacter is a Number method.
//
// isControlCharacter is true if the target is a Unicode codepoint
// corresponding to a control character.
func NumberIsControlCharacter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsControl(rune(target.(*Number).Value)))
}

// NumberIsDigit is a Number method.
//
// isDigit is true if the target is a Unicode codepoint corresponding to a
// decimal digit.
func NumberIsDigit(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsDigit(rune(target.(*Number).Value)))
}

// NumberIsEven is a Number method.
//
// isEven is true if the integer value of the target is divisible by 2.
func NumberIsEven(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(int64(target.(*Number).Value)&1 == 0)
}

// NumberIsGraph is a Number method.
//
// isGraph is true if the target is a Unicode codepoint corresponding to a
// graphic character (categories L, M, N, P, S, Zs).
func NumberIsGraph(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsGraphic(rune(target.(*Number).Value)))
}

// NumberIsHexDigit is a Number method.
//
// isHexDigit is true if the target is a Unicode codepoint corresponding to the
// characters 0 through 9, A through F, or a through f.
func NumberIsHexDigit(vm *VM, target, locals Interface, msg *Message) Interface {
	x := rune(target.(*Number).Value)
	return vm.IoBool(('0' <= x && x <= '9') || ('A' <= x && x <= 'F') || ('a' <= x && x <= 'f'))
}

// NumberIsLetter is a Number method.
//
// isLetter is true if the target is a Unicode codepoint corresponding to a
// letter (category L).
func NumberIsLetter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsLetter(rune(target.(*Number).Value)))
}

// NumberIsLowercase is a Number method.
//
// isLowercase is true if the target is a Unicode codepoint corresponding to a
// lowercase letter (category Ll).
func NumberIsLowercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsLower(rune(target.(*Number).Value)))
}

// NumberIsNan is a Number method.
//
// isNan is true if the target is an IEEE-754 Not a Number.
func NumberIsNan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(math.IsNaN(target.(*Number).Value))
}

// NumberIsOdd is a Number method.
//
// isOdd is true if the integer value of the target is not divisible by 2.
func NumberIsOdd(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(int64(target.(*Number).Value)&1 == 1)
}

// NumberIsPrint is a Number method.
//
// isPrint is true if the target is a Unicode codepoint corresponding to a
// printable character (categories L, M, N, P, S, and ASCII space, U+0020).
func NumberIsPrint(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsPrint(rune(target.(*Number).Value)))
}

// NumberIsPunctuation is a Number method.
//
// isPunctuation is true if the target is a Unicode codepoint corresponding to
// a punctuation character (category P).
func NumberIsPunctuation(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsPunct(rune(target.(*Number).Value)))
}

// NumberIsSpace is a Number method.
//
// isSpace is true if the target is a Unicode codepoint corresponding to a
// space character.
func NumberIsSpace(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsSpace(rune(target.(*Number).Value)))
}

// NumberIsUppercase is a Number method.
//
// isUppercase is true if the target is a Unicode codepoint corresponding to an
// uppercase letter (category Lu).
func NumberIsUppercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(unicode.IsUpper(rune(target.(*Number).Value)))
}

// NumberLog is a Number method.
//
// log returns the natural logarithm of the target.
func NumberLog(vm *VM, target, locals Interface, msg *Message) Interface {
	if msg.ArgCount() > 0 {
		b, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != nil {
			return stop
		}
		return vm.NewNumber(math.Log(target.(*Number).Value) / math.Log(b.Value))
	}
	return vm.NewNumber(math.Log(target.(*Number).Value))
}

// NumberLog10 is a Number method.
//
// log10 returns the base-10 logarithm of the target.
func NumberLog10(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Log10(target.(*Number).Value))
}

// NumberLog2 is a Number method.
//
// log2 returns the base-2 logarithm of the target.
func NumberLog2(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Log2(target.(*Number).Value))
}

// NumberMax is a Number method.
//
// max returns the larger of the target and the argument.
func NumberMax(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if target.(*Number).Value >= arg.Value {
		return target
	}
	return arg
}

// NumberMin is a Number method.
//
// min returns the smaller of the target and the argument.
func NumberMin(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if target.(*Number).Value <= arg.Value {
		return target
	}
	return arg
}

// NumberMod is a Number method.
//
// mod returns the remainder of division of the target by the argument.
func NumberMod(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(math.Remainder(target.(*Number).Value, arg.Value))
}

// NumberNegate is a Number method.
//
// negate returns the opposite of the target.
func NumberNegate(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(-target.(*Number).Value)
}

// NumberMul is a Number method.
//
// * is an operator which multiplies its operands.
func NumberMul(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(target.(*Number).Value * arg.Value)
}

// NumberPow is a Number method.
//
// pow returns the target raised to the power of the argument. The ** operator
// is equivalent.
func NumberPow(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(math.Pow(target.(*Number).Value, arg.Value))
}

// NumberRound is a Number method.
//
// round returns the integer nearest the target, with halfway cases rounding
// away from zero.
func NumberRound(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	if x < 0 {
		x = math.Ceil(x - 0.5)
	} else {
		x = math.Floor(x + 0.5)
	}
	return vm.NewNumber(x)
}

// NumberRoundDown is a Number method.
//
// roundDown returns the integer nearest the target, with halfway cases
// rounding toward positive infinity.
func NumberRoundDown(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Floor(target.(*Number).Value + 0.5))
}

// NumberShiftLeft is a Number method.
//
// shiftLeft returns the target as a 64-bit integer shifted left by the
// argument as a 64-bit unsigned integer.
func NumberShiftLeft(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) << uint64(arg.Value)))
}

// NumberShiftRight is a Number method.
//
// shiftRight returns the target as a 64-bit integer shifted right by the
// argument as a 64-bit unsigned integer.
func NumberShiftRight(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) >> uint64(arg.Value)))
}

// NumberSin is a Number method.
//
// sin returns the sine of the target.
func NumberSin(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Sin(target.(*Number).Value))
}

// NumberSqrt is a Number method.
//
// sqrt returns the square root of the target.
func NumberSqrt(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Sqrt(target.(*Number).Value))
}

// NumberSquared is a Number method.
//
// squared returns the target raised to the second power.
func NumberSquared(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	return vm.NewNumber(x * x)
}

// NumberSub is a Number method.
//
// - is an operator which subtracts the right value from the left.
func NumberSub(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	return vm.NewNumber(target.(*Number).Value - arg.Value)
}

// NumberTan is a Number method.
//
// tan returns the tangent of the target.
func NumberTan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Tan(target.(*Number).Value))
}

// NumberToBase is a Number method.
//
// toBase returns the string representation of the target in the radix given in
// the argument. Bases less than 2 and greater than 36 are not supported.
func NumberToBase(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	base := int(arg.Value)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), base))
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
func NumberToBaseWholeBytes(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	base := int(arg.Value)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	cols := int(toBaseWholeBytesCols[base-2])
	s := strconv.FormatInt(int64(target.(*Number).Value), base)
	w := (len(s) + cols - 1) / cols
	return vm.NewString(strings.Repeat("0", w*cols-len(s)) + s)
}

// NumberToggle is a Number method.
//
// toggle returns 1 if the target is 0, otherwise 0.
func NumberToggle(vm *VM, target, locals Interface, msg *Message) Interface {
	if target.(*Number).Value == 0 {
		return vm.NewNumber(1)
	}
	return vm.NewNumber(0)
}
