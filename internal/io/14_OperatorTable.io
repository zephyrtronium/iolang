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
