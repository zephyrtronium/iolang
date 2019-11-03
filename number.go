package iolang

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// NumberTag is the tag for Number objects.
const NumberTag = BasicTag("Number")

// NewNumber creates a Number object with a given value.
func (vm *VM) NewNumber(value float64) *Object {
	return vm.ObjectWith(nil, vm.CoreProto("Number"), value, NumberTag)
}

// NumberArgAt evaluates the nth argument and returns its Number value. If a
// stop occurs during evaluation, the value will be 0, and the stop status and
// result will be returned. If the evaluated result is not a Number, the result
// will be 0 and an exception will be returned with an ExceptionStop.
func (m *Message) NumberArgAt(vm *VM, locals *Object, n int) (float64, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		// We still have to lock here; if v is not a Number, its value could
		// change during the type assertion.
		v.Lock()
		x, ok := v.Value.(float64)
		v.Unlock()
		if ok {
			return x, nil, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Number, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return 0, v, s
}

// initNumber initializes Number on this VM.
func (vm *VM) initNumber() {
	slots := Slots{
		"*":                  vm.NewCFunction(NumberMul, NumberTag),
		"+":                  vm.NewCFunction(NumberAdd, NumberTag),
		"-":                  vm.NewCFunction(NumberSub, NumberTag),
		"/":                  vm.NewCFunction(NumberDiv, NumberTag),
		"abs":                vm.NewCFunction(NumberAbs, NumberTag),
		"acos":               vm.NewCFunction(NumberAcos, NumberTag),
		"asBuffer":           vm.NewCFunction(NumberAsBuffer, NumberTag),
		"asCharacter":        vm.NewCFunction(NumberAsCharacter, NumberTag),
		"asLowercase":        vm.NewCFunction(NumberAsLowercase, NumberTag),
		"asNumber":           vm.NewCFunction(ObjectThisContext, NumberTag), // hax
		"asString":           vm.NewCFunction(NumberAsString, NumberTag),
		"asUint32Buffer":     vm.NewCFunction(NumberAsUint32Buffer, NumberTag),
		"asUppercase":        vm.NewCFunction(NumberAsUppercase, NumberTag),
		"asin":               vm.NewCFunction(NumberAsin, NumberTag),
		"at":                 vm.NewCFunction(NumberAt, NumberTag),
		"atan":               vm.NewCFunction(NumberAtan, NumberTag),
		"atan2":              vm.NewCFunction(NumberAtan2, NumberTag),
		"between":            vm.NewCFunction(NumberBetween, NumberTag),
		"bitwiseAnd":         vm.NewCFunction(NumberBitwiseAnd, NumberTag),
		"bitwiseComplement":  vm.NewCFunction(NumberBitwiseComplement, NumberTag),
		"bitwiseOr":          vm.NewCFunction(NumberBitwiseOr, NumberTag),
		"bitwiseXor":         vm.NewCFunction(NumberBitwiseXor, NumberTag),
		"ceil":               vm.NewCFunction(NumberCeil, NumberTag),
		"clip":               vm.NewCFunction(NumberClip, NumberTag),
		"compare":            vm.NewCFunction(NumberCompare, NumberTag),
		"cos":                vm.NewCFunction(NumberCos, NumberTag),
		"cubed":              vm.NewCFunction(NumberCubed, NumberTag),
		"exp":                vm.NewCFunction(NumberExp, NumberTag),
		"factorial":          vm.NewCFunction(NumberFactorial, NumberTag),
		"floor":              vm.NewCFunction(NumberFloor, NumberTag),
		"isAlphaNumeric":     vm.NewCFunction(NumberIsAlphaNumeric, NumberTag),
		"isControlCharacter": vm.NewCFunction(NumberIsControlCharacter, NumberTag),
		"isDigit":            vm.NewCFunction(NumberIsDigit, NumberTag),
		"isEven":             vm.NewCFunction(NumberIsEven, NumberTag),
		"isHexDigit":         vm.NewCFunction(NumberIsHexDigit, NumberTag),
		"isLetter":           vm.NewCFunction(NumberIsLetter, NumberTag),
		"isLowercase":        vm.NewCFunction(NumberIsLowercase, NumberTag),
		"isNan":              vm.NewCFunction(NumberIsNan, NumberTag),
		"isOdd":              vm.NewCFunction(NumberIsOdd, NumberTag),
		"isPrint":            vm.NewCFunction(NumberIsPrint, NumberTag),
		"isPunctuation":      vm.NewCFunction(NumberIsPunctuation, NumberTag),
		"isSpace":            vm.NewCFunction(NumberIsSpace, NumberTag),
		"isUppercase":        vm.NewCFunction(NumberIsUppercase, NumberTag),
		"log":                vm.NewCFunction(NumberLog, NumberTag),
		"log10":              vm.NewCFunction(NumberLog10, NumberTag),
		"log2":               vm.NewCFunction(NumberLog2, NumberTag),
		"max":                vm.NewCFunction(NumberMax, NumberTag),
		"min":                vm.NewCFunction(NumberMin, NumberTag),
		"mod":                vm.NewCFunction(NumberMod, NumberTag),
		"negate":             vm.NewCFunction(NumberNegate, NumberTag),
		"pow":                vm.NewCFunction(NumberPow, NumberTag),
		"repeat":             vm.NewCFunction(NumberRepeat, NumberTag),
		"round":              vm.NewCFunction(NumberRound, NumberTag),
		"roundDown":          vm.NewCFunction(NumberRoundDown, NumberTag),
		"shiftLeft":          vm.NewCFunction(NumberShiftLeft, NumberTag),
		"shiftRight":         vm.NewCFunction(NumberShiftRight, NumberTag),
		"sin":                vm.NewCFunction(NumberSin, NumberTag),
		"sqrt":               vm.NewCFunction(NumberSqrt, NumberTag),
		"squared":            vm.NewCFunction(NumberSquared, NumberTag),
		"tan":                vm.NewCFunction(NumberTan, NumberTag),
		"toBase":             vm.NewCFunction(NumberToBase, NumberTag),
		"toBaseWholeBytes":   vm.NewCFunction(NumberToBaseWholeBytes, NumberTag),
		"toggle":             vm.NewCFunction(NumberToggle, NumberTag),
		"type":               vm.NewString("Number"),
	}
	vm.coreInstall("Number", slots, float64(0), NumberTag)

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
	slots["constants"] = vm.NewObject(Slots{
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
func NumberAbs(vm *VM, target, locals *Object, msg *Message) *Object {
	n := target.Value.(float64)
	if n < 0 {
		return vm.NewNumber(-n)
	}
	return target
}

// NumberAcos is a Number method.
//
// acos returns the arccosine of the target.
func NumberAcos(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Acos(target.Value.(float64)))
}

// NumberAdd is a Number method.
//
// + is an operator which sums two numbers.
func NumberAdd(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(target.Value.(float64) + arg)
}

// NumberAsBuffer is a Number method.
//
// asBuffer creates a Latin-1 Sequence with bytes equal to the binary
// representation of the target. An optional byte count for the size of the
// buffer may be supplied, with a default of 8.
func NumberAsBuffer(vm *VM, target, locals *Object, msg *Message) *Object {
	n := 8
	if msg.ArgCount() > 0 {
		arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		n = int(arg)
		if n < 0 {
			return vm.RaiseExceptionf("buffer size must be nonnegative")
		}
	}
	x := target.Value.(float64)
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
func NumberAsCharacter(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(string(rune(target.Value.(float64))))
}

// NumberAsLowercase is a Number method.
//
// asLowercase returns the Number which is the Unicode codepoint corresponding
// to the lowercase version of the target as a Unicode codepoint.
func NumberAsLowercase(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(float64(unicode.ToLower(rune(target.Value.(float64)))))
}

// NumberAsString is a Number method.
//
// asString returns the decimal string representation of the target.
func NumberAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(strconv.FormatFloat(target.Value.(float64), 'g', -1, 64))
}

// NumberAsUint32Buffer is a Number method.
//
// asUint32Buffer returns a 4-byte buffer representing the target's value
// converted to a uint32.
func NumberAsUint32Buffer(vm *VM, target, locals *Object, msg *Message) *Object {
	x := uint32(target.Value.(float64))
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, x)
	return vm.NewSequence(v, true, "latin1")
}

// NumberAsUppercase is a Number method.
//
// asUppercase returns the Number which is the Unicode codepoint corresponding
// to the uppercase version of the target as a Unicode codepoint.
func NumberAsUppercase(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(float64(unicode.ToUpper(rune(target.Value.(float64)))))
}

// NumberAsin is a Number method.
//
// asin returns the arcsine of the target.
func NumberAsin(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Asin(target.Value.(float64)))
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
func NumberAt(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) >> uint64(arg) & 1))
}

