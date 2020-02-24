Call do(
	argCount := method(self message argCount)
	argAt := method(n, self message argAt(n))
	evalArgAt := method(n, self sender doMessage(self message argAt(n)))
	hasArgs := method(argCount > 0)
	evalArgs := method(self message argsEvaluatedIn(self sender)) setPassStops(true)

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

	resetStopStatus := method(setStopStatus(Normal))
	relayStopStatus := method(
		stop := stopStatus(r := call evalArgAt(0))
		call sender call setStopStatus(stop)
		getSlot("r")
	)

	type := "Call"
)
