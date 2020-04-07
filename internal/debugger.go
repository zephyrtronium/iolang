package internal

import "sync/atomic"

// Debugger is a debugger for a single coroutine.
type Debugger struct {
	// msgs is the queue of messages to process.
	msgs chan DebugMessage
}

// DebuggerWith allows coreext/debugger to create a Debugger object.
func DebuggerWith(msgs chan DebugMessage) Debugger {
	return Debugger{msgs: msgs}
}

// DebuggerChan allows coreext/debugger to retrieve a Debugger's message
// channel.
func DebuggerChan(d Debugger) chan DebugMessage {
	return d.msgs
}

// DebugMessage holds a context to be debugged and a channel to indicate it has.
type DebugMessage struct {
	Msg    *Message
	Target *Object
	Locals *Object
	Dbg    chan struct{}
}

// tagDebugger is the Tag type for Debugger objects.
type tagDebugger struct{}

func (tagDebugger) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagDebugger) CloneValue(value interface{}) interface{} {
	return Debugger{msgs: make(chan DebugMessage)}
}

func (tagDebugger) String() string {
	return "Debugger"
}

// DebuggerTag is the tag for Debugger objects. Activate returns self.
// CloneValue creates a new debug channel.
var DebuggerTag tagDebugger

// DebugMessage does nothing if debugging is disabled for the VM; otherwise, it
// sends the execution context to the debugger and waits for it to be handled.
func (vm *VM) DebugMessage(target, locals *Object, msg *Message) {
	if atomic.LoadUint32(&vm.Debug) != 0 {
		vm.debugMessageSlow(target, locals, msg)
	}
}

// debugMessageSlow is an outlined path of DebugMessage.
func (vm *VM) debugMessageSlow(target, locals *Object, msg *Message) {
	debug, ok := vm.GetLocalSlot(vm.Core, "Debugger")
	if !ok {
		return
	}
	if debug.Tag() != DebuggerTag {
		return
	}
	dbg := debug.Value.(Debugger)
	done := make(chan struct{})
	dbg.msgs <- DebugMessage{Msg: msg, Target: target, Locals: locals, Dbg: done}
	<-done
}
