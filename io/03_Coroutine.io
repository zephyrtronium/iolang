Scheduler do(
	currentCoroutine := Coroutine getSlot("currentCoroutine")
	waitForCorosToComplete := method(while(yieldingCoros size > 0, yield))
)

Coroutine do(
	parentCoroutine ::= nil
	runMessage ::= nil
	runTarget ::= nil
	runLocals ::= nil
	exception ::= nil
	result ::= nil

	yieldingCoros := Scheduler getSlot("yieldingCoros")

	label := method(self uniqueId)
	setLabel := method(s, self label = s .. "_" .. self uniqueId)

	showYielding := method(s,
		File standardOutput writeln("   ", label, " ", s)
		yieldingCoros foreach(v, File standardOutput writeln("    ", v label))
	)

	isYielding := method(yieldingCoros contains(self))

	main := method(setResult(self getSlot("runTarget") doMessage(runMessage, self getSlot("runLocals"))))
)

Object do(
	setSlot("@", getSlot("futureSend"))
	setSlot("@@", getSlot("asyncSend"))

	coroDo := method(call delegateToMethod(self, "coroFor") run)
	coroDoLater := method(call delegateToMethod(self, "coroWith") run)
	coroFor := method(Coroutine clone setRunTarget(call sender) setRunLocals(call sender) setRunMessage(call argAt(0)))
	coroWith := method(Coroutine clone setRunTarget(self) setRunLocals(call sender) setRunMessage(call argAt(0)))
	
	currentCoro := method(Coroutine currentCoroutine)
)