// NumberAtan is a Number method.
//
// atan returns the arctangent of the target.
func NumberAtan(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Atan(target.Value.(float64)))
}

// NumberAtan2 is a Number method.
//
// atan2 returns the directional arctangent of the target divided by the
// argument.
func NumberAtan2(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(math.Atan2(target.Value.(float64), arg))
}

// NumberBetween is a Number method.
//
// between is true if the target is greater than or equal to the first argument
// and less than or equal to the second.
func NumberBetween(vm *VM, target, locals *Object, msg *Message) *Object {
	arg1, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	arg2, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v := target.Value.(float64)
	return vm.IoBool(arg1 <= v && v <= arg2)
}

// NumberBitwiseAnd is a Number method.
//
// bitwiseAnd returns the bitwise intersection of the target and the argument,
// with each converted to 64-bit integers.
func NumberBitwiseAnd(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) & int64(arg)))
}

// NumberBitwiseComplement is a Number method.
//
// bitwiseComplement returns the bitwise complement of the 64-bit integer value
// of the target.
func NumberBitwiseComplement(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(float64(^int64(target.Value.(float64))))
}

// NumberBitwiseOr is a Number method.
//
// bitwiseOr returns the bitwise union of the target and the argument, with
// each converted to 64-bit integers.
func NumberBitwiseOr(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) | int64(arg)))
}

