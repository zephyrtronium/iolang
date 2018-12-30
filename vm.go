package iolang

import (
	"fmt"
	"time"
)

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

	// StartTime is the time at which VM initialization began, used for the
	// Date clock method.
	StartTime time.Time

	// Common numbers and strings to avoid needing new objects for each use.
	NumberMemo map[float64]*Number
	StringMemo map[string]*Sequence
}

// NewVM prepares a new VM to interpret Io code. String arguments may be passed
// to occupy the System args slot, typically os.Args[1:].
func NewVM(args ...string) *VM {
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

		// TODO: should this be since program start instead to match Io?
		StartTime: time.Now(),

		// Memoize all integers in [-1, 255], 1/2, 1/3, 1/4, all mathematical
		// constants defined in package math, +/- inf, and float/int extrema.
		NumberMemo: make(map[float64]*Number, 277),
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
	vm.initDirectory()
	vm.initDate()
	vm.initDuration()
	vm.initSystem()
	vm.initArgs(args)
	vm.initMap()
	vm.initCFunction2() // CFunction needs sequences
	vm.initCollector()

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
	r := vm.SimpleActivate(isTrue, obj, obj, "isTrue")
	if a, ok := r.(*Object); ok {
		if a == vm.False || a == vm.Nil {
			return false
		}
	}
	return true
}

