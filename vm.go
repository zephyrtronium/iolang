package iolang

import (
	"fmt"
	"time"
)

// VM is an object for processing Io programs.
type VM struct {
	// The VM is an object because it also represents a coroutine.
	Object

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
	Operators  *Object

	// Sched is the scheduler for this VM and all related coroutines.
	Sched *Scheduler
	// Stop is a buffered channel for remote control of this coroutine. The
	// evaluator checks this between each message. Stops with NoStop status
	// tell the coroutine to yield, ones with ExceptionStop status are returned
	// directly, and all other statuses cause the current evaluated result to
	// be returned.
	Stop chan Stop

	// StartTime is the time at which VM initialization began, used for the
	// Date clock method.
	StartTime time.Time

	// Common numbers to avoid needing new objects for each use.
	NumberMemo map[float64]*Number
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

		Stop: make(chan Stop, 1),

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
	// initializes String, so that we can use NewString. After that, we must
	// have Map before OpTable and OpTable before Object. Lastly, we must have
	// a scheduler in order to evaluate Io statements.
	vm.initCore()
	vm.initCFunction()
	vm.initSequence()
	vm.initMessage()
	vm.initNumber()
	vm.initException()
	vm.initBlock()
	vm.initCall()
	vm.initMap()
	vm.initOpTable()
	vm.initObject()
	vm.initTrue()
	vm.initFalse()
	vm.initNil()
	vm.initLocals()
	vm.initList()
	vm.initFile()
	vm.initDirectory()
	vm.initDate()
	vm.initDuration()
	vm.initSystem()
	vm.initArgs(args)
	vm.initCollector()
	vm.initScheduler()
	vm.initCoroutine()
	vm.initFuture()

	vm.finalInit()

	return &vm
}

// Activate returns the VM (coroutine).
func (vm *VM) Activate(vm2 *VM, target, locals, context Interface, msg *Message) Interface {
	return vm
}

