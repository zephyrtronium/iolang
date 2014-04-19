package iolang

import "fmt"

type VM struct {
	Lobby *Object

	BaseObject *Object
	True       *Object
	False      *Object
	Nil        *Object

	Operators *OpTable

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

		Operators: &OpTable{},

		// Memoize all integers in [-1, 255], all mathematical constants
		// defined in package math, +/- inf, and float/integer extrema.
		NumberMemo: make(map[float64]*Number, 274),
		// Memoize the empty string and all strings one byte in (UTF8) length.
		StringMemo: make(map[string]*String, 129),
	}

	vm.Lobby.Protos = []Interface{vm.BaseObject}
	// NOTE: the number here should be >= the number of types
	vm.DefaultSlots = make(map[string]Slots, 20)
	// We have to make CFunction's slots exist first to use NewCFunction.
	vm.DefaultSlots["CFunction"] = Slots{}
	vm.initMessage()
	vm.initNumber()
	vm.DefaultSlots["Sequence"] = Slots{}
	vm.DefaultSlots["ImmutableSequence"] = Slots{}
	vm.DefaultSlots["Exception"] = Slots{}
	vm.DefaultSlots["Error"] = Slots{}
	vm.initBlock()
	vm.initCall()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()
	vm.initOpTable()

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

func (vm *VM) AsString(obj Interface) string {
	if asString, proto := GetSlot(obj, "asString"); proto != nil {
		for {
			switch a := asString.(type) {
			case Actor:
				asString = vm.SimpleActivate(a, obj, obj, "asString")
				continue
			case *String:
				return a.Value
			}
			if IsIoError(asString) {
				return asString.(error).Error()
			}
			break
		}
	}
	if s, ok := obj.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%T_%p", obj)
}

// Define extras in Io once the VM is capable of executing code.
func (vm *VM) finalInit() {
	const (
		object = `Object setSlot("and", method(v, v isTrue))`
		false_ = `false setSlot("or", method(v, v isTrue))`
		nil_   = `nil setSlot("or", method(v, v isTrue))`
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
