Path do(
	type := "Path"

	with := method(
		s := Sequence clone
		call message arguments foreach(p,
			v := call sender doMessage(p)
			v ifNonNil(s appendPathSeq(v))
		)
		s
	)

	thisSourceFilePath := method(Path absolute(call message label))
)
