package iolang

/*
// Call contains information on how a Block was activated.
type Call struct {
	Object
	// Sender is the locals in the context of the activation.
	Sender Interface
	// Activated is the (Block) object which is being activated.
	Activated Interface
	// Msg is the message received to activate the block.
	Msg *Message
	// Target is the object to which the message was sent.
	Target Interface
	// Context is the object which actually owned the activated slot.
	Context Interface
	// Coroutine is the coroutine running this call.
	Coroutine *VM
}

// Activate returns the call.
func (c *Call) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return c
}

// Clone creates a clone of this call with the same values. The clone holds the
// same message pointer.
func (c *Call) Clone() Interface {
	return &Call{
		Object:    Object{Slots: Slots{}, Protos: []Interface{c}},
		Sender:    c.Sender,
		Activated: c.Activated,
		Msg:       c.Msg,
		Target:    c.Target,
		Coroutine: c.Coroutine,
	}
}
*/

// NewCall creates a Call object sent from sender to the target's actor using
// the message msg.
func (vm *VM) NewCall(sender, actor Interface, msg *Message, target, context Interface) Interface {
	c := vm.CoreInstance("Call")
	c.Slots = Slots{
		"activated":   actor,
		"coroutine":   vm,
		"message":     msg,
		"sender":      sender,
		"slotContext": context,
		"target":      target,
	}
	return c
}

func (vm *VM) initCall() {
	slots := Slots{
		"argAt":     vm.NewCFunction(CallArgAt, nil),
		"argCount":  vm.NewCFunction(CallArgCount, nil),
		"evalArgAt": vm.NewCFunction(CallEvalArgAt, nil),
		"type":      vm.NewString("Call"),
	}
	vm.SetSlot(vm.Core, "Call", vm.ObjectWith(slots))
}

// CallArgAt is a Call method.
//
// argAt returns the nth argument to the call, or nil if n is out of bounds.
// The argument is returned as the original message and is not evaluated.
func CallArgAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, proto := vm.GetSlot(target, "message")
	if proto == nil {
		return vm.RaiseException("no message slot for Call argAt")
	}
	m, ok := s.(*Message)
	if !ok {
		return vm.RaiseException("call message must be Message, not " + vm.TypeName(s))
	}
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r := m.ArgAt(int(v.Value))
	if r != nil {
		return r, NoStop
	}
	return vm.Nil, NoStop
}

// CallArgCount is a Call method.
//
// argCount returns the number of arguments passed in the call.
func CallArgCount(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, proto := vm.GetSlot(target, "message")
	if proto == nil {
		return vm.RaiseException("no message slot for Call argCount")
	}
	m, ok := s.(*Message)
	if !ok {
		return vm.RaiseException("call message must be Message, not " + vm.TypeName(s))
	}
	return vm.NewNumber(float64(m.ArgCount())), NoStop
}

// CallEvalArgAt is a Call method.
//
// evalArgAt evaluates the nth argument to the call in the context of the
// sender.
func CallEvalArgAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, proto := vm.GetSlot(target, "message")
	if proto == nil {
		return vm.RaiseException("no message slot for Call evalArgAt")
	}
	m, ok := s.(*Message)
	if !ok {
		return vm.RaiseException("call message must be Message, not " + vm.TypeName(s))
	}
	snd, proto := vm.GetSlot(target, "sender")
	if proto == nil {
		return vm.RaiseException("no sender slot for Call evalArgAt")
	}
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return m.EvalArgAt(vm, snd, int(v.Value))
}
