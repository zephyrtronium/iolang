package iolang

import "fmt"

// VM is an object for processing Io programs.
type VM struct {
	// Lobby is the default target of messages.
	Lobby *Object

	// Singletons.
	BaseObject *Object
	True       *Object
	False      *Object
	Nil        *Object
	Operators  *OpTable

	// Common numbers and strings to avoid needing new objects for each use.
	NumberMemo map[float64]*Number
	StringMemo map[string]*String

	// Sets of default slots for new objects of any type.
	// Addons might want to use this to hold their slots.
	DefaultSlots map[string]Slots
}

// NewVM prepares a new VM to interpret Io code.
func NewVM() *VM {
	vm := VM{
		Lobby: &Object{Slots: Slots{}}, // TODO: what are a vm's slots and protos?

		BaseObject: &Object{},
		True:       &Object{},
		False:      &Object{},
		Nil:        &Object{},

		Operators: &OpTable{},

		// Memoize all integers in [-1, 255], 1/2, 1/3, 1/4, all mathematical
		// constants defined in package math, +/- inf, and float/int extrema.
		NumberMemo: make(map[float64]*Number, 277),
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
	vm.initLocals()

	vm.MemoizeString("")
	for i := rune(0); i <= 127; i++ {
		vm.MemoizeString(string(i))
	}

	return &vm
}

// MemoizeNumber creates a quick-access Number with the given value.
func (vm *VM) MemoizeNumber(v float64) {
	vm.NumberMemo[v] = vm.NewNumber(v)
}

// MemoizeString creates a quick-access String with the given value.
func (vm *VM) MemoizeString(v string) {
	vm.StringMemo[v] = vm.NewString(v)
}

// IoBool converts a bool to the appropriate Io boolean object.
func (vm *VM) IoBool(c bool) *Object {
	if c {
		return vm.True
	}
	return vm.False
}

// AsBool attempts to convert an Io object to a bool by activating its isTrue
// slot. If the object has no such slot, it is true.
func (vm *VM) AsBool(obj Interface) bool {
	if obj == nil {
		obj = vm.Nil
	}
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

// AsString attempts to convert an Io object to a string by activating its
// asString slot. If the object has no such slot but is an fmt.Stringer, then
// it returns the value of String(); otherwise, a default representation is
// used. If the asString method returns an error, then the error message is the
// return value.
func (vm *VM) AsString(obj Interface) string {
	if obj == nil {
		obj = vm.Nil
	}
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
	return fmt.Sprintf("%T_%p", obj, obj)
}

func (vm *VM) finalInit() {
	// Define extras in Io once the VM is capable of executing code.
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
