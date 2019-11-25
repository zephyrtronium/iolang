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

	justSerialized := method(stream,
		stream write("list(")
		self foreach(i, v,
			getSlot("v") justSerialized(stream)
			stream write(if(i < self size - 1, ", ", ")"))
		)
	)
)