// NumberBitwiseXor is a Number method.
//
// bitwiseXor returns the bitwise symmetric difference of the target and the
// argument, with each converted to 64-bit integers.
func NumberBitwiseXor(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) ^ int64(arg)))
}

// NumberCeil is a Number method.
//
// ceil returns the smallest integer larger than or equal to the target.
func NumberCeil(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Ceil(target.Value.(float64)))
}

// NumberClip is a Number method.
//
// clip returns the target if it is between the given bounds or else the
// exceeded bound.
func NumberClip(vm *VM, target, locals *Object, msg *Message) *Object {
	arg1, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	arg2, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	v := target.Value.(float64)
	if v > arg2 {
		return vm.NewNumber(arg2)
	}
	if v < arg1 {
		return vm.NewNumber(arg1)
	}
	return target
}

// NumberCompare is a Number method.
//
// compare returns -1 if the receiver is less than the argument, 1 if it is
// greater, or 0 if they are equal.
func NumberCompare(vm *VM, target, locals *Object, msg *Message) *Object {
	// Io doesn't actually define a Number compare, but I'm doing it anyway.
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	a, b := target.Value.(float64), arg
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
func NumberCos(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Cos(target.Value.(float64)))
}

// NumberCubed is a Number method.
//
// cubed returns the target raised to the third power.
func NumberCubed(vm *VM, target, locals *Object, msg *Message) *Object {
	x := target.Value.(float64)
	return vm.NewNumber(x * x * x)
}

// NumberDiv is a Number method.
//
// / is an operator which divides the left value by the right.
func NumberDiv(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(target.Value.(float64) / arg)
}

// NumberExp is a Number method.
//
// exp returns e (the base of the natural logarithm) raised to the power of
// the target.
func NumberExp(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Exp(target.Value.(float64)))
}

// NumberFactorial is a Number method.
//
// factorial computes the product of each integer between 1 and the target.
func NumberFactorial(vm *VM, target, locals *Object, msg *Message) *Object {
	x := int64(target.Value.(float64))
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
func NumberFloor(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Floor(target.Value.(float64)))
}

// NumberIsAlphaNumeric is a Number method.
//
// isAlphaNumeric is true if the target is a Unicode codepoint corresponding to
// a letter (category L) or number (category N).
func NumberIsAlphaNumeric(vm *VM, target, locals *Object, msg *Message) *Object {
	x := rune(target.Value.(float64))
	return vm.IoBool(unicode.IsLetter(x) || unicode.IsNumber(x))
}

