Exception do(
	caughtMessage ::= nil
	coroutine ::= nil
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

Error := Object clone do(
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
