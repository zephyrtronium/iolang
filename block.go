package iolang

import (
	"bytes"
	"strings"
)

// A Block is a reusable, portable set of executable messages. Essentially a
// function.
type Block struct {
	Object
	Message     *Message
	Self        Interface
	ArgNames    []string
	Activatable bool
	PassStops   bool
}

// Activate performs the messages in this block if the block is activatable.
// Otherwise, this block is returned.
func (b *Block) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	// If this block isn't actually activatable, then it should be the result
	// of activation.
	if !b.Activatable {
		return b, NoStop
	}
	return b.reallyActivate(vm, target, locals, context, msg)
}

func (b *Block) reallyActivate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	scope := b.Self
	if scope == nil {
		scope = target
	}
	call := vm.NewCall(locals, b, msg, target, context)
	blkLocals := vm.NewLocals(scope, call)
	for i, arg := range b.ArgNames {
		x, stop := msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return x, stop
		}
		blkLocals.SetSlot(arg, x)
	}
	result, stop := b.Message.Eval(vm, blkLocals)
	if b.PassStops || stop == ExceptionStop {
		return result, stop
	}
	return result, NoStop
}

// Clone creates a clone of the block with a deep copy of its message.
func (b *Block) Clone() Interface {
	return &Block{
		Object:      Object{Slots: Slots{}, Protos: []Interface{b}},
		Message:     b.Message.DeepCopy(),
		Self:        b.Self,
		ArgNames:    append([]string{}, b.ArgNames...),
		Activatable: b.Activatable,
		PassStops:   b.PassStops,
	}
}

// NewLocals instantiates a Locals object for a block activation.
func (vm *VM) NewLocals(self, call Interface) *Object {
	lc := vm.CoreInstance("Locals")
	lc.SetSlot("self", self)
	lc.SetSlot("call", call)
	return lc
}

func (vm *VM) initBlock() {
	var kind *Block
	slots := Slots{
		"argumentNames":    vm.NewCFunction(BlockArgumentNames, kind),
		"asString":         vm.NewCFunction(BlockAsString, kind),
		"call":             vm.NewCFunction(BlockCall, kind),
		"message":          vm.NewCFunction(BlockMessage, kind),
		"passStops":        vm.NewCFunction(BlockPassStops, kind),
		"performOn":        vm.NewCFunction(BlockPerformOn, kind),
		"scope":            vm.NewCFunction(BlockScope, kind),
		"setArgumentNames": vm.NewCFunction(BlockSetArgumentNames, kind),
		"setMessage":       vm.NewCFunction(BlockSetMessage, kind),
		"setPassStops":     vm.NewCFunction(BlockSetPassStops, kind),
		"setScope":         vm.NewCFunction(BlockSetScope, kind),
		"type":             vm.NewString("Block"),
	}
	slots["code"] = slots["asString"]
	vm.Core.SetSlot("Block", &Block{Object: *vm.ObjectWith(slots)})
}

func (vm *VM) initLocals() {
	// Locals have no protos, so that messages forward to self. Instead, they
	// have copies of each built-in Object slot.
	slots := make(Slots, len(vm.BaseObject.Slots)+2)
	for k, v := range vm.BaseObject.Slots {
		slots[k] = v
	}
	slots["forward"] = vm.NewCFunction(LocalsForward, nil)
	slots["updateSlot"] = vm.NewCFunction(LocalsUpdateSlot, nil)
	vm.Core.SetSlot("Locals", &Object{Slots: slots, Protos: []Interface{}})
}

// ObjectBlock is an Object method.
//
// block creates a block of messages. Argument names are supplied first, and
// the block's code is the last argument. For example, to create and call a
// block which adds 1 to its argument:
//
//   io> succ := block(x, x + 1)
//   block(x, x +(1))
//   io> succ call(3)
//   4
func ObjectBlock(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	if msg.ArgCount() == 0 {
		return &Block{
			Object:  *vm.CoreInstance("Block"),
			Message: vm.CachedMessage(vm.Nil),
			Self:    locals,
		}, NoStop
	}
	blk := Block{
		Object:   *vm.CoreInstance("Block"),
		Message:  msg.ArgAt(len(msg.Args) - 1),
		Self:     locals,
		ArgNames: make([]string, len(msg.Args)-1),
	}
	for i, arg := range msg.Args[:len(msg.Args)-1] {
		blk.ArgNames[i] = arg.Name()
	}
	return &blk, NoStop
}

// ObjectMethod is an Object method, which is less redundant than it sounds.
//
// method creates a block of messages referring to the method antecedent.
// Argument names are supplied first, and the method's code is the last
// argument. For example, to create and call a method on numbers which return
// the number 1 higher:
//
//   io> Number succ := Number method(+ 1)
//   method(+(1))
//   io> 3 succ
//   4
func ObjectMethod(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r, _ := ObjectBlock(vm, target, locals, msg)
	blk := r.(*Block)
	blk.Activatable = true
	blk.Self = nil
	return blk, NoStop
}

