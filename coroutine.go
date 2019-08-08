package iolang

func (vm *VM) initCoroutine() {
	slots := Slots{
		"currentCoroutine":      vm.NewCFunction(CoroutineCurrentCoroutine, nil),
		"implementation":        vm.NewString("goroutines"),
		"implementationVersion": vm.NewNumber(0), // in case API changes
		"isCurrent":             vm.NewCFunction(CoroutineIsCurrent, vm),
		"pause":                 vm.NewCFunction(CoroutinePause, vm),
		"resume":                vm.NewCFunction(CoroutineResume, vm),
		"run":                   vm.NewCFunction(CoroutineRun, vm),
		"type":                  vm.NewString("Coroutine"),
		"yield":                 vm.NewCFunction(CoroutineYield, vm),
	}
	slots["resumeLater"] = slots["resume"]
	vm.Object = *vm.ObjectWith(slots)
	vm.SetSlot(vm.Core, "Coroutine", vm)
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
func CoroutineCurrentCoroutine(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm, NoStop
}

// CoroutineIsCurrent is a Coroutine method.
//
// isCurrent returns whether the receiver is the current coroutine.
func CoroutineIsCurrent(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(vm == target), NoStop
}

// CoroutinePause is a Coroutine method.
//
// pause stops the coroutine's execution until it is sent the resume message. It
// will finish evaluating its current message before pausing. If all coroutines
// are paused, the program ends.
func CoroutinePause(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.(*VM).Stop <- RemoteStop{Control: PauseStop}
	return target, NoStop
}

// CoroutineResume is a Coroutine method.
//
// resume unpauses the coroutine, or starts it if it was not started.
func CoroutineResume(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.(*VM).Stop <- RemoteStop{Control: ResumeStop}
	return target, NoStop
}

// CoroutineRun is a Coroutine method.
//
// run starts this coroutine if it was not already running. The coroutine
// activates its main slot, which by default performs the message in runMessage
// upon runTarget using runLocals.
func CoroutineRun(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	coro := target.(*VM)
	vm.Sched.Start(coro)
	go coro.run()
	return target, NoStop
}

// CoroutineYield is a Coroutine method.
//
// yield reschedules all goroutines.
func CoroutineYield(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	target.(*VM).Stop <- RemoteStop{}
	return target, NoStop
}