// Clone creates a new, inactive coroutine cloned from this one.
func (vm *VM) Clone() Interface {
	nv := VM{
		Object:     Object{Slots: Slots{}, Protos: []Interface{vm}},
		Lobby:      vm.Lobby,
		Core:       vm.Core,
		Addons:     vm.Addons,
		BaseObject: vm.BaseObject,
		True:       vm.True,
		False:      vm.False,
		Nil:        vm.Nil,
		Operators:  vm.Operators,
		Sched:      vm.Sched,
		Stop:       make(chan Stop, 1),
		StartTime:  vm.StartTime,
		NumberMemo: vm.NumberMemo,
	}
	return &nv
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
	print := method(File standardOutput write(getSlot("self") asString); getSlot("self"))
	println := method(File standardOutput write(getSlot("self") asString, "\n"); getSlot("self"))
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
	newSlot := method(slot, value,
		getSlot("self") setSlot(slot, getSlot("value"))
		getSlot("self") setSlot("set" .. slot asCapitalized,
			doString("method(" .. slot .. " = call evalArgAt(0); self)"))
		getSlot("value")
	)
	lazySlot := method(
		if(call argCount == 1,
			m := method(self setSlot(call message name, nil))
			args := getSlot("m") message next arguments
			args atPut(1, call argAt(0) clone)
			getSlot("m") message next setArguments(args)
			getSlot("m") clone
		,
			name := call evalArgAt(0)
			m := "self setSlot(\"#{name}\", #{call argAt(1) asString})" interpolate asMessage
			self setSlot(name, method() setMessage(m))
			nil
		)
	)

	isActivatable := false
	setIsActivatable := method(v, getSlot("self") isActivatable := v; self)

	asBoolean := true

	addTrait := method(v,
		if(call argCount == 0, Exception raise("Object addTrait requires one or two arguments"))
		res := call evalArgAt(1) ifNilEval(Map clone)
		getSlot("v") foreachSlot(k, v,
			if(getSlot("self") hasLocalSlot(k),
				if(k == "type", continue)
				res at(k) ifNil(Exception raise("Slot " .. k .. " already exists"))
				getSlot("self") setSlot(res at(k), getSlot("v"))
			,
				getSlot("self") setSlot(k, getSlot("v"))
			)
		)
		getSlot("self")
	)

	slotDescriptionMap := method(
		slots := getSlot("self") slotNames sortInPlace
		descs := slots map(slot, getSlot("self") getSlot(slot) asSimpleString)
		Map clone addKeysAndValues(slots, descs)
	)
	apropos := method(kw,
		Core foreachSlot(slot, p,
			descs := getSlot("p") slotDescriptionMap ?select(k, v, k asMutable lowercase containsSeq(kw))
			if(descs and descs size > 0,
				s := Sequence clone
				descs keys sortInPlace foreach(k,
					s appendSeq("  ", k alignLeft(16), " = ", descs at(k), "\n")
				)
				slot println
				s println
			)
		)
		nil
	)
	slotSummary := method(kw,
		// We should use isKindOf(block), but that's actually much slower.
		if(type == "Block", return asSimpleString)
		s := Sequence clone appendSeq(" ", asSimpleString, ":\n")
		descs := slotDescriptionMap
		kw ifNonNil(descs = descs select(k, v, k asMutable lowercase containsSeq(kw)))
		descs keys sortInPlace foreach(k,
			s appendSeq("  ", k alignLeft(16), " = ", descs at(k), "\n")
		)
		s
	)
	asSimpleString := method(getSlot("self") type .. "_" .. getSlot("self") uniqueId)

	setSlot("?", method(
			m := call argAt(0)
			self getSlot(m name) ifNonNilEval(m doInContext(self, call sender))
		) setPassStops(true)
	)

	super := method(
		sc := call sender call slotContext ifNil(Exception raise("Object super called outside a block"))
		m := call argAt(0) ifNil(Exception raise("Object super requires an argument"))
		name := m name
		a := sc ancestorWithSlot(name) ifNilEval(sc ancestorWithSlot(name = "forward"))
		if(a isIdenticalTo(sc), Exception raise("super slot " .. name .. " not found"))
		b := a getSlot(name)
		if(getSlot("b") isActivatable == false, b, getSlot("b") performOn(call sender call target, call sender, m, a))
	)
	resend := method(
		sc := call sender call slotContext ifNil(Exception raise("Object super called outside a block"))
		m := call argAt(0) ifNil(Exception raise("Object super requires an argument"))
		name := m name
		a := sc ancestorWithSlot(name) ifNilEval(sc ancestorWithSlot(name = "forward"))
		if(a isIdenticalTo(sc), Exception raise("super slot " .. name .. " not found"))
		b := a getSlot(name)
		getSlot("b") ifNonNilEval(getSlot("b") performOn(call sender getSlot("self"), call sender call sender, m, a))
	)

	in := method(l, l contains(self))

	switch := method(
		m := for(case, 0, call argCount - 2, 2,
			// We can't return here because this method passes stops.
			if(call evalArgAt(case) == self, break(call argAt(case + 1)))
		)
		if(m not,
			if(call argCount isOdd,
				call evalArgAt(call argCount - 1)
			,
				nil
			)
		,
			doMessage(m)
		)
	) setPassStops(true)

	isLaunchScript := method(call message label == System launchScript)

	// These won't work until we implement Path and Sequence pathComponent.
	relativeDoFile := doRelativeFile := method(p,
		self doFile(Path with(call message label pathComponent, p))
	)
	
	yield := method(Coroutine currentCoroutine yield)
	pause := method(Coroutine currentCoroutine pause)
)

Call do(
	description := method(
		m := self message
		s := self target type .. " " .. m name
		s alignLeft(36) .. " " .. m label ?(lastPathComponent) .. " " .. m lineNumber
	)

	delegateTo := method(target, ns,
		target doMessage(self message clone setNext, ns ifNilEval(self sender))
	) setPassStops(true)
	delegateToMethod := method(target, name,
		target doMessage(self message clone setNext setName(name), self sender)
	) setPassStops(true)

	hasArgs := method(argCount > 0)
	evalArgs := method(self message argsEvaluatedIn(self sender)) setPassStops(true)
)

