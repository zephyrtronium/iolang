package iolang

import (
	"math"
	"strconv"
	"unicode"
)

// Numeric type. These should be considered immutable; change Value only with
// extreme magic.
type Number struct {
	Object
	Value float64
}

// Create a Number object with a particular value. If the value is memoized by
// the VM, that object is returned; otherwise, a new object will be created.
func (vm *VM) NewNumber(value float64) *Number {
	if x, ok := vm.NumberMemo[value]; ok {
		return x
	}
	return &Number{
		Object{Slots: vm.DefaultSlots["Number"], Protos: []Interface{vm.BaseObject}},
		value,
	}
}

func (n *Number) Clone() Interface {
	return &Number{
		Object{Slots: Slots{}, Protos: []Interface{n}},
		n.Value,
	}
}

func (n *Number) String() string {
	return strconv.FormatFloat(n.Value, 'g', -1, 64)
}

func (vm *VM) initNumber() {
	slots := Slots{
		"abs":               vm.NewCFunction(NumberAbs, "NumberAbs()"),
		"acos":              vm.NewCFunction(NumberAcos, "NumberAcos()"),
		"asBinary":          vm.NewCFunction(NumberAsBinary, "NumberAsBinary()"),
		"asCharacter":       vm.NewCFunction(NumberAsCharacter, "NumberAsCharacter()"),
		"asHex":             vm.NewCFunction(NumberAsHex, "NumberAsHex()"),
		"asLowercase":       vm.NewCFunction(NumberAsLowercase, "NumberAsLowercase()"),
		"asNumber":          vm.NewCFunction(NumberAsNumber, "NumberAsNumber()"),
		"asOctal":           vm.NewCFunction(NumberAsOctal, "NumberAsOctal()"),
		"asString":          vm.NewCFunction(NumberAsString, "NumberAsString()"),
		"asUppercase":       vm.NewCFunction(NumberAsUppercase, "NumberAsUppercase()"),
		"asin":              vm.NewCFunction(NumberAsin, "NumberAsin()"),
		"at":                vm.NewCFunction(NumberAt, "NumberAt(idx)"),
		"atan":              vm.NewCFunction(NumberAtan, "NumberAtan()"),
		"atan2":             vm.NewCFunction(NumberAtan2, "NumberAtan2(x)"),
		"between":           vm.NewCFunction(NumberBetween, "NumberBetween(low, high)"),
		"bitwiseAnd":        vm.NewCFunction(NumberBitwiseAnd, "NumberBitwiseAnd(v)"),
		"bitwiseComplement": vm.NewCFunction(NumberBitwiseComplement, "NumberBitwiseComplement()"),
		"bitwiseOr":         vm.NewCFunction(NumberBitwiseOr, "NumberBitwiseOr(v)"),
		"bitwiseXor":        vm.NewCFunction(NumberBitwiseXor, "NumberBitwiseXor(v)"),
		"ceil":              vm.NewCFunction(NumberCeil, "NumberCeil()"),
		"clip":              vm.NewCFunction(NumberClip, "NumberClip(low, high)"),
		"constants": vm.ObjectWith(Slots{
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
		}),
		"cos":                vm.NewCFunction(NumberCos, "NumberCos()"),
		"cubed":              vm.NewCFunction(NumberCubed, "NumberCubed()"),
		"exp":                vm.NewCFunction(NumberExp, "NumberExp()"),
		"factorial":          vm.NewCFunction(NumberFactorial, "NumberFactorial()"),
		"floor":              vm.NewCFunction(NumberFloor, "NumberFloor()"),
		"isAlphaNumeric":     vm.NewCFunction(NumberIsAlphaNumeric, "NumberIsAlphaNumeric()"),
		"isControlCharacter": vm.NewCFunction(NumberIsControlCharacter, "NumberIsControlCharacter()"),
		"isDigit":            vm.NewCFunction(NumberIsDigit, "NumberIsDigit()"),
		"isEven":             vm.NewCFunction(NumberIsEven, "NumberIsEven()"),
		"isHexDigit":         vm.NewCFunction(NumberIsHexDigit, "NumberIsHexDigit()"),
		"isLetter":           vm.NewCFunction(NumberIsLetter, "NumberIsLetter()"),
		"isLowercase":        vm.NewCFunction(NumberIsLowercase, "NumberIsLowercase()"),
		"isNan":              vm.NewCFunction(NumberIsNan, "NumberIsNan()"),
		"isOdd":              vm.NewCFunction(NumberIsOdd, "NumberIsOdd()"),
		"isPrint":            vm.NewCFunction(NumberIsPrint, "NumberIsPrint()"),
		"isPunctuation":      vm.NewCFunction(NumberIsPunctuation, "NumberIsPunctuation()"),
		"isSpace":            vm.NewCFunction(NumberIsSpace, "NumberIsSpace()"),
		"isUppercase":        vm.NewCFunction(NumberIsUppercase, "NumberIsUppercase()"),
		"log":                vm.NewCFunction(NumberLog, "NumberLog()"),
		"log10":              vm.NewCFunction(NumberLog10, "NumberLog10()"),
		"max":                vm.NewCFunction(NumberMax, "NumberMax(other)"),
		"min":                vm.NewCFunction(NumberMin, "NumberMin(other)"),
		"mod":                vm.NewCFunction(NumberMod, "NumberMod(v)"),
		"negate":             vm.NewCFunction(NumberNegate, "NumberNegate()"),
		"pow":                vm.NewCFunction(NumberPow, "NumberPow(v)"),
		"repeat":             vm.NewCFunction(NumberRepeat, "NumberRepeat([counter,] message"),
		"round":              vm.NewCFunction(NumberRound, "NumberRound()"),
		"roundDown":          vm.NewCFunction(NumberRoundDown, "NumberRoundDown()"),
		"shiftLeft":          vm.NewCFunction(NumberShiftLeft, "NumberShiftLeft(v)"),
		"shiftRight":         vm.NewCFunction(NumberShiftRight, "NumberShiftRight(v)"),
		"sin":                vm.NewCFunction(NumberSin, "NumberSin()"),
		"sqrt":               vm.NewCFunction(NumberSqrt, "NumberSqrt()"),
		"squared":            vm.NewCFunction(NumberSquared, "NumberSquared()"),
		"tan":                vm.NewCFunction(NumberTan, "NumberTan()"),
		"toBase":             vm.NewCFunction(NumberToBase, "NumberToBase(b)"),
		"toggle":             vm.NewCFunction(NumberToggle, "NumberToggle()"),
	}
	vm.DefaultSlots["Number"] = slots

	for i := -1; i <= 255; i++ {
		vm.MemoizeNumber(float64(i))
	}
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
	// TODO: long/unsigned limits, toBaseWholeBytes
}