// NumberIsControlCharacter is a Number method.
//
// isControlCharacter is true if the target is a Unicode codepoint
// corresponding to a control character.
func NumberIsControlCharacter(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsControl(rune(target.Value.(float64))))
}

// NumberIsDigit is a Number method.
//
// isDigit is true if the target is a Unicode codepoint corresponding to a
// decimal digit.
func NumberIsDigit(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsDigit(rune(target.Value.(float64))))
}

// NumberIsEven is a Number method.
//
// isEven is true if the integer value of the target is divisible by 2.
func NumberIsEven(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(int64(target.Value.(float64))&1 == 0)
}

// NumberIsGraph is a Number method.
//
// isGraph is true if the target is a Unicode codepoint corresponding to a
// graphic character (categories L, M, N, P, S, Zs).
func NumberIsGraph(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsGraphic(rune(target.Value.(float64))))
}

// NumberIsHexDigit is a Number method.
//
// isHexDigit is true if the target is a Unicode codepoint corresponding to the
// characters 0 through 9, A through F, or a through f.
func NumberIsHexDigit(vm *VM, target, locals *Object, msg *Message) *Object {
	x := rune(target.Value.(float64))
	return vm.IoBool(('0' <= x && x <= '9') || ('A' <= x && x <= 'F') || ('a' <= x && x <= 'f'))
}

// NumberIsLetter is a Number method.
//
// isLetter is true if the target is a Unicode codepoint corresponding to a
// letter (category L).
func NumberIsLetter(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsLetter(rune(target.Value.(float64))))
}

// NumberIsLowercase is a Number method.
//
// isLowercase is true if the target is a Unicode codepoint corresponding to a
// lowercase letter (category Ll).
func NumberIsLowercase(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsLower(rune(target.Value.(float64))))
}

// NumberIsNan is a Number method.
//
// isNan is true if the target is an IEEE-754 Not a Number.
func NumberIsNan(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(math.IsNaN(target.Value.(float64)))
}

// NumberIsOdd is a Number method.
//
// isOdd is true if the integer value of the target is not divisible by 2.
func NumberIsOdd(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(int64(target.Value.(float64))&1 == 1)
}

// NumberIsPrint is a Number method.
//
// isPrint is true if the target is a Unicode codepoint corresponding to a
// printable character (categories L, M, N, P, S, and ASCII space, U+0020).
func NumberIsPrint(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsPrint(rune(target.Value.(float64))))
}

// NumberIsPunctuation is a Number method.
//
// isPunctuation is true if the target is a Unicode codepoint corresponding to
// a punctuation character (category P).
func NumberIsPunctuation(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsPunct(rune(target.Value.(float64))))
}

// NumberIsSpace is a Number method.
//
// isSpace is true if the target is a Unicode codepoint corresponding to a
// space character.
func NumberIsSpace(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsSpace(rune(target.Value.(float64))))
}

// NumberIsUppercase is a Number method.
//
// isUppercase is true if the target is a Unicode codepoint corresponding to an
// uppercase letter (category Lu).
func NumberIsUppercase(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(unicode.IsUpper(rune(target.Value.(float64))))
}

// NumberLog is a Number method.
//
// log returns the natural logarithm of the target.
func NumberLog(vm *VM, target, locals *Object, msg *Message) *Object {
	if msg.ArgCount() > 0 {
		b, exc, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		return vm.NewNumber(math.Log(target.Value.(float64)) / math.Log(b))
	}
	return vm.NewNumber(math.Log(target.Value.(float64)))
}

// NumberLog10 is a Number method.
//
// log10 returns the base-10 logarithm of the target.
func NumberLog10(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Log10(target.Value.(float64)))
}

// NumberLog2 is a Number method.
//
// log2 returns the base-2 logarithm of the target.
func NumberLog2(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Log2(target.Value.(float64)))
}

// NumberMax is a Number method.
//
// max returns the larger of the target and the argument.
func NumberMax(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	if target.Value.(float64) >= arg {
		return target
	}
	return vm.NewNumber(arg)
}

