package iolang

// A Coroutine holds control flow and debugging for a single Io coroutine.
type Coroutine struct {
	// Control is the control flow channel for the VM associated with this
	// coroutine.
	Control chan RemoteStop
	// Debug is a pointer to the VM's Debug flag.
	Debug *uint32
}

// tagCoro is the Tag type for Coroutine objects.
type tagCoro struct{}

func (tagCoro) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagCoro) CloneValue(value interface{}) interface{} {
	return Coroutine{Control: make(chan RemoteStop, 1)}
}

func (tagCoro) String() string {
	return "Coroutine"
}

// CoroutineTag is the Tag for Coroutine objects. Activate returns the
// coroutine. CloneValue creates a new control flow channel and no debugging.
var CoroutineTag tagCoro

// VMFor creates a VM for a given Coroutine object so that it can run Io code.
// The new VM does not have debugging enabled. Panics if the object is not a
// Coroutine.
func (vm *VM) VMFor(coro *Object) *VM {
	coro.Lock()
	c := coro.Value.(Coroutine)
	r := &VM{
		Lobby:      vm.Lobby,
		Core:       vm.Core,
		Addons:     vm.Addons,
		BaseObject: vm.BaseObject,
		True:       vm.True,
		False:      vm.False,
		Nil:        vm.Nil,
		Operators:  vm.Operators,
		Sched:      vm.Sched,
		Control:    c.Control,
		Coro:       coro,
		addonmaps:  vm.addonmaps,
		StartTime:  vm.StartTime,
	}
	c.Debug = &r.Debug
	coro.Value = c
	coro.Unlock()
	return r
}

func (vm *VM) initCoroutine() {
	slots := Slots{
		"currentCoroutine":      vm.NewCFunction(CoroutineCurrentCoroutine, nil),
		"implementation":        vm.NewString("goroutines"),
		"implementationVersion": vm.NewNumber(0), // in case API changes
		"isCurrent":             vm.NewCFunction(CoroutineIsCurrent, CoroutineTag),
		"pause":                 vm.NewCFunction(CoroutinePause, CoroutineTag),
		"resume":                vm.NewCFunction(CoroutineResume, CoroutineTag),
		"run":                   vm.NewCFunction(CoroutineRun, CoroutineTag),
		"setMessageDebugging":   vm.NewCFunction(CoroutineSetMessageDebugging, CoroutineTag), // debugger.go
		"type":                  vm.NewString("Coroutine"),
		"yield":                 vm.NewCFunction(CoroutineYield, CoroutineTag),
	}
	slots["resumeLater"] = slots["resume"]
	value := Coroutine{Control: vm.Control, Debug: &vm.Debug}
	vm.Coro = vm.ObjectWith(slots, []*Object{vm.BaseObject}, value, CoroutineTag)
	vm.Core.SetSlot("Coroutine", vm.Coro)
}

// run starts this inactive coroutine by activating its main slot. It should be
// used in a go statement.
func (vm *VM) run() {
	vm.Perform(vm.Coro, vm.Coro, vm.IdentMessage("main"))
	vm.Sched.Finish(vm)
}

// CoroutineCurrentCoroutine is a Coroutine method.
//
// currentCoroutine returns the current coroutine.
func CoroutineCurrentCoroutine(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.Coro
}

// CoroutineIsCurrent is a Coroutine method.
//
// isCurrent returns whether the receiver is the current coroutine.
func CoroutineIsCurrent(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.IoBool(vm.Coro == target)
}

// CoroutinePause is a Coroutine method.
//
// pause stops the coroutine's execution until it is sent the resume message. It
// will finish evaluating its current message before pausing. If all coroutines
// are paused, the program ends.
func CoroutinePause(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Value.(Coroutine).Control <- RemoteStop{Control: PauseStop}
	return target
}

// CoroutineResume is a Coroutine method.
//
// resume unpauses the coroutine, or starts it if it was not started.
func CoroutineResume(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Value.(Coroutine).Control <- RemoteStop{Control: ResumeStop}
	return target
}

// CoroutineRun is a Coroutine method.
//
// run starts this coroutine if it was not already running. The coroutine
// activates its main slot, which by default performs the message in runMessage
// upon runTarget using runLocals.
func CoroutineRun(vm *VM, target, locals *Object, msg *Message) *Object {
	coro := vm.VMFor(target)
	vm.Sched.Start(coro)
	go coro.run()
	return target
}

// CoroutineYield is a Coroutine method.
//
// yield reschedules all goroutines.
func CoroutineYield(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Value.(Coroutine).Control <- RemoteStop{}
	return target
}
