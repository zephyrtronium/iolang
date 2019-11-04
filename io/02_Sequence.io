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
