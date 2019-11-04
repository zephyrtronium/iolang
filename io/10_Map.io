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