Sequence do(
	asSimpleString := method("\"" .. self asString asMutable escape .. "\"")

	setSlot("..", method(v, self asString cloneAppendSeq(v asString)))
	setSlot("*", method(v, Sequence clone copy(self) *= v))
	setSlot("**", method(v, Sequence clone copy(self) **=(v)))
	setSlot("+", method(v, Sequence clone copy(self) += v))
	setSlot("-", method(v, Sequence clone copy(self) -= v))
	setSlot("/", method(v, Sequence clone copy(self) /= v))
	rootMeanSquare := method(meanSquare sqrt)

	isEmpty := method(self size == 0)
	isSymbol := method(self isMutable not)

	itemCopy := method(Sequence clone copy(self))

	alignLeftInPlace := method(w, pad,
		os := size
		if(pad isNil or pad size == 0, pad = " ")
		((w - size) / pad size) ceil repeat(appendSeq(pad))
		setSize(w max(os))
	)
	alignLeft := method(w, pad, asMutable alignLeftInPlace(w, pad))
	alignRight := method(w, pad, Sequence clone alignLeftInPlace(w - size, pad) appendSeq(self))
	alignCenter := method(w, pad, alignRight(((size + w)/2) floor, pad) alignLeftInPlace(w, pad))

	asCapitalized := method(if(isMutable, capitalize, asMutable capitalize asSymbol))
	asLowercase := method(asMutable lowercase asSymbol)
	asUppercase := method(asMutable uppercase asSymbol)

	containsAnyCaseSeq := method(s, self asLowercase containsSeq(s asLowercase))
	isEqualAnyCase := method(s, if(self size == s size, containsAnyCaseSeq(s)))

	linePrint := method(File standardOutput write(self, "\n"); self)

	asFile := method(File with(self))
	fileName := method(
		isEmpty ifTrue(return self)
		p := lastPathComponent split(".")
		if(p size > 1, p removeLast)
		p join(".")
	)
	stringByExpandingTilde := method(split("~") join(tildeExpandsTo))

	interpolateInPlace := method(self copy(self interpolate))
	makeFirstCharacterLowercase := method(if(size > 0, atPut(0, at(0) asLowercase)))
	makeFirstCharacterUppercase := method(if(size > 0, atPut(0, at(0) asUppercase)))
	prependSeq := method(self atInsertSeq(0, call evalArgs join); self)
	replaceMap := method(m, m foreach(k, v, self replaceSeq(k, v)))
	setItemsToLong := method(x, self setItemsToDouble(x roundDown))

	asHex := method(
		s := Sequence clone
		self foreach(r, s appendSeq(r asHex))
	)
	findNthSeq := method(s, n,
		k := findSeq(s)
		k ifNil(return nil)
		if(n == 1, return k)
		k + self exSlice(k + 1, self size) findNthSeq(s, n - 1)
	)
	orderedSplit := method(
		seps := call evalArgs
		if(seps size == 0, return list(self))
		i := 0
		x := 0
		r := list()
		seps foreach(sep,
			j := findSeq(sep, i) ifNil(
				x = x + 1
				continue
			)
			r append(exSlice(i, j))
			if(x > 0, x repeat(r append(nil)); x = 0)
			i = j + sep size
		)
		if(size == 0, r append(nil), r append(exSlice(i)))
		x repeat(r append(nil))
		r
	)
	repeated := method(n,
		s := Sequence clone
		n repeat(s appendSeq(self))
	)
	reverse := method(self asMutable reverseInPlace)
	sizeInBytes := method(size * itemSize)
	slicesBetween := method(start, end,
		l := list()
		k := 0
		while(a := self findSeq(start, k),
			b := self findSeq(end, k + start size)
			b ifNil(break)
			l append(self exSlice(a + start size, b))
			k = b + end size
		)
		l
	)
	splitNoEmpties := method(self performWithArgList("split", call evalArgs) selectInPlace(size != 0))
	with := method(
		s := Sequence clone performWithArgList("append", call evalArgs)
		if(self isMutable, s, s asSymbol)
	)

	sequenceSets := Map clone do(
		atPut("lowercaseSequence", "abcdefghijklmnopqrstuvwxyz" asList)
		atPut("uppercaseSequence", "ABCDEFGHIJKLMNOPQRSTUVWXYZ" asList)
		atPut("digitSequence", "0123456789" asList)
	)
	whiteSpaceStrings := list("\t", "\n", "\v", "\f", "\r", " ", "\x85", "\xa0",
		"\u1680", "\u2000", "\u2001", "\u2002", "\u2003", "\u2004", "\u2005", "\u2006",
		"\u2007", "\u2008", "\u2009", "\u200a", "\u2028", "\u2029", "\u202f", "\u205f", "\u3000"
	)
	validEncodings := "utf8 utf16 utf32 latin1 number ascii" split
	validItemTypes := "uint8 uint16 uint32 uint64 int8 int16 int32 int64 float32 float64" split

	x := method(at(0))
	y := method(at(1))
	z := method(at(2))
	setX := method(v, atPut(0, v); self)
	setY := method(v, atPut(1, v); self)
	setZ := method(v, atPut(2, v); self)
	set := method(call evalArgs foreach(i, v, atPut(i, v)))
)

