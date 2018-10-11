package iolang

// Call contains information on how a message was activated.
type Call struct {
	Object
	Sender    Interface
	Activated Actor
	Msg       *Message
	Target    Interface
}

// NewCall creates a Call object sent from sender to the target's actor using
// the message msg.
func (vm *VM) NewCall(sender Interface, actor Actor, msg *Message, target Interface) *Call {
	return &Call{
		Object:    *vm.CoreInstance("Call"),
		Sender:    sender,
		Activated: actor,
		Msg:       msg,
		Target:    target,
	}
}

func (vm *VM) initCall() {
	slots := Slots{
		"activated": vm.NewTypedCFunction(CallActivated, "CallActivated()"),
		"argAt":     vm.NewTypedCFunction(CallArgAt, "CallArgAt(n)"),
		"argCount":  vm.NewTypedCFunction(CallArgCount, "CallArgCount()"),
		"evalArgAt": vm.NewTypedCFunction(CallEvalArgAt, "CallEvalArgAt(n)"),
		"message":   vm.NewTypedCFunction(CallMessage, "CallMessage()"),
		"sender":    vm.NewTypedCFunction(CallSender, "CallSender()"),
		"target":    vm.NewTypedCFunction(CallTarget, "CallTarget()"),
		"type":      vm.NewString("Call"),
	}
	SetSlot(vm.Core, "Call", &Call{Object: *vm.ObjectWith(slots)})
}

// CallActivated is a Call method.
//
// activated returns the activated slot.
func CallActivated(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Activated
}

// CallArgAt is a Call method.
//
// argAt returns the nth argument to the call, or nil if n is out of bounds.
// The argument is returned as the original message and is not evaluated.
func CallArgAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Call).Msg
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return m.ArgAt(int(v.Value))
}

// CallArgCount is a Call method.
//
// argCount returns the number of arguments passed in the call.
func CallArgCount(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(len(target.(*Call).Msg.Args)))
}

// CallEvalArgAt is a Call method.
//
// evalArgAt evaluates the nth argument to the call.
func CallEvalArgAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Call).Msg
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return m.EvalArgAt(vm, locals, int(v.Value))
}

// CallMessage is a Call method.
//
// message returns the message that caused the call.
func CallMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Msg
}

// CallSender is a Call method.
//
// sender returns the object which sent the call.
func CallSender(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Sender
}

// CallTarget is a Call method.
//
// target is the object which was the target of the call.
func CallTarget(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Target
}
