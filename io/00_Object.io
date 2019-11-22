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
	
	yield := method(Coroutine currentCoroutine yield)
	pause := method(Coroutine currentCoroutine pause)
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