Scheduler do(
	currentCoroutine := Coroutine getSlot("currentCoroutine")
	waitForCorosToComplete := method(while(yieldingCoros size > 0, yield))
)

Coroutine do(
	parentCoroutine ::= nil
	runMessage ::= nil
	runTarget ::= nil
	runLocals ::= nil
	exception ::= nil
	result ::= nil

	yieldingCoros := Scheduler getSlot("yieldingCoros")

	label := method(self uniqueId)
	setLabel := method(s, self label = s .. "_" .. self uniqueId)

	showYielding := method(s,
		File standardOutput writeln("   ", label, " ", s)
		yieldingCoros foreach(v, File standardOutput writeln("    ", v label))
	)

	isYielding := method(yieldingCoros contains(self))

	main := method(setResult(self getSlot("runTarget") doMessage(runMessage, self getSlot("runLocals"))))
)

Exception do(
	caughtMessage ::= nil
	coroutine ::= nil
	error ::= nil
	nestedException ::= nil
	originalCall ::= nil

	showStack := method(
		self stack reverseForeach(m,
			m previous ifNil(
				File standardOutput write("\t", m name, "\t", m label, ":", m lineNumber asString, "\n")
			) ifNonNil(
				File standardOutput write("\t", m previous name, " ", m name, "\t", m label, ":", m lineNumber asString, "\n")
			)
		)
	)

	catch := method(proto, if(self isKindOf(proto), call evalArgAt(1); nil, self))
)

Core Error := Object clone do(
	ifError := method(
		if(call argCount == 1,
			call evalArgAt(0)
		) elseif(call argCount > 1,
			call sender setSlot(call message argAt(0) name, self)
			call evalArgAt(1)
		) else(
			Exception raise("ifError requires 1 or 2 arguments")
		)
		self
	) setPassStops(true)

	returnIfError := method(call sender return self) setPassStops(true)
	raiseIfError := method(Exception raise(message))

	with := method(msg,
		err := self clone
		err message := msg
		err location := call message label .. ":" .. call message lineNumber
		err
	)
	withShow := method(msg,
		("ERROR: " .. msg) println
		with(msg)
	)

	isError := true
)

false do(
	setSlot("or",  method(v, v isTrue))
	asBoolean := false
)

nil do(
	setSlot("or",  method(v, v isTrue))
	catch := nil
	pass := nil
	asBoolean := nil
)