func NumberAbs(vm *VM, target, locals Interface, msg *Message) Interface {
	n := target.(*Number)
	if n.Value < 0 {
		return vm.NewNumber(-n.Value)
	}
	return target
}

func NumberAcos(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Acos(target.(*Number).Value))
}

func NumberAsBinary(vm *VM, target, locals Interface, msg *Message) Interface {
	// TODO: zero-fill to octets
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), 2))
}

func NumberAsCharacter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(string(rune(target.(*Number).Value)))
}

func NumberAsHex(vm *VM, target, locals Interface, msg *Message) Interface {
	// TODO: zero-fill to octets
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), 16))
}

func NumberAsLowercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(unicode.ToLower(rune(target.(*Number).Value))))
}

func NumberAsNumber(vm *VM, target, locals Interface, msg *Message) Interface {
	return target
}

func NumberAsOctal(vm *VM, target, locals Interface, msg *Message) Interface {
	// TODO: zero-fill to nonets
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), 8))
}

func NumberAsString(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewString(target.(*Number).String())
}

func NumberAsUppercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(unicode.ToUpper(rune(target.(*Number).Value))))
}

func NumberAsin(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Asin(target.(*Number).Value))
}

func NumberAt(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) & (1 << uint64(arg.Value))))
}

func NumberAtan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Atan(target.(*Number).Value))
}

func NumberAtan2(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(math.Atan2(target.(*Number).Value, arg.Value))
}

func NumberBetween(vm *VM, target, locals Interface, msg *Message) Interface {
	arg1, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	arg2, err := msg.NumberArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
	}
	v := target.(*Number).Value
	return vm.Bool(arg1.Value <= v && v <= arg2.Value)
}

func NumberBitwiseAnd(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) & int64(arg.Value)))
}

