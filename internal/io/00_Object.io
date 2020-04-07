Object do(
	println := method(getSlot("self") print; "\n" print)
	write := method(call evalArgs foreach(print); getSlot("self"))
	writeln := method(call evalArgs foreach(print); "\n" print; getSlot("self"))
	// Use setSlot directly to circumvent operator shuffling.
	setSlot("and", method(v, v isTrue))
	setSlot("-", method(v, v negate))
	setSlot("..", method(v, getSlot("self") asString .. v asString))

	init := getSlot("thisContext")

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

	proto := method(getSlot("self") protos first)
	hasProto := getSlot("isKindOf")

	hasSlot := method(slot, getSlot("self") hasLocalSlot(slot) or getSlot("self") ancestorWithSlot(slot) isNil not)
	setSlotWithType := method(slot, value,
		getSlot("self") setSlot(slot, value)
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

	addTrait := method(v, res,
		if(call argCount == 0, Exception raise("Object addTrait requires one or two arguments"))
		res := res ifNilEval(Map clone)
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
		if(getSlot("self") isKindOf(Block), return getSlot("self") asSimpleString)
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
			// Even though ? is the most binding operator, operator shuffling
			// changes ?(m) into ?((m)) to try to preserve precedence of
			// parentheses. Check for this case and circumvent it (although the
			// original doesn't).
			if(m name isEmpty and m next isNil and m argCount == 1,
				m = m argAt(0)
			)
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

	relativeDoFile := doRelativeFile := method(p,
		self doFile(Path with(call message label pathComponent, p))
	)
	launchFile := method(path, args,
		args ifNil(args = list)
		System do(launchPath := path pathComponent; launchScript := path)
		Directory setCurrentWorkingDirectory(System launchPath)
		self doFile(path)
	)

	inlineMethod := method(
		m := call message argAt(0) clone
		m setIsActivatable(true)
		getSlot("m")
	)

	deprecatedWarning := method(alt,
		m := call sender call ifNil(Exception raise("deprecatedWarning must be called from within a Block")) message
		self writeln("Warning in ", m label, ": ", m name, if(alt, " is deprecated. Use " .. alt .. " instead.", "is deprecated."))
	)

	serialized := method(stream,
		if(stream isNil, stream := SerializationStream clone)
		justSerialized(stream)
		stream ?output
	)
	justSerialized := method(stream,
		stream write(getSlot("self") getLocalSlot("type") ifNonNilEval(getSlot("self") proto type) ifNilEval(getSlot("self") type), " clone do(\n")
		getSlot("self") serializedSlots(stream)
		stream write(")\n")
	)
	serializedSlots := method(stream,
		getSlot("self") serializedSlotsWithNames(getSlot("self") slotNames, stream)
	)
	serializedSlotsWithNames := method(names, stream,
		names foreach(name,
			stream write("\t", name, " := ")
			getSlot("self") getSlot(name) serialized(stream)
			stream write("\n")
		)
	)
)

true do(
	ifTrue := Object getSlot("evalArgAndReturnSelf")
	then := Object getSlot("evalArgAndReturnNil")
	justSerialized := method(stream, stream write("true"))
)

false do(
	ifFalse := Object getSlot("evalArgAndReturnSelf")
	else := Object getSlot("evalArgAndReturnNil")
	elseif := Object getSlot("if")
	setSlot("or",  method(v, v isTrue))
	asBoolean := false
	justSerialized := method(stream, stream write("false"))
)

nil do(
	ifNil := Object getSlot("evalArgAndReturnSelf")
	ifNilEval := Object getSlot("evalArg")
	ifNonNil := nil
	ifNonNilEval := nil
	setSlot("or",  method(v, v isTrue))
	catch := nil
	pass := nil
	asBoolean := nil
	justSerialized := method(stream, stream write("nil"))
)

SerializationStream := Object clone do(
	init := method(
		self output := Sequence clone
	)
	write := method(call evalArgs foreach(v, self output appendSeq(v)))
)
