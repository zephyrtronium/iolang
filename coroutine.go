package iolang

func (vm *VM) initCoroutine() {
	slots := Slots{
		"currentCoroutine": vm.NewCFunction(CoroutineCurrentCoroutine),
		"implementation": vm.NewString("goroutines"),
		"implementationVersion": vm.NewNumber(0), // in case API changes
		"isCurrent": vm.NewTypedCFunction(CoroutineIsCurrent, vm),
		"pause": vm.NewTypedCFunction(CoroutinePause, vm),
		"resume": vm.NewTypedCFunction(CoroutineResume, vm),
		"run": vm.NewTypedCFunction(CoroutineRun, vm),
		"type": vm.NewString("Coroutine"),
		"yield": vm.NewTypedCFunction(CoroutineYield, vm),
	}
	slots["resumeLater"] = slots["resume"]
	vm.Object = *vm.ObjectWith(slots)
	SetSlot(vm.Core, "Coroutine", vm)
}

// run starts this inactive coroutine by activating its main slot. It should be
// used in a go statement.
func (vm *VM) run() {
	vm.Perform(vm, vm, vm.IdentMessage("main"))
	vm.Sched.Finish(vm)
}

// CoroutineCurrentCoroutine is a Coroutine method.
//
// currentCoroutine returns the current coroutine.
func CoroutineCurrentCoroutine(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm
}

// CoroutineIsCurrent is a Coroutine method.
//
// isCurrent returns whether the receiver is the current coroutine.
func CoroutineIsCurrent(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(vm == target)
}

// CoroutinePause is a Coroutine method.
//
// pause stops the coroutine's execution until it is sent the resume message. It
// will finish evaluating its current message before pausing. If all coroutines
// are paused, the program ends.
func CoroutinePause(vm *VM, target, locals Interface, msg *Message) Interface {
	target.(*VM).Stop <- Stop{Status: PauseStop}
	return target
}

// CoroutineResume is a Coroutine method.
//
// resume unpauses the coroutine, or starts it if it was not started.
func CoroutineResume(vm *VM, target, locals Interface, msg *Message) Interface {
	target.(*VM).Stop <- Stop{Status: ResumeStop}
	return target
}

// CoroutineRun is a Coroutine method.
//
// run starts this coroutine if it was not already running. The coroutine
// activates its main slot, which by default performs the message in runMessage
// upon runTarget using runLocals.
func CoroutineRun(vm *VM, target, locals Interface, msg *Message) Interface {
	coro := target.(*VM)
	vm.Sched.Start(coro)
	go coro.run()
	return target
}

// CoroutineYield is a Coroutine method.
//
// yield reschedules all goroutines.
func CoroutineYield(vm *VM, target, locals Interface, msg *Message) Interface {
	target.(*VM).Stop <- Stop{}
	return target
}
