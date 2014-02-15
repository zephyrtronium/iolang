package iolang

type VM struct {
	Lobby *Object

	BaseObject *Object
	True       *Object
	False      *Object
	Nil        *Object

	NumberMemo map[float64]*Number
	StringMemo map[string]*String

	DefaultSlots map[string]Slots
}

// Prepare a new VM to interpret Io code.
func NewVM() *VM {
	vm := VM{
		Lobby: &Object{Slots: Slots{}}, // TODO: what are a vm's slots and protos?

		BaseObject: &Object{},
		True:       &Object{},
		False:      &Object{},
		Nil:        &Object{},

		// Memoize all integers in [-1, 255], all mathematical constants
		// defined in package math, +/- inf, and float/integer extrema.
		NumberMemo: make(map[float64]*Number, 274),
		// Memoize the empty string and all strings one byte in (UTF8) length.
		StringMemo: make(map[string]*String, 129),
	}

	vm.Lobby.Protos = []Interface{vm.Lobby}
	// NOTE: the number here should be >= the number of types
	vm.DefaultSlots = make(map[string]Slots, 9)
	// We have to make CFunction's slots exist first to use NewCFunction.
	vm.DefaultSlots["CFunction"] = Slots{}
	vm.DefaultSlots["Message"] = Slots{}
	vm.initNumber()
	vm.DefaultSlots["Sequence"] = Slots{}
	vm.DefaultSlots["ImmutableSequence"] = Slots{}
	vm.DefaultSlots["Exception"] = Slots{}
	vm.DefaultSlots["Error"] = Slots{}
	vm.initCall()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()

	vm.MemoizeString("")
	for i := rune(0); i <= 127; i++ {
		vm.MemoizeString(string(i))
	}

	return &vm
}

func (vm *VM) MemoizeNumber(v float64) {
	vm.NumberMemo[v] = vm.NewNumber(v)
}

func (vm *VM) MemoizeString(v string) {
	vm.StringMemo[v] = vm.NewString(v)
}

func (vm *VM) Bool(c bool) *Object {
	if c {
		return vm.True
	}
	return vm.False
}

func (vm *VM) AsBool(obj Interface) bool {
	o := obj.SP()
	isTrue, proto := GetSlot(o, "isTrue")
	if proto == nil {
		return true
	}
	switch a := isTrue.(type) {
	case *Object:
		switch a {
		case vm.True:
			return true
		case vm.False, vm.Nil:
			return false
		}
	case Actor:
		// This recursion could be avoided, but I just don't care enough.
		// TODO: provide a Call
		return vm.AsBool(vm.SimpleActivate(a, proto, vm.ObjectWith(Slots{}), "isTrue"))
	}
	return true
}

// Define extras in Io once the VM is capable of executing code.
func (vm *VM) finalInit() {
	const (
		object = `Object do(
			and := method(v, v isTrue)
		)`
		false_ = `false do(
			or := method(v, v isTrue)
		)`
		nil_ = `nil do(
			or := method(v, v isTrue)
		)`
		number = `Number do(
			combinations := method(r, self factorial / ((self - r) factorial) / (r factorial))
			permutations := method(r, self factorial / ((self - r) factorial))
		)`
	)
	vm.DoString(object)
	vm.DoString(false_)
	vm.DoString(nil_)
	vm.DoString(number)
}
