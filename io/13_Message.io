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
