//go:generate go run ../../cmd/gencore future_init.go future ./io
//go:generate gofmt -s -w future_init.go

package future

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/zephyrtronium/iolang"
	_ "github.com/zephyrtronium/iolang/coreext/coroutine" // dependency
	"github.com/zephyrtronium/iolang/internal"
)

// A Future is a placeholder object that will be filled in by a dedicated
// coroutine.
type Future struct {
	// M is an atomic flag for whether the value has been computed.
	M uintptr
	// Value is the computed result, or nil while waiting for it.
	Value *iolang.Object
	// Coro is the coroutine which will fill in the value.
	Coro *iolang.VM
}

// tagFuture is the Tag type for Future objects.
type tagFuture struct{}

func (tagFuture) Activate(vm *iolang.VM, self, target, locals, context *iolang.Object, msg *iolang.Message) *iolang.Object {
	if f := self.Value.(*Future); atomic.LoadUintptr(&f.M) == 1 {
		return f.Value.Activate(vm, target, locals, context, msg)
	}
	return self
}

func (tagFuture) CloneValue(value interface{}) interface{} {
	return &Future{}
}

func (tagFuture) String() string {
	return "Future"
}

// FutureTag is the Tag for Future objects. Activate activates the future's
// result if it is available and returns self if it is not. CloneValue returns
// a new Future with no coroutine.
var FutureTag tagFuture

func init() {
	internal.Register(initFuture)
}

func initFuture(vm *iolang.VM) {
	slots := iolang.Slots{
		"forward":      vm.NewCFunction(forward, FutureTag),
		"waitOnResult": vm.NewCFunction(waitOnResult, FutureTag),
	}
	// Don't use coreInstall because we want no protos so we forward where
	// possible.
	vm.SetSlot(vm.Core, "Future", vm.ObjectWith(slots, nil, &Future{}, FutureTag))
	// Install Object methods that use Futures.
	slots = iolang.Slots{
		"asyncSend":  vm.NewCFunction(objectAsyncSend, nil),
		"futureSend": vm.NewCFunction(objectFutureSend, nil),
	}
	vm.SetSlots(vm.BaseObject, slots)
	internal.Ioz(vm, coreIo, coreFiles)
}

// New creates a new Future object with its own coroutine and runs it.
func New(vm *iolang.VM, target *iolang.Object, msg *iolang.Message) *iolang.Object {
	coro := vm.Coro.Clone()
	m := vm.MessageObject(msg)
	f := &Future{Coro: vm.VMFor(coro)}
	o := vm.ObjectWith(iolang.Slots{"runTarget": target, "runMessage": m}, vm.CoreProto("Future"), f, FutureTag)
	vm.SetSlots(coro, iolang.Slots{
		"runTarget":       target,
		"runMessage":      m,
		"runLocals":       target,
		"parentCoroutine": vm.Coro,
	})
	go f.run()
	return o
}

// run starts the Future's coroutine and manages its lifetime. It should be used
// in a go statement.
func (f *Future) run() {
	vm := f.Coro
	vm.Sched.Start(f.Coro)
	defer vm.Sched.Finish(f.Coro)
	target, _ := vm.GetSlot(vm.Coro, "runTarget")
	msg, _ := vm.GetSlot(vm.Coro, "runMessage")
	m, ok := msg.Value.(*iolang.Message)
	if !ok {
		panic("Future started without a message to run")
	}
	r, stop := m.Send(vm, target, target)
	if stop == iolang.ExceptionStop {
		// Exception. Send it to the target's handleActorException slot.
		m := vm.IdentMessage("handleActorException", vm.CachedMessage(r))
		r, stop = vm.Perform(target, target, m)
		if !ok {
			// Another exception while trying to handle the previous one. Give
			// up and send the exception back to the coroutine. It's probably
			// pointless, but there isn't anything else to do.
			// TODO: indicate that this new exception resulted while handling
			// the old one
			vm.SetSlot(vm.Coro, "exception", r)
			vm.Raise(r)
		}
	}
	f.Value = r
	if !atomic.CompareAndSwapUintptr(&f.M, 0, 1) {
		// Someone had already set the result!
		panic(fmt.Sprintf("iolang: future proxied by multiple goroutines: %#v", f))
	}
}