// AsString attempts to convert an Io object to a string by activating its
// asString slot. If the object has no such slot but is an fmt.Stringer, then
// it returns the value of String(); otherwise, a default representation is
// used. If the asString method raises an exception, then the exception message
// is the return value.
func (vm *VM) AsString(obj Interface) string {
	if obj == nil {
		obj = vm.Nil
	}
	if asString, proto := GetSlot(obj, "asString"); proto != nil {
		obj = vm.SimpleActivate(asString, obj, obj, "asString")
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
	vm.MustDoString(finalInitCode)
}

const finalInitCode = `
Object do(
	print := method(File standardOutput write(self asString); self)
	println := method(File standardOutput write(self asString, "\n"); self)
	// Use setSlot directly to circumvent operator shuffling.
	setSlot("and", method(v, v isTrue))
	setSlot("-", method(v, v negate))
	setSlot("..", method(v, getSlot("self") asString .. v asString))

	ancestors := method(a,
		if(a,
			if(a containsIdenticalTo(getSlot("self")), return a)
		,
			a = List clone
		)
		a append(getSlot("self"))
		getSlot("self") protos foreach(ancestors(a))
		a
	)
	isKindOf := method(proto,
		// Lazy method, building the entire list of ancestors.
		getSlot("self") ancestors contains(proto)
	)

	proto := method(protos first)
	hasProto := getSlot("isKindOf")

	hasSlot := method(slot, hasLocalSlot(slot) or ancestorWithSlot(slot) != nil)
	setSlotWithType := method(slot, value,
		setSlot(slot, value)
		value type := slot
	)

	isActivatable := false
	setIsActivatable := method(v, getSlot("self") isActivatable := v; self)

	asBoolean := true
)

Sequence do(
	setSlot("..", method(v, self asString cloneAppendSeq(v asString)))
	setSlot("*", method(v, Sequence clone copy(self) *= v))
	setSlot("**", method(v, Sequence clone copy(self) **=(v)))
	setSlot("+", method(v, Sequence clone copy(self) += v))
	setSlot("-", method(v, Sequence clone copy(self) -= v))
	setSlot("/", method(v, Sequence clone copy(self) /= v))
)

Exception do(
	catch := method(proto, if(self isKindOf(proto), call evalArgAt(1); nil, self))
)

false do(
	setSlot("or",  method(v, v isTrue))
	asBoolean := false
)

nil do(
	setSlot("or",  method(v, v isTrue))
	catch := nil
	asBoolean := nil
)

Number do(
	combinations := method(r, self factorial / ((self - r) factorial) / (r factorial))
	permutations := method(r, self factorial / ((self - r) factorial))
)

List do(
	first := method(at(0))
	second := method(at(1))
	third := method(at(2))
	last := method(at(size - 1))
	rest := method(slice(1))
	isEmpty := method(size == 0)
	isNotEmpty := method(size > 0)

	copy := method(l, empty appendSeq(l))
	itemCopy := method(List clone copy(self))
	sort := method(List clone copy(self) sortInPlace)
	sortBy := method(b, List clone copy(self) sortInPlaceBy(getSlot("b")))
	reverse := method(List clone copy(self) reverseInPlace)

	unique := method(
		l := List clone
		foreach(v, l appendIfAbsent(v))
	)
	uniqueCount := method(unique map(v, list(v, select(== item) size)))

	intersect := method(l, l select(v, contains(v)))
	difference := method(l, select(v, l contains(v) not))
	union := method(l, itemCopy appendSeq(l difference(self)))

	reduce := method(
		argc := call argCount
		if(argc == 0, Exception raise("List reduce must be called with 1 to 4 arguments"))
		if(argc == 2 or argc == 4,
			l := self
			acc := call sender doMessage(call argAt(argc - 1))
		,
			l := slice(1)
			acc := at(0)
		)

		if(argc <= 2,
			args := list(nil)
			m := call argAt(0) name
			l foreach(x, acc = acc performWithArgList(m, args atPut(0, x)))
		,
			accName := call argAt(0) name
			xName := call argAt(1) name
			m := call argAt(2)
			ctxt := Locals clone prependProto(call sender)
			if(call sender hasLocalSlot("self"),
				ctxt setSlot("self", call sender self)
			)
			l foreach(x,
				ctxt setSlot(accName, acc)
				ctxt setSlot(xName, x)
				acc = ctxt doMessage(m)
			)
		)
		acc
	)
	reverseReduce := method(
		argc := call argCount
		if(argc == 0, Exception raise("List reverseReduce must be called with 1 to 4 arguments"))
		if(argc == 2 or argc == 4,
			l := self
			acc := call sender doMessage(call argAt(argc - 1))
		,
			l := slice(0, -1)
			acc := self at(self size - 1)
		)

		if(argc <= 2,
			args := list(nil)
			m := call argAt(0) name
			l reverseForeach(x, acc = acc performWithArgList(m, args atPut(0, x)))
		,
			accName := call argAt(0) name
			xName := call argAt(1) name
			m := call argAt(2)
			ctxt := Locals clone prependProto(call sender)
			if(call sender hasLocalSlot("self"),
				ctxt setSlot("self", call sender self)
			)
			l reverseForeach(x,
				ctxt setSlot(accName, acc)
				ctxt setSlot(xName, x)
				acc = ctxt doMessage(m)
			)
		)
		acc
	)

	selectInPlace := method(
		ctxt := Locals clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List selectInPlace requires 1 to 3 arguments"))
		d := 0
		if(argc == 1) then (
			m := call argAt(0)
			size repeat(k,
				if(at(k - d) doMessage(m, ctxt) not,
					removeAt(k - d)
					d = d + 1
				)
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			size repeat(k,
				v := at(k - d)
				ctxt setSlot(vn, v)
				if(ctxt doMessage(m) not,
					removeAt(k - d)
					d = d + 1
				)
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			size repeat(k,
				v := at(k - d)
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, v)
				if(ctxt doMessage(m) not,
					removeAt(k - d)
					d = d + 1
				)
			)
		)
		self
	)
	select := method(
		ctxt := Locals clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List select requires 1 to 3 arguments"))
		l := List clone preallocateToSize(size)
		if(argc == 1) then (
			m := call argAt(0)
			foreach(v,
				if(v doMessage(m, ctxt),
					l append(v)
				)
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			foreach(v,
				ctxt setSlot(vn, v)
				if(ctxt doMessage(m),
					l append(v)
				)
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, v)
				if(ctxt doMessage(m),
					l append(v)
				)
			)
		)
		l
	)

	detect := method(
		ctxt := Locals clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List detect requires 1 to 3 arguments"))
		if(argc == 1) then (
			m := call argAt(0)
			foreach(v,
				if(getSlot("v") doMessage(m, ctxt),
					return getSlot("v")
				)
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			foreach(v,
				ctxt setSlot(vn, getSlot("v"))
				if(ctxt doMessage(m),
					return getSlot("v")
				)
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, getSlot("v"))
				if(ctxt doMessage(m),
					return getSlot("v")
				)
			)
		)
		nil
	)

	mapInPlace := method(
		ctxt := Locals clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List mapInPlace requires 1 to 3 arguments"))
		if(argc == 1) then (
			m := call argAt(0)
			foreach(k, v,
				atPut(k, getSlot("v") doMessage(m, ctxt))
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			foreach(k, v,
				ctxt setSlot(vn, getSlot("v"))
				atPut(k, ctxt doMessage(m))
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, getSlot("v"))
				atPut(k, ctxt doMessage(m))
			)
		)
		self
	)
	map := method(
		List clone copy(self) doMessage(call message clone setName("mapInPlace"))
	)

	groupBy := method(
		ctxt := Locals clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List groupBy requires 1 to 3 arguments"))
		r := Map clone
		if(argc == 1) then (
			m := call argAt(0)
			foreach(v,
				r atIfAbsentPut(getSlot("v") doMessage(m, ctxt) asString, List clone) append(getSlot("v"))
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			foreach(v,
				ctxt setSlot(vn, getSlot("v"))
				r atIfAbsentPut(ctxt doMessage(m) asString, List clone) append(getSlot("v"))
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, getSlot("v"))
				r atIfAbsentPut(ctxt doMessage(m) asString, List clone) append(getSlot("v"))
			)
		)
		r
	)

	sum := method(reduce(+))
	average := method(sum / size)

	removeLast := pop := method(if(isNotEmpty, removeAt(size - 1), nil))
	removeFirst := method(if(isNotEmpty, removeAt(0), nil))
	removeSeq := method(s, s foreach(v, self remove(getSlot("v"))))

	join := method(sep,
		r := Sequence clone
		sep ifNonNil(
			n := size - 1
			foreach(k, v,
				r appendSeq(v)
				if(k < n, r appendSeq(sep))
			)
		) ifNil(
			foreach(v, r appendSeq(v))
		)
		r
	)

	insertBefore := method(v, before,
		k := indexOf(before)
		if(k, atInsert(k, v), append(v))
	)
	insertAfter := method(v, after,
		k := indexOf(after)
		if(k, atInsert(k + 1, v), append(v))
	)
	insertAt := method(v, k, atInsert(k, v))

	min := method(
		m := call argAt(0)
		r := first
		m ifNil(
			foreach(v, if(v < r, r := v))
		) ifNonNil(
			foreach(v,
				x := getSlot("r") doMessage(m, call sender)
				y := getSlot("v") doMessage(m, call sender)
				if(y < x, r := y)
			)
		)
		r
	)
	max := method(
		m := call argAt(0)
		r := first
		m ifNil(
			foreach(v, if(v > r, r := v))
		) ifNonNil(
			foreach(v,
				x := getSlot("r") doMessage(m, call sender)
				y := getSlot("v") doMessage(m, call sender)
				if(y > x, r := y)
			)
		)
		r
	)

	asMessage := method(
		m := Message clone
		foreach(v, m setArguments(m arguments append(Message clone setCachedResult(getSlot("v")))))
	)

	asSimpleString := method(
		r := slice(0, 30) mapInPlace(asSimpleString) asString
		if(r size > 40, r exSlice(0, 37) .. "...", r)
	)

	asMap := method(
		m := Map clone
		foreach(pair, m atPut(pair at(0), pair at(1)))
	)
)

Directory do(
	size := method(self items size)
)

Map do(
	hasValue := method(value, self values contains(value))
)

Date do(
	setSlot("+", method(dur, self clone += dur))
)

Duration do(
	setSlot("+", method(other, self clone += other))
	setSlot("-", method(other, self clone -= other))
)
`