// NumberMin is a Number method.
//
// min returns the smaller of the target and the argument.
func NumberMin(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	if target.Value.(float64) <= arg {
		return target
	}
	return vm.NewNumber(arg)
}

// NumberMod is a Number method.
//
// mod returns the remainder of division of the target by the argument.
func NumberMod(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(math.Remainder(target.Value.(float64), arg))
}

// NumberNegate is a Number method.
//
// negate returns the opposite of the target.
func NumberNegate(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(-target.Value.(float64))
}

// NumberMul is a Number method.
//
// * is an operator which multiplies its operands.
func NumberMul(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(target.Value.(float64) * arg)
}

// NumberPow is a Number method.
//
// pow returns the target raised to the power of the argument. The ** operator
// is equivalent.
func NumberPow(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(math.Pow(target.Value.(float64), arg))
}

// NumberRepeat is a Number method.
//
// repeat performs a loop the given number of times.
func NumberRepeat(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	if len(msg.Args) < 1 {
		return vm.RaiseExceptionf("Number repeat requires 1 or 2 arguments")
	}
	counter, eval := msg.ArgAt(0), msg.ArgAt(1)
	c := counter.Name()
	if eval == nil {
		// One argument was supplied.
		counter, eval = nil, counter
	}
	max := int(math.Ceil(target.Value.(float64)))
	var control Stop
	for i := 0; i < max; i++ {
		if counter != nil {
			locals.SetSlot(c, vm.NewNumber(float64(i)))
		}
		result, control = eval.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
	return result
}

// NumberRound is a Number method.
//
// round returns the integer nearest the target, with halfway cases rounding
// away from zero.
func NumberRound(vm *VM, target, locals *Object, msg *Message) *Object {
	x := target.Value.(float64)
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
func NumberRoundDown(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Floor(target.Value.(float64) + 0.5))
}

// NumberShiftLeft is a Number method.
//
// shiftLeft returns the target as a 64-bit integer shifted left by the
// argument as a 64-bit unsigned integer.
func NumberShiftLeft(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) << uint64(arg)))
}

// NumberShiftRight is a Number method.
//
// shiftRight returns the target as a 64-bit integer shifted right by the
// argument as a 64-bit unsigned integer.
func NumberShiftRight(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(float64(int64(target.Value.(float64)) >> uint64(arg)))
}

// NumberSin is a Number method.
//
// sin returns the sine of the target.
func NumberSin(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Sin(target.Value.(float64)))
}

// NumberSqrt is a Number method.
//
// sqrt returns the square root of the target.
func NumberSqrt(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Sqrt(target.Value.(float64)))
}

// NumberSquared is a Number method.
//
// squared returns the target raised to the second power.
func NumberSquared(vm *VM, target, locals *Object, msg *Message) *Object {
	x := target.Value.(float64)
	return vm.NewNumber(x * x)
}

// NumberSub is a Number method.
//
// - is an operator which subtracts the right value from the left.
func NumberSub(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.NewNumber(target.Value.(float64) - arg)
}

// NumberTan is a Number method.
//
// tan returns the tangent of the target.
func NumberTan(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewNumber(math.Tan(target.Value.(float64)))
}

// NumberToBase is a Number method.
//
// toBase returns the string representation of the target in the radix given in
// the argument. Bases less than 2 and greater than 36 are not supported.
func NumberToBase(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	base := int(arg)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	return vm.NewString(strconv.FormatInt(int64(target.Value.(float64)), base))
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
func NumberToBaseWholeBytes(vm *VM, target, locals *Object, msg *Message) *Object {
	arg, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	base := int(arg)
	if base < 2 || base > 36 {
		return vm.RaiseExceptionf("conversion to base %d not supported", base)
	}
	cols := int(toBaseWholeBytesCols[base-2])
	s := strconv.FormatInt(int64(target.Value.(float64)), base)
	w := (len(s) + cols - 1) / cols
	return vm.NewString(strings.Repeat("0", w*cols-len(s)) + s)
}

// NumberToggle is a Number method.
//
// toggle returns 1 if the target is 0, otherwise 0.
func NumberToggle(vm *VM, target, locals *Object, msg *Message) *Object {
	if target.Value.(float64) == 0 {
		return vm.NewNumber(1)
	}
	return vm.NewNumber(0)
}
