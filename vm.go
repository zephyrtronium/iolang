package iolang

import "fmt"

// VM is an object for processing Io programs.
type VM struct {
	// Lobby is the default target of messages.
	Lobby *Object
	// Core is the object containing the basic types of Io.
	Core *Object
	// Addons is the object which will contain imported addons.
	Addons *Object

	// Singletons.
	BaseObject *Object
	True       *Object
	False      *Object
	Nil        *Object
	Operators  *OpTable

	// ValidEncodings is the list of accepted sequence encodings.
	ValidEncodings []string

	// Common numbers and strings to avoid needing new objects for each use.
	NumberMemo map[float64]*Number
	StringMemo map[string]*Sequence
}

// NewVM prepares a new VM to interpret Io code.
func NewVM() *VM {
	vm := VM{
		Lobby: &Object{Slots: Slots{}},

		Core:   &Object{},
		Addons: &Object{},

		BaseObject: &Object{},
		True:       &Object{},
		False:      &Object{},
		Nil:        &Object{},

		Operators: &OpTable{},

		ValidEncodings: []string{"ascii", "utf8", "number", "latin1", "utf16", "utf32"},

		// Memoize all integers in [-1, 255], 1/2, 1/3, 1/4, all mathematical
		// constants defined in package math, +/- inf, and float/int extrema.
		NumberMemo: make(map[float64]*Number, 277),
		// Memoize the empty string and all strings one byte in (UTF8) length.
		StringMemo: make(map[string]*Sequence, 129),
	}

	// There is a specific order for initialization. First, we have to
	// initialize Core, so that other init methods can set up their protos on
	// it. Then, we must initialize CFunction, so that others can use
	// NewCFunction. Following that, we must initialize Sequence, which in turn
	// initializes String, so that we can use NewString. After that, the order
	// matters less, because other types are less likely to be used in inits.
	vm.initCore()
	vm.initCFunction()
	vm.initSequence()
	vm.initMessage()
	vm.initNumber()
	vm.initException()
	vm.initBlock()
	vm.initCall()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()
	vm.initOpTable()
	vm.initLocals()
	vm.initList()
	vm.initFile()
	vm.initCFunction2() // CFunction needs sequences

	vm.MemoizeString("")
	for i := rune(0); i <= 127; i++ {
		vm.MemoizeString(string(i))
	}

	vm.finalInit()

	return &vm
}

// CoreInstance instantiates a type whose default slots are in vm.Core,
// returning an Object with that type as its proto. Panics if there is no such
// type!
func (vm *VM) CoreInstance(name string) *Object {
	// We only want to check vm.Core, not any of its protos (which includes
	// Object, so it would be a long search), so we have to do the lookup
	// manually.
	vm.Core.L.Lock()
	p, ok := vm.Core.Slots[name]
	vm.Core.L.Unlock()
	if ok {
		return &Object{Slots: Slots{}, Protos: []Interface{p}}
	}
	panic("iolang: no Core proto named " + name)
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
			case *Sequence:
				return a.String()
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

// initCore initializes Lobby, Core, and Addons for this VM. This only creates
// room for other init functions to work with.
func (vm *VM) initCore() {
	// Other init* functions will set up Core slots, but it is courteous to
	// make room for them.
	vm.Core.Slots = make(Slots, 32)
	vm.Core.Protos = []Interface{vm.BaseObject}
	lp := &Object{Slots: Slots{"Core": vm.Core, "Addons": vm.Addons}, Protos: []Interface{vm.Core}}
	vm.Lobby.Protos = []Interface{lp}
	SetSlot(vm.Lobby, "Protos", lp)
	SetSlot(vm.Lobby, "Lobby", vm.Lobby)
}

func (vm *VM) finalInit() {
	// Define extras in Io once the VM is capable of executing code.
	const (
		object = `Object setSlot("and", method(v, v isTrue))
		Object setSlot("-", method(v, v negate))`
		false_ = `false setSlot("or", method(v, v isTrue))`
		nil_   = `nil setSlot("or", method(v, v isTrue))`
		// number = `Number do(
		// 	combinations := method(r, self factorial / ((self - r) factorial) / (r factorial))
		// 	permutations := method(r, self factorial / ((self - r) factorial))
		// )`
		// list = `List do(
		// 	first := method(self at(0))
		// 	last  := method(self at(self size - 1))
		// )`
	)
	vm.DoString(object)
	vm.DoString(false_)
	vm.DoString(nil_)
	// vm.DoString(number)
	// vm.DoString(list)
}
