package iolang

type Call struct {
	Object
	Sender    Interface
	Activated Actor
	Msg       *Message
	Target    Interface
}

func (vm *VM) NewCall(sender Interface, actor Actor, msg *Message, target Interface) *Call {
	return &Call{
		Object:    Object{Slots: vm.DefaultSlots["Call"], Protos: []Interface{vm.BaseObject}},
		Sender:    sender,
		Activated: actor,
		Msg:       msg,
		Target:    target,
	}
}

func (vm *VM) initCall() {
	slots := Slots{
		"activated": vm.NewCFunction(CallActivated, "CallActivated()"),
		"argAt":     vm.NewCFunction(CallArgAt, "CallArgAt(n)"),
		"argCount":  vm.NewCFunction(CallArgCount, "CallArgCount()"),
		"evalArgAt": vm.NewCFunction(CallEvalArgAt, "CallEvalArgAt(n)"),
		"message":   vm.NewCFunction(CallMessage, "CallMessage()"),
		"sender":    vm.NewCFunction(CallSender, "CallSender()"),
		"target":    vm.NewCFunction(CallTarget, "CallTarget()"),
	}
	vm.DefaultSlots["Call"] = slots
}

func CallActivated(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Activated
}

func CallArgAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Call).Msg
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return m.ArgAt(int(v.Value))
}

func CallArgCount(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(len(target.(*Call).Msg.Args)))
}

func CallEvalArgAt(vm *VM, target, locals Interface, msg *Message) Interface {
	m := target.(*Call).Msg
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	return m.EvalArgAt(vm, locals, int(v.Value))
}

func CallMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Msg
}

func CallSender(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Sender
}

func CallTarget(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Call).Target
}
