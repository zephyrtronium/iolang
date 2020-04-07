//go:generate go run ../../cmd/gencore coroutine_init.go coroutine ./io
//go:generate gofmt -s -w coroutine_init.go

package coroutine

import (
	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"
)

// A Coroutine holds control flow and debugging for a single Io coroutine.
type Coroutine = internal.Coroutine

// CoroutineTag is the Tag for Coroutine objects. Activate returns the
// coroutine. CloneValue creates a new control flow channel and no debugging.
var CoroutineTag = internal.CoroutineTag

func init() {
	internal.Register(initCoroutine)
}

func initCoroutine(vm *iolang.VM) {
	slots := iolang.Slots{
		"currentCoroutine":      vm.NewCFunction(coroutineCurrentCoroutine, nil),
		"implementation":        vm.NewString("goroutines"),
		"implementationVersion": vm.NewNumber(0), // in case API changes
		"isCurrent":             vm.NewCFunction(coroutineIsCurrent, CoroutineTag),
		"pause":                 vm.NewCFunction(coroutinePause, CoroutineTag),
		"resume":                vm.NewCFunction(coroutineResume, CoroutineTag),
		"run":                   vm.NewCFunction(coroutineRun, CoroutineTag),
		"type":                  vm.NewString("Coroutine"),
		"yield":                 vm.NewCFunction(coroutineYield, CoroutineTag),
	}
	slots["resumeLater"] = slots["resume"]
	vm.SetSlots(vm.Coro, slots)
	vm.SetSlot(vm.Core, "Coroutine", vm.Coro)
	internal.Ioz(vm, coreIo, coreFiles)
}

// coroutineCurrentCoroutine is a Coroutine method.
//
// currentCoroutine returns the current coroutine.
func coroutineCurrentCoroutine(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	return vm.Coro
}

// coroutineIsCurrent is a Coroutine method.
//
// isCurrent returns whether the receiver is the current coroutine.
func coroutineIsCurrent(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	return vm.IoBool(vm.Coro == target)
}

// coroutinePause is a Coroutine method.
//
// pause stops the coroutine's execution until it is sent the resume message. It
// will finish evaluating its current message before pausing. If all coroutines
// are paused, the program ends.
func coroutinePause(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Value.(Coroutine).Control <- iolang.RemoteStop{Control: internal.PauseStop}
	return target
}

// coroutineResume is a Coroutine method.
//
// resume unpauses the coroutine, or starts it if it was not started.
func coroutineResume(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Value.(Coroutine).Control <- iolang.RemoteStop{Control: internal.ResumeStop}
	return target
}

// coroutineRun is a Coroutine method.
//
// run starts this coroutine if it was not already running. The coroutine
// activates its main slot, which by default performs the message in runMessage
// upon runTarget using runLocals.
func coroutineRun(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	coro := vm.VMFor(target)
	vm.Sched.Start(coro)
	go internal.RunCoro(coro)
	return target
}

// coroutineYield is a Coroutine method.
//
// yield reschedules all goroutines.
func coroutineYield(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Value.(Coroutine).Control <- iolang.RemoteStop{}
	return target
}