// Wait spins until the value is ready. While spinning, the future monitors the
// coroutine's remote control flow channel. The value is returned if it is
// ready, otherwise the Stop that ceased monitoring. Panics if the coroutine
// hasn't started yet.
//
// NOTE: If Wait returns a Stop, then that Stop was sent to the waiting
// coroutine, not the Future's.
func (f *Future) Wait(vm *iolang.VM) (*iolang.Object, iolang.Stop) {
	vm.Sched.Await(vm, f.Coro)
	for atomic.LoadUintptr(&f.M) == 0 {
		select {
		case stop := <-vm.Control:
			switch stop.Control {
			case iolang.NoStop, internal.ResumeStop:
				runtime.Gosched()
			case iolang.ContinueStop, iolang.BreakStop, iolang.ReturnStop, iolang.ExceptionStop, iolang.ExitStop:
				return stop.Result, stop.Control
			case internal.PauseStop:
				vm.Sched.Pause(vm)
				for stop.Control != internal.ResumeStop {
					switch stop = <-vm.Control; stop.Control {
					case iolang.NoStop, internal.PauseStop: // do nothing
					case iolang.ContinueStop, iolang.BreakStop, iolang.ReturnStop, iolang.ExceptionStop, iolang.ExitStop:
						return stop.Result, stop.Control
					case internal.ResumeStop:
						vm.Sched.Await(vm, f.Coro)
					default:
						panic(fmt.Errorf("iolang: invalid Stop: %w, value: %v", stop.Control.Err(), stop.Result))
					}
				}
			default:
				panic(fmt.Errorf("iolang: invalid Stop: %w, value: %v", stop.Control.Err(), stop.Result))
			}
		default: // do nothing
		}
	}
	return f.Value, iolang.NoStop
}

// FutureForward is a Future method.
//
// forward responds to messages to which the Future does not respond by proxying
// to the evaluated result. This causes a wait.
func forward(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	f := target.Value.(*Future)
	if atomic.LoadUintptr(&f.M) == 0 {
		if f.Coro == nil {
			// This should apply only to Core Future, most likely due to
			// Core slotSummary or Future slotSummary. Grabbing the slot from
			// BaseObject is probably reasonable.
			t, proto := vm.GetSlot(vm.BaseObject, msg.Name())
			if proto == nil {
				return vm.RaiseExceptionf("cannot use unstarted Future")
			}
			return t.Activate(vm, target, locals, proto, msg)
		}
		if r, stop := f.Wait(vm); stop != iolang.NoStop {
			return vm.Stop(r, stop)
		}
	}
	return vm.Stop(vm.Perform(f.Value, locals, msg))
}

// FutureWaitOnResult is a Future method.
//
// waitOnResult blocks until the result of the Future is computed. Returns nil.
func waitOnResult(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	f := target.Value.(*Future)
	if f.Coro == nil {
		// Either it hasn't been started yet or it's already finished. In the
		// latter case, M should already be set.
		if atomic.LoadUintptr(&f.M) == 1 {
			return vm.Nil
		}
		// Technically, it's possible for the coro to have started between the
		// atomic check and now, but that's probably always an erroneous race.
		return vm.RaiseExceptionf("cannot wait on unstarted Future")
	}
	if r, stop := f.Wait(vm); stop != iolang.NoStop {
		return vm.Stop(r, stop)
	}
	return vm.Nil
}

// objectAsyncSend is an Object method.
//
// asyncSend evaluates a message in a new coroutine.
func objectAsyncSend(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	if msg.ArgCount() == 0 {
		return vm.RaiseExceptionf("asyncSend requires an argument")
	}
	New(vm, target, msg.ArgAt(0))
	return vm.Nil
}

// objectFutureSend is an Object method.
//
// futureSend evaluates a message in a new coroutine and returns a Future which
// will become the result.
func objectFutureSend(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	if msg.ArgCount() == 0 {
		return vm.RaiseExceptionf("futureSend requires an argument")
	}
	return New(vm, target, msg.ArgAt(0))
}
