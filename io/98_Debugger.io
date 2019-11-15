Debugger do(
	vmWillSendMessage := method(
		("Debugger vmWillSendMessage(" .. self message name .. ")") println
	)
	debuggerCoroutine := coroDo(start)
)