// BlockArgumentNames is a Block method.
//
// argumentNames returns a list of the argument names of the block.
func BlockArgumentNames(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	l := make([]Interface, len(blk.ArgNames))
	for i, n := range blk.ArgNames {
		l[i] = vm.NewString(n)
	}
	return vm.NewList(l...), NoStop
}

// BlockAsString is a Block method.
//
// asString creates a string representation of an object.
func BlockAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	b := bytes.Buffer{}
	if blk.Self == nil {
		b.WriteString("method(")
	} else {
		b.WriteString("block(")
	}
	b.WriteString(strings.Join(blk.ArgNames, ", "))
	if len(blk.ArgNames) > 0 {
		b.WriteByte(',')
	}
	b.WriteByte('\n')
	blk.Message.stringRecurse(vm, &b)
	b.WriteString("\n)")
	return vm.NewString(b.String()), NoStop
}

// BlockCall is a Block method.
//
// call activates a block.
func BlockCall(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target.(*Block).reallyActivate(vm, target, locals, locals, msg)
}

// BlockMessage is a Block method.
//
// message returns the block's message.
func BlockMessage(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target.(*Block).Message, NoStop
}

// BlockPassStops is a Block method.
//
// passStops returns whether the block returns control flow signals upward.
func BlockPassStops(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.IoBool(target.(*Block).PassStops), NoStop
}

// BlockPerformOn is a Block method.
//
// performOn executes the block in the context of the argument. Optional
// arguments may be supplied to give non-default locals and message.
func BlockPerformOn(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	nt, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return nt, stop
	}
	nl := locals
	nm := msg
	nc := nt
	if msg.ArgCount() > 1 {
		nl, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return nl, stop
		}
		if msg.ArgCount() > 2 {
			r, stop := msg.EvalArgAt(vm, locals, 2)
			if stop != NoStop {
				return r, stop
			}
			var ok bool
			if nm, ok = r.(*Message); !ok {
				return vm.RaiseException("argument 2 to performOn must evaluate to a Message")
			}
			if msg.ArgCount() > 3 {
				nc, stop = msg.EvalArgAt(vm, locals, 3)
				if stop != NoStop {
					return nc, stop
				}
			}
		}
	}
	return blk.reallyActivate(vm, nt, nl, nc, nm)
}

// BlockScope is a Block method.
//
// scope returns the scope of the block, or nil if the block is a method.
func BlockScope(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	if blk.Self == nil {
		return vm.Nil, NoStop
	}
	return blk.Self, NoStop
}

// BlockSetArgumentNames is a Block method.
//
// setArgumentNames changes the names of the arguments of the block. This does
// not modify the block code, so some arguments might change to context lookups
// and vice-versa.
func BlockSetArgumentNames(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	l := make([]string, len(msg.Args))
	for i := range msg.Args {
		s, _, stop := msg.StringArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.RaiseException("all arguments to setArgumentNames must be strings")
		}
		l[i] = s.String()
	}
	blk.ArgNames = l
	return target, NoStop
}

// BlockSetMessage is a Block method.
//
// setMessage changes the message executed by the block.
func BlockSetMessage(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	m, ok := r.(*Message)
	if !ok {
		return vm.RaiseException("argument to setMessage must evaluate to a Message")
	}
	blk.Message = m
	return target, NoStop
}

// BlockSetPassStops is a Block method.
//
// setPassStops changes whether the block allows control flow signals to
// propagate out to the block's caller.
func BlockSetPassStops(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	blk.PassStops = vm.AsBool(r)
	return target, NoStop
}

// BlockSetScope is a Block method.
//
// setScope changes the context of the block. If nil, the block becomes a
// method.
func BlockSetScope(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	blk := target.(*Block)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	if r == vm.Nil {
		blk.Self = nil
	} else {
		blk.Self = r
	}
	return target, NoStop
}

// LocalsForward is a Locals method.
//
// forward handles messages to which the object does not respond.
func LocalsForward(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	self, ok := target.GetLocalSlot("self")
	if ok && self != target {
		return vm.Perform(self, locals, msg)
	}
	return vm.Nil, NoStop
}

// LocalsUpdateSlot is a Locals method.
//
// updateSlot changes the value of an existing slot.
func LocalsUpdateSlot(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	name, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	slot := name.String()
	_, proto := target.GetSlot(slot)
	if proto != nil {
		// The slot exists on the locals object.
		v, stop := msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return v, stop
		}
		target.SetSlot(slot, v)
		return v, NoStop
	}
	// If the slot doesn't exist on the locals, forward to self, which is the
	// block scope or method receiver.
	self, proto := target.GetSlot("self")
	if proto != nil {
		return vm.Perform(self, locals, msg)
	}
	return vm.RaiseExceptionf("no slot named %s in %s", slot, vm.TypeName(target))
}