func NumberBitwiseComplement(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(^int64(target.(*Number).Value)))
}

func NumberBitwiseOr(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) | int64(arg.Value)))
}

func NumberBitwiseXor(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) ^ int64(arg.Value)))
}

func NumberCeil(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Ceil(target.(*Number).Value))
}

func NumberClip(vm *VM, target, locals Interface, msg *Message) Interface {
	arg1, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	arg2, err := msg.NumberArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
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

func NumberCos(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Cos(target.(*Number).Value))
}

func NumberCubed(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	return vm.NewNumber(x * x * x)
}

func NumberExp(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Exp(target.(*Number).Value))
}

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

func NumberFloor(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Floor(target.(*Number).Value))
}

func NumberIsAlphaNumeric(vm *VM, target, locals Interface, msg *Message) Interface {
	x := rune(target.(*Number).Value)
	return vm.Bool(unicode.IsLetter(x) || unicode.IsNumber(x))
}

func NumberIsControlCharacter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsControl(rune(target.(*Number).Value)))
}

func NumberIsDigit(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsNumber(rune(target.(*Number).Value)))
}

func NumberIsEven(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(int64(target.(*Number).Value)&1 == 0)
}

func NumberIsGraph(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsGraphic(rune(target.(*Number).Value)))
}

func NumberIsHexDigit(vm *VM, target, locals Interface, msg *Message) Interface {
	x := rune(target.(*Number).Value)
	return vm.Bool(('0' <= x && x <= '9') || ('A' <= x && x <= 'F') || ('a' <= x && x <= 'f'))
}

func NumberIsLetter(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsLetter(rune(target.(*Number).Value)))
}

func NumberIsLowercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsLower(rune(target.(*Number).Value)))
}

func NumberIsNan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(math.IsNaN(target.(*Number).Value))
}

func NumberIsOdd(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(int64(target.(*Number).Value)&1 == 1)
}

func NumberIsPrint(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsPrint(rune(target.(*Number).Value)))
}

func NumberIsPunctuation(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsPunct(rune(target.(*Number).Value)))
}

func NumberIsSpace(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsSpace(rune(target.(*Number).Value)))
}

func NumberIsUppercase(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.Bool(unicode.IsUpper(rune(target.(*Number).Value)))
}

func NumberLog(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Log(target.(*Number).Value))
}

func NumberLog10(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Log10(target.(*Number).Value))
}

func NumberLog2(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Log2(target.(*Number).Value))
}

func NumberMax(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	if target.(*Number).Value >= arg.Value {
		return target
	}
	return arg
}

func NumberMin(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	if target.(*Number).Value <= arg.Value {
		return target
	}
	return arg
}

func NumberMod(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(math.Remainder(target.(*Number).Value, arg.Value))
}

func NumberNegate(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(-target.(*Number).Value)
}

func NumberPow(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(math.Pow(target.(*Number).Value, arg.Value))
}

func NumberRound(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	if x < 0 {
		x = math.Ceil(x - 0.5)
	} else {
		x = math.Floor(x + 0.5)
	}
	return vm.NewNumber(x)
}

func NumberRoundDown(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Floor(target.(*Number).Value + 0.5))
}

func NumberShiftLeft(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) << uint64(arg.Value)))
}

func NumberShiftRight(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(int64(target.(*Number).Value) >> uint64(arg.Value)))
}

func NumberSin(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Sin(target.(*Number).Value))
}

func NumberSqrt(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Sqrt(target.(*Number).Value))
}

func NumberSquared(vm *VM, target, locals Interface, msg *Message) Interface {
	x := target.(*Number).Value
	return vm.NewNumber(x * x)
}

func NumberTan(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(math.Tan(target.(*Number).Value))
}

func NumberToBase(vm *VM, target, locals Interface, msg *Message) Interface {
	arg, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	base := int(arg.Value)
	if base < 2 || base > 36 {
		return vm.NewExceptionf("conversion to base %d not supported")
	}
	return vm.NewString(strconv.FormatInt(int64(target.(*Number).Value), base))
}

func NumberToggle(vm *VM, target, locals Interface, msg *Message) Interface {
	if target.(*Number).Value == 0 {
		return vm.NewNumber(1)
	}
	return vm.NewNumber(0)
}
