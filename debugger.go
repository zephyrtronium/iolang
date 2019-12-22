package iolang

import (
	"fmt"
	"sync/atomic"
)

// Debugger is a debugger for a single coroutine.
type Debugger struct {
	// msgs is the queue of messages to process.
	msgs chan debugMessage
}

// debugMessage holds a context to be debugged and a channel to indicate it has.
type debugMessage struct {
	msg    *Message
	target *Object
	locals *Object
	dbg    chan struct{}
}

// tagDebugger is the Tag type for Debugger objects.
type tagDebugger struct{}

func (tagDebugger) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagDebugger) CloneValue(value interface{}) interface{} {
	return Debugger{msgs: make(chan debugMessage)}
}

func (tagDebugger) String() string {
	return "Debugger"
}

// DebuggerTag is the tag for Debugger objects. Activate returns self.
// CloneValue creates a new debug channel.
var DebuggerTag tagDebugger

func (vm *VM) initDebugger() {
	slots := Slots{
		"start": vm.NewCFunction(DebuggerStart, DebuggerTag),
	}
	debugger := Debugger{msgs: make(chan debugMessage)}
	vm.coreInstall("Debugger", slots, debugger, DebuggerTag)
}

// DebugMessage does nothing if debugging is disabled for the VM; otherwise, it
// sends the execution context to the debugger and waits for it to be handled.
func (vm *VM) DebugMessage(target, locals *Object, msg *Message) {
	if atomic.LoadUint32(&vm.Debug) != 0 {
		debug, ok := vm.Core.GetLocalSlot("Debugger")
		if !ok {
			return
		}
		if debug.Tag() != DebuggerTag {
			return
		}
		dbg := debug.Value.(Debugger)
		done := make(chan struct{})
		dbg.msgs <- debugMessage{msg: msg, target: target, locals: locals, dbg: done}
		<-done
	}
}

// DebuggerStart is a Debugger method.
//
// start begins processing the debugger's message queue. This behaves like
// loop(self vmWillSendMessage(nextMessageInQueue)).
func DebuggerStart(vm *VM, target, locals *Object, msg *Message) *Object {
	dbg := target.Value.(Debugger)
	pf := vm.IdentMessage("vmWillSendMessage")
	for {
		select {
		case work := <-dbg.msgs:
			target.SetSlots(Slots{
				"messageCoroutine": vm.Coro,
				"messageSelf":      work.target,
				"messageLocals":    work.locals,
				"message":          vm.MessageObject(work.msg),
			})
			r, stop := vm.Perform(target, locals, pf)
			close(work.dbg)
			switch stop {
			case NoStop, ContinueStop: // do nothing
			case BreakStop:
				return r
			case ExceptionStop, ExitStop:
				return vm.Stop(r, stop)
			default:
				panic(fmt.Errorf("iolang: invalid Stop: %v", stop))
			}
		case <-vm.Sched.Alive:
			return nil
		}
	}
}

// CoroutineSetMessageDebugging is a Coroutine method.
//
// setMessageDebugging activates the debugger for the coroutine.
func CoroutineSetMessageDebugging(vm *VM, target, locals *Object, msg *Message) *Object {
	coro := target.Value.(Coroutine)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	if vm.AsBool(r) {
		atomic.StoreUint32(coro.Debug, 1)
	} else {
		atomic.StoreUint32(coro.Debug, 0)
	}
	return target
}