Number do(
	combinations := method(k,
		n := self + 1
		k = k min(self - k)
		v := 1
		for(i, 1, k, v = v * (n - i) / i)
		v
	)
	permutations := method(k,
		v := 1
		for(i, 0, k-1, v = v * (self - i))
		v
	)

	asHex := method(toBaseWholeBytes(16))
	asBinary := method(toBaseWholeBytes(2))
	asOctal := method(toBaseWholeBytes(8))

	// This won't work until Sequence sequenceSets exists. (それは何ですか)
	isInASequenceSet := method(
		Sequence sequenceSets foreach(set, if(in(set), return true))
		false
	)
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
			ctxt := Object clone prependProto(call sender)
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
			ctxt := Object clone prependProto(call sender)
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
		ctxt := Object clone prependProto(call sender)
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
		ctxt := Object clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List select requires 1 to 3 arguments"))
		l := List clone preallocateToSize(size)
		if(argc == 1) then (
			m := call argAt(0)
			self foreach(v,
				if(getSlot("v") doMessage(m, ctxt),
					l append(getSlot("v"))
				)
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			self foreach(v,
				ctxt setSlot(vn, getSlot("v"))
				if(ctxt doMessage(m),
					l append(getSlot("v"))
				)
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			self foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, getSlot("v"))
				if(ctxt doMessage(m),
					l append(getSlot("v"))
				)
			)
		)
		l
	)

	detect := method(
		ctxt := Object clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List detect requires 1 to 3 arguments"))
		if(argc == 1) then (
			m := call argAt(0)
			self foreach(v,
				if(getSlot("v") doMessage(m, ctxt),
					return getSlot("v")
				)
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			self foreach(v,
				ctxt setSlot(vn, getSlot("v"))
				if(ctxt doMessage(m),
					return getSlot("v")
				)
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			self foreach(k, v,
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
		ctxt := Object clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List mapInPlace requires 1 to 3 arguments"))
		if(argc == 1) then (
			m := call argAt(0)
			self foreach(k, v,
				atPut(k, getSlot("v") doMessage(m, ctxt))
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			self foreach(k, v,
				ctxt setSlot(vn, getSlot("v"))
				atPut(k, ctxt doMessage(m))
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			self foreach(k, v,
				ctxt setSlot(kn, k)
				ctxt setSlot(vn, getSlot("v"))
				atPut(k, ctxt doMessage(m))
			)
		)
		self
	)
	map := method(
		List clone copy(self) doMessage(call message clone setName("mapInPlace"), call sender)
	)

	groupBy := method(
		ctxt := Object clone prependProto(call sender)
		if(call sender hasLocalSlot("self"),
			ctxt setSlot("self", call sender self)
		)
		argc := call argCount
		if(argc == 0, Exception raise("List groupBy requires 1 to 3 arguments"))
		r := Map clone
		if(argc == 1) then (
			m := call argAt(0)
			self foreach(v,
				r atIfAbsentPut(getSlot("v") doMessage(m, ctxt) asString, List clone) append(getSlot("v"))
			)
		) elseif(argc == 2) then (
			vn := call argAt(0) name
			m := call argAt(1)
			self foreach(v,
				ctxt setSlot(vn, getSlot("v"))
				r atIfAbsentPut(ctxt doMessage(m) asString, List clone) append(getSlot("v"))
			)
		) else (
			kn := call argAt(0) name
			vn := call argAt(1) name
			m := call argAt(2)
			self foreach(k, v,
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

	asJson := method("[" .. self map(asJson) join(",") .. "]")

	asMap := method(
		m := Map clone
		foreach(pair, m atPut(pair at(0), pair at(1)))
	)

	ListCursor := Object clone do(
		index ::= 0
		collection ::= nil
		next := method(
			index = index + 1
			max := collection size - 1
			if(index > max, index = max; false, true)
		)
		previous := method(
			index = index - 1
			if(index < 0, index = 0; false, true)
		)
		value := method(collection at(index))
		insert := method(v, collection atInsert(index, getSlot("v")))
		remove := method(collection removeAt(index))
	)
	cursor := method(ListCursor clone setCollection(self))

	sortKey := method(
		schw := call activated SchwartzianList clone
		if(call argCount == 1,
			body := call argAt(0)
			foreach(val,
				k := val doMessage(body, call sender)
				schw addPair(k, val)
			)
		,
			valName := call argAt(0) name
			foreach(val,
				call sender setSlot(valName, val)
				k := call evalArgAt(1)
				schw addPair(k, val)
			)
		)
		schw
	) do(
		SchwartzianList := Object clone do(
			pairs ::= nil
			init := method(pairs = list())

			SchwartzianPair := Object clone do(
				key ::= nil
				value ::= nil
				asSimpleString := method("(#{key asSimpleString}: #{value asSimpleString}" interpolate)
			)

			addPair := method(k, v, pairs append(SchwartzianPair clone setKey(k) setValue(v)))

			sort := method(
				if(call argCount == 0,
					pairs sortBy(block(x, y, x key < y key))
				,
					if(call argCount == 1,
						op := call argAt(0) name
						args := list(nil)
						pairs sortBy(block(x, y, x key performWithArgList(op, args atPut(0, y key))))
					,
						xn := call argAt(0) name
						yn := call argAt(1) name
						sc := call
						pairs sortBy(
							block(x, y,
								sc sender setSlot(xn, x key)
								sc sender setSlot(yn, y key)
								sc evalArgAt(2)
							)
						)
					)
				) mapInPlace(value)
			)
		)
	)
	sortByKey := method(call delegateToMethod(self, "sortKey") sort)
)

File do(
	with := method(p, self clone setPath(p))

	streamReadSize := 65536
	streamTo := method(dst,
		b := Sequence clone
		open
		while(isAtEnd not,
			b empty
			readToBufferLength(b, streamReadSize)
			dst write(b)
			?yield
		)
	)
	streamToWithoutYielding := method(dst,
		b := Sequence clone
		open
		while(isAtEnd not,
			b empty
			readToBufferLength(b, streamReadSize)
			dst write(b)
		)
	)

	copyToPath := method(p,
		dst := File with(p) open
		open streamTo(dst)
		dst close
		close
	)
	copyToPathWithoutYielding := method(p,
		dst := File with(p) open
		open streamToWithoutYielding(dst)
		dst close
		close
	)

	setContents := method(v, truncateToSize(0) open write(v) close)
	appendToContents := method(
		openForAppending
		call evalArgs foreach(v, write(v))
		close
	)

	create := method(if(open, close, nil))

	// This won't work until Sequence split exists.
	baseName := method(name split(".") slice(0, -1) join("."))

	thisSourceFile := method(File with(call message label))

	// This won't work until Sequence pathComponent exists.
	parentDirectory := method(Directory with(path pathComponent))
)

Directory do(
	size := method(self items size)
)

Map do(
	hasValue := method(value, self values contains(value))
	addKeysAndValues := method(keys, values,
		keys foreach(i, key, self atPut(key, values at(i)))
		self
	)

	with := method(
		m := Map clone
		for(i, 0, call argCount - 1, 2,
			m atPut(call evalArgAt(i), call evalArgAt(i+1))
		)
		m
	)

	asJson := method("{" .. self keys map(k, k asJson .. ":" .. self at(k) asJson) join(",") .. "}")
	asList := method(keys map(k, list(k, at(k))))

	map := method(
		l := List clone
		kn := call argAt(0) name
		vn := call argAt(1) name
		m := call argAt(2)
		self foreach(k, v,
			call sender setSlot(kn, k)
			call sender setSlot(vn, getSlot("v"))
			//FIXME: setSlot eats control flow here
			r := call sender doMessage(m)
			l append(r)
		)
		l
	)
	select := method(
		m := Map clone
		keys foreach(k,
			if(call argCount > 1,
				call sender setSlot(call argAt(0) name, k)
				if(call argCount == 3,
					call sender setSlot(call argAt(1) name, self at(k))
				)
			)
			//FIXME: setSlot eats control flow here
			v := call evalArgAt(call argCount - 1)
			if(getSlot("v"), m atPut(k, self at(k)))
		)
		m
	)
	detect := method(
		keys foreach(k,
			if(call argCount > 1,
				call sender setSlot(call argAt(0) name, k)
				if(call argCount == 3,
					call sender setSlot(call argAt(1) name, self at(k))
				)
			)
			//FIXME: setSlot eats control flow here
			v := call evalArgAt(call argCount - 1)
			if(getSlot("v"), return list(k, self at(k)))
		)
	)

	merge := method(other,
		self clone mergeInPlace(other)
	)
	mergeInPlace := method(other,
		k := keys
		v := keys map(x, other at(x))
		addKeysAndValues(k, v)
	)

	reverseMap := method(
		k := keys
		v := k map(x, self at(x))
		Map clone addKeysAndValues(v, k)
	)

	asObject := method(
		x := Object clone
		self foreach(k, v, x setSlot(k, getSlot("v")))
		x
	)

	isEmpty := method(size == 0)
	isNotEmpty := method(size != 0)
)

Date do(
	setSlot("+", method(dur, self clone += dur))

	// The original implementation of Date today starts with Date now, which
	// means it modifies the Core Date instead of the receiver...
	today := method(now setHour(0) setMinute(0) setSecond(0))
	isToday := method(
		n := Date clone now
		n year == year and n month == month and n day == day
	)

	// The only real difference between secondsToRun and cpuSecondsToRun is
	// that the former relays its stop status, so any control flow in the
	// message will cause it to quit instead. If we were to make the latter
	// actually measure CPU time only, it would be a different story.
	secondsToRun := method(
		t := Date clone now
		call evalArgAt(0)
		t secondsSinceNow
	) setPassStops(true)

	asAtomDate := method(clone convertToUTC asString("%Y-%m-%dT%H:%M:%SZ"))
	asJson := method(asString asJson)

	asNumberString := method(asNumber asString alignLeft(27, "0"))
	timeStampString := method(Date clone now asNumberString)
)

Duration do(
	setSlot("+", method(other, self clone += other))
	setSlot("-", method(other, self clone -= other))
)

Block do(
	asSimpleString := method(
		if(scope, "block", "method") .. "(" .. argumentNames append("...") join(", ") .. ")"
	)
	asString := method(Formatter clone formatBlock(getSlot("self")) buf)

	callWithArgList := method(args,
		getSlot("self") doMessage(argList asMessage setName("call"))
	)

	Formatter := Object clone do(
		line    ::= 0
		isEmpty ::= true
		depth   ::= 0
		buf     ::= nil
		ops     ::= nil
		asgn    ::= nil

		init := method(
			buf = Sequence clone
			ops = OperatorTable operators
			asgn = OperatorTable reverseAssignOperators
			asgn atPut("setSlotWithType", ":=")
		)

		appendSeq := method(call delegateTo(buf); isEmpty = false)

		newLine := method(
			buf appendSeq("\n")
			line = line + 1
			isEmpty = true
		)
		newLinesTo := method(msg,
			(msg lineNumber - line) minMax(0, 2) repeat(newLine)
			line = msg lineNumber
		)

		indent := method(depth repeat(appendSeq("    ")))

		formatBlock := method(blk,
			msg := getSlot("blk") message

			// This won't work until CLI exists.
			// if(msg label != CLI commandLineLabel,
			appendSeq("# " .. msg label .. ":" .. msg lineNumber, "\n") // )

			appendSeq("method(")
			if(getSlot("blk") argumentNames size > 0,
				appendSeq(getSlot("blk") argumentNames join(", "), ",")
			)
			newLine
			line = msg lineNumber
			formatIndentedMessage(msg)
			newLine
			appendSeq(")")
			self
		)

		formatIndentedMessage := method(msg,
			depth = depth + 1
			formatMessage(msg)
			depth = depth - 1
		)

		formatMessage := method(msg,
			m := msg
			while(m,
				if(m isEndOfLine,
					if(line == m next ?lineNumber,
						appendSeq(m name)
					)
					m = m next
					continue
				)
				newLinesTo(m)
				if(isEmpty, indent, if(m != msg, appendSeq(" ")))
				if(asgn hasKey(m name)) then(
					args := m arguments
					if(args first hasCachedResult,
						appendSeq(args first cachedResult .. " " .. asgn at(m name) .. " ")
						if(args at(1), formatMessage(args at(1)))
					,
						appendSeq(m name)
						if(m argCount > 0, formatArguments(m))
					)
				) elseif(ops hasKey(m name)) then(
					appendSeq(m name, " ")
					if(m arguments first, formatMessage(m arguments first))
				) else(
					appendSeq(m name)
					if(m argCount > 0, formatArguments(m))
				)
				m = m next
			)
		)

		formatArguments := method(msg,
			appendSeq("(")
			start := line
			msg arguments foreach(i, arg,
				if(i > 0,
					if(line == start, appendSeq(", "), newLine; indent; appendSeq(","); newLine)
				)
				formatIndentedMessage(arg)
			)
			if(line != start, newLine; indent)
			appendSeq(")")
		)
	)
)

Core getLocalSlot("CFunction") ifNil(Exception raise) do(
	type := "CFunction"
)

Message do(
	asSimpleString := method(
		s := self asString asMutable replaceSeq(" ;\n", "; ")
		if(s size > 40, s exSlice(0, 37) .. "...", s)
	)

	union := method(
		m := Message clone
		l := list(self)
		call message argAt(0) arguments foreach(arg, l append(arg))
		m setArguments(l)
	)
)

OperatorTable do(
	addOperator := method(name, prec, operators atPut(name, prec ifNilEval(0)); self)
	addAssignOperator := method(name, calls, assignOperators atPut(name, calls asSymbol asUTF8); self)

	reverseAssignOperators := method(assignOperators reverseMap)

	asString := method(
		b := Sequence clone appendSeq(self asSimpleString, ":\nOperators")
		self operators values unique sort foreach(prec,
			b appendSeq("\n  ", prec asString alignLeft(4), self operators select(k, v, v == prec) keys sort join(" "))
		)
		b appendSeq("\n\nAssign Operators")
		self assignOperators keys sort foreach(name,
			calls := self assignOperators at(name)
			b appendSeq("\n  ", name alignLeft(4), calls)
		)
		b
	)
)

// I don't know why this is on Core, but it is.
Core tildeExpandsTo := if(System platform == "windows",
	method(System getEnvironmentVariable("UserProfile"))
,
	method(System getEnvironmentVariable("HOME"))
)

// Unit testing stuff ------

TestRunner := Object clone do(
	width ::= 70

	init := method(
		self cases := nil
		self exceptions := list()
		self runtime := 0
	)

	testCount := method(
		self cases values reduce(n, names, n + names size, 0)
	)

	name := method(
		if(self cases size > 1,
			System launchScript fileName
		,
			if(self cases size == 1,
				self cases keys first
			,
				""
			)
		)
	)

	linebreak := method(
		if(self ?dots, self dots = self dots + 1, self dots := 1)
		if(dots % width == 0, "\n" print)
	)
	success := method("." print; linebreak)
	error := method(name, exc,
		exceptions append(list(name, exc))
		"E" print
		linebreak
	)

	run := method(tests,
		self cases := tests
		self runtime := Date secondsToRun(
			tests foreach(name, slots,
				case := Lobby getSlot(name)
				slots foreach(slot,
					case setUp
					exc := try(case doString(slot))
					if(exc, error(name .. " " .. slot, exc), success)
					case tearDown
				)
			)
		)
		printExceptions
		printSummary
	)

	printExceptions := method(
		"\n" print
		exceptions foreach(exc,
			("=" repeated(width) .. "\nFAIL: " .. exc at(0) .. "\n" .. "-" repeated(width)) println
			exc showStack
		)
	)
	printSummary := method(
		"-" repeated(width) println
		("Ran " .. testCount .. " test" .. if(testCount != 1, "s", "") .. " in " .. runtime .. "s\n") println
		result := if(exceptions isEmpty not, "FAILED (failures #{exceptions size})" interpolate, "OK")
		(result .. name alignRight(width - result size) .. "\n") println
	)
)

RunnerMixIn := Object clone do(
	run := method(TestRunner clone run(prepare))
)

UnitTest := Object clone prependProto(RunnerMixIn) do(
	setUp := method(nil)
	tearDown := method(nil)

	testSlotNames := method(
		names := self slotNames select(beginsWithSeq("test"))
		if(names isEmpty, names, names sortByKey(name, self getSlot(name) message lineNumber))
	)

	prepare := method(Map with(self type, testSlotNames))

	fail := method(error, Exception raise(if(error, error, "fail")))

	assertEquals := method(a, b, m,
		m ifNil(m = call message)
		if(a != b, fail(
			"'#{m argAt(0)} != #{m argAt(1)}' --> '#{a asSimpleString} != #{b asSimpleString}'" interpolate
		))
	)
	assertNotEquals := method(a, b, m,
		m ifNil(m = call message)
		if(a == b, fail(
			"'#{m argAt(0)} == #{m argAt(1)}' --> '#{a asSimpleString} == #{b asSimpleString}'" interpolate
		))
	)
	assertSame := method(a, b, assertEquals(a uniqueId, b uniqueId, call message))
	assertNotSame := method(a, b, assertNotEquals(a uniqueId, b uniqueId, call message))
	assertNil := method(a, assertEquals(a, nil, call message))
	assertNotNil := method(a, assertNotEquals(a, nil, call message))
	assertTrue := method(a, assertEquals(a, true, call message))
	assertFalse := method(a, assertEquals(a, false, call message))
	assertRaisesException := method(
		try(call evalArgAt(0)) ifNil(
			fail("'#{call argAt(0)}' should have raised Exception" interpolate)
		)
	)
	assertEqualsWithinDelta := method(a, b, delta,
		if((a - b) abs > delta,
			fail("#{a} expected, but was #{b} (allowed delta: #{delta})")
		)
	)
	knownBug := method(fail("'#{call argAt(0)}' is a known bug" interpolate))
)

DirectoryCollector := TestSuite := Object clone prependProto(RunnerMixIn) do(
	path ::= lazySlot(System launchPath)
	with := method(p, self clone setPath(p))
	testFiles := method(Directory with(path) files select(name endsWithSeq("Test.io")))
	prepare := method(
		testFiles foreach(file, Lobby doString(file contents, file path))
		FileCollector prepare
	)
)

FileCollector := Object clone prependProto(RunnerMixIn) do(
	prepare := method(
		cases := Map clone
		Lobby foreachSlot(name, value,
			if(getSlot("value") isActivatable not and value isKindOf(UnitTest),
				cases mergeInPlace(value prepare)
			)
		)
		cases
	)
)
`
