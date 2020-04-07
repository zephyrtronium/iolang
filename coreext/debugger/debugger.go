//go:generate go run ../../cmd/gencore debugger_init.go debugger ./io
//go:generate gofmt -s -w debugger_init.go

package debugger

import (
	"fmt"
	"sync/atomic"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/coroutine"
	"github.com/zephyrtronium/iolang/internal"
)

// Debugger is a debugger for a single coroutine.
type Debugger = internal.Debugger

// DebuggerTag is the tag for Debugger objects. Activate returns self.
// CloneValue creates a new debug channel.
var DebuggerTag = internal.DebuggerTag

func init() {
	internal.Register(initDebugger)
}

func initDebugger(vm *iolang.VM) {
	slots := iolang.Slots{
		"start": vm.NewCFunction(start, DebuggerTag),
	}
	debugger := internal.DebuggerWith(make(chan internal.DebugMessage))
	internal.CoreInstall(vm, "Debugger", slots, debugger, DebuggerTag)
	// Add setMessageDebugging to Coroutine.
	proto, _ := vm.GetLocalSlot(vm.Core, "Coroutine")
	vm.SetSlot(proto, "setMessageDebugging", vm.NewCFunction(coroutineSetMessageDebugging, coroutine.CoroutineTag))
	internal.Ioz(vm, coreIo, coreFiles)
}

// start is a Debugger method.
//
// start begins processing the debugger's message queue. This behaves like
// loop(self vmWillSendMessage(nextMessageInQueue)).
func start(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dbg := target.Value.(Debugger)
	// This method is run in a new coroutine, but we don't want the coroutine
	// to show up in scheduler methods. Tell the scheduler we've finished, even
	// though we really haven't and probably won't.
	vm.Sched.Finish(vm)
	pf := vm.IdentMessage("vmWillSendMessage")
	for {
		select {
		case work := <-internal.DebuggerChan(dbg):
			vm.SetSlots(target, iolang.Slots{
				"messageCoroutine": vm.Coro,
				"messageSelf":      work.Target,
				"messageLocals":    work.Locals,
				"message":          vm.MessageObject(work.Msg),
			})
			r, stop := vm.Perform(target, locals, pf)
			close(work.Dbg)
			switch stop {
			case iolang.NoStop, iolang.ContinueStop: // do nothing
			case iolang.BreakStop:
				return r
			case iolang.ExceptionStop, iolang.ExitStop:
				return vm.Stop(r, stop)
			default:
				panic(fmt.Errorf("iolang: invalid Stop: %v", stop))
			}
		case <-vm.Sched.Alive:
			return nil
		}
	}
}

// coroutineSetMessageDebugging is a Coroutine method.
//
// setMessageDebugging activates the debugger for the coroutine.
func coroutineSetMessageDebugging(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	coro := target.Value.(coroutine.Coroutine)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(r, stop)
	}
	if vm.AsBool(r) {
		atomic.StoreUint32(coro.Debug, 1)
	} else {
		atomic.StoreUint32(coro.Debug, 0)
	}
	return target
}
