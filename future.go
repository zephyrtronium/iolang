package iolang

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

// A Future is a placeholder object that will be filled in by a dedicated
// coroutine.
type Future struct {
	// M is an atomic flag for whether the value has been computed.
	M uintptr
	// Value is the computed result, or nil while waiting for it.
	Value *Object
	// Coro is the coroutine which will fill in the value.
	Coro *VM
}

// tagFuture is the Tag type for Future objects.
type tagFuture struct{}

func (tagFuture) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
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
var FutureTag Tag = tagFuture{}

func (vm *VM) initFuture() {
	slots := Slots{
		"forward":      vm.NewCFunction(FutureForward, FutureTag),
		"waitOnResult": vm.NewCFunction(FutureWaitOnResult, FutureTag),
	}
	// Don't use coreInstall because we want no protos so we forward where
	// possible.
	vm.Core.SetSlot("Future", vm.NewObject(slots, nil, &Future{}, FutureTag))
}

// NewFuture creates a new Future object with its own coroutine and runs it.
func (vm *VM) NewFuture(target *Object, msg *Message) *Object {
	coro := vm.Coro.Clone()
	m := vm.MessageObject(msg)
	f := &Future{Coro: vm.VMFor(coro)}
	o := vm.NewObject(Slots{"runTarget": target, "runMessage": m}, vm.CoreProto("Future"), f, FutureTag)
	coro.SetSlots(Slots{
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
	target, _ := vm.Coro.GetSlot("runTarget")
	msg, _ := vm.Coro.GetSlot("runMessage")
	m, ok := msg.Value.(*Message)
	if !ok {
		panic("Future started without a message to run")
	}
	r, stop := m.Send(vm, target, target)
	if stop == ExceptionStop {
		// Exception. Send it to the target's handleActorException slot.
		m := vm.IdentMessage("handleActorException", vm.CachedMessage(r))
		r, stop = vm.Perform(target, target, m)
		if !ok {
			// Another exception while trying to handle the previous one. Give
			// up and send the exception back to the coroutine. It's probably
			// pointless, but there isn't anything else to do.
			// TODO: indicate that this new exception resulted while handling
			// the old one
			vm.Coro.SetSlot("exception", r)
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
func (f *Future) Wait(vm *VM) (*Object, Stop) {
	vm.Sched.Await(vm, f.Coro)
	for atomic.LoadUintptr(&f.M) == 0 {
		select {
		case stop := <-vm.Control:
			switch stop.Control {
			case NoStop, ResumeStop:
				runtime.Gosched()
			case ContinueStop, BreakStop, ReturnStop, ExceptionStop:
				return stop.Result, stop.Control
			case PauseStop:
				vm.Sched.pause <- vm
				for stop.Control != ResumeStop {
					switch stop = <-vm.Control; stop.Control {
					case NoStop, PauseStop: // do nothing
					case ContinueStop, BreakStop, ReturnStop, ExceptionStop:
						return stop.Result, stop.Control
					case ResumeStop:
						vm.Sched.Await(vm, f.Coro)
					default:
						panic(fmt.Sprintf("invalid status in received stop %#v", stop))
					}
				}
			default:
				panic(fmt.Sprintf("invalid status in received stop %#v", stop))
			}
		default: // do nothing
		}
	}
	return f.Value, NoStop
}

// FutureForward is a Future method.
//
// forward responds to messages to which the Future does not respond by proxying
// to the evaluated result. This causes a wait.
func FutureForward(vm *VM, target, locals *Object, msg *Message) *Object {
	f := target.Value.(*Future)
	if atomic.LoadUintptr(&f.M) == 0 {
		if f.Coro == nil {
			// This should apply only to Core Future, most likely due to
			// Core slotSummary or Future slotSummary. Grabbing the slot from
			// BaseObject is probably reasonable.
			t, proto := vm.BaseObject.GetSlot(msg.Name())
			if proto == nil {
				return vm.RaiseExceptionf("cannot use unstarted Future")
			}
			return t.Activate(vm, target, locals, proto, msg)
		}
		if r, stop := f.Wait(vm); stop != NoStop {
			return vm.Stop(r, stop)
		}
	}
	return vm.Stop(vm.Perform(f.Value, locals, msg))
}

// FutureWaitOnResult is a Future method.
//
// waitOnResult blocks until the result of the Future is computed. Returns nil.
func FutureWaitOnResult(vm *VM, target, locals *Object, msg *Message) *Object {
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
	if r, stop := f.Wait(vm); stop != NoStop {
		return vm.Stop(r, stop)
	}
	return vm.Nil
}

// ObjectAsyncSend is an Object method.
//
// asyncSend evaluates a message in a new coroutine.
func ObjectAsyncSend(vm *VM, target, locals *Object, msg *Message) *Object {
	if msg.ArgCount() == 0 {
		return vm.RaiseExceptionf("asyncSend requires an argument")
	}
	vm.NewFuture(target, msg.ArgAt(0))
	return vm.Nil
}

// ObjectFutureSend is an Object method.
//
// futureSend evaluates a message in a new coroutine and returns a Future which
// will become the result.
func ObjectFutureSend(vm *VM, target, locals *Object, msg *Message) *Object {
	if msg.ArgCount() == 0 {
		return vm.RaiseExceptionf("futureSend requires an argument")
	}
	return vm.NewFuture(target, msg.ArgAt(0))
}
