package iolang

import (
	"bytes"
	"strings"
)

// A Block is a reusable, lexically scoped message. Essentially a function.
//
// NOTE: Unlike most other primitives in iolang, Block values are NOT
// synchronized. It is a race condition to modify a block that might be in use,
// such as 'call activated' or any block or method object in a scope other than
// the locals of the innermost currently executing block.
type Block struct {
	// Message is the message that the block performs.
	Message *Message
	// Self is the block's lexical scope. If nil, then the block is a method,
	// and the scope becomes the receiver of the message that activated the
	// block.
	Self *Object
	// ArgNames is the list of argument slot names.
	ArgNames []string

	// Activatable controls whether the block performs its message or returns
	// itself when activated.
	Activatable bool
	// PassStops controls whether the block resends control flow signals that
	// are returned from evaluating its message.
	PassStops bool
}

// tagBlock is the Tag type for Block objects.
type tagBlock struct{}

func (tagBlock) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	// If this block isn't actually activatable, then it should be the result
	// of activation.
	b := self.Value.(*Block)
	if !b.Activatable {
		return self
	}
	return vm.ActivateBlock(self, target, locals, context, msg)
}

func (tagBlock) CloneValue(value interface{}) interface{} {
	b := value.(*Block)
	return &Block{
		Message:     b.Message.DeepCopy(),
		Self:        b.Self,
		ArgNames:    append([]string{}, b.ArgNames...),
		Activatable: b.Activatable,
		PassStops:   b.PassStops,
	}
}

func (tagBlock) String() string {
	return "Block"
}

// BlockTag is the Tag for Block objects. Activate activates the block if it is
// activatable and otherwise returns the block. CloneValue creates a new block
// with a deep copy of the parent's message.
var BlockTag tagBlock

// ActivateBlock activates a block directly, regardless of the value of its
// Activatable flag. Panics if blk is not a Block object.
func (vm *VM) ActivateBlock(blk, target, locals, context *Object, msg *Message) *Object {
	b := blk.Value.(*Block)
	scope := b.Self
	if scope == nil {
		scope = target
	}
	call := vm.NewCall(locals, blk, msg, target, context)
	blkLocals := vm.NewLocals(scope, call)
	// We don't want to be holding the block's lock while evaluating its code
	// or arguments in case any of them refer to the block itself. Copy out the
	// information we need while we are still holding the lock.
	args := append([]string{}, b.ArgNames...)
	m := b.Message
	pass := b.PassStops
	for i, arg := range args {
		x, stop := msg.EvalArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.Stop(x, stop)
		}
		blkLocals.SetSlot(arg, x)
	}
	result, stop := m.Eval(vm, blkLocals)
	if pass || stop == ExceptionStop {
		return vm.Stop(result, stop)
	}
	return result
}

// NewBlock creates a new Block object for a message. If scope is nil, then the
// returned block is a method, and it is activatable; otherwise, it is a
// lexically scoped block that is not activatable.
func (vm *VM) NewBlock(msg *Message, scope *Object, args ...string) *Object {
	if msg == nil {
		msg = vm.CachedMessage(vm.Nil)
	}
	value := &Block{
		Message:     msg,
		Self:        scope,
		ArgNames:    args,
		Activatable: scope == nil,
	}
	return vm.ObjectWith(nil, vm.CoreProto("Block"), value, BlockTag)
}

// NewLocals instantiates a Locals object for a block activation.
func (vm *VM) NewLocals(self, call *Object) *Object {
	slots := Slots{
		"self": self,
		"call": call,
	}
	return vm.ObjectWith(slots, vm.CoreProto("Locals"), nil, nil)
}

// NewCall creates a Call object sent from sender to the target's actor using
// the message msg.
func (vm *VM) NewCall(sender, actor *Object, msg *Message, target, context *Object) *Object {
	slots := Slots{
		"activated":   actor,
		"coroutine":   vm.Coro,
		"message":     vm.MessageObject(msg),
		"sender":      sender,
		"slotContext": context,
		"target":      target,
	}
	return vm.ObjectWith(slots, vm.CoreProto("Call"), nil, nil)
}

func (vm *VM) initBlock() {
	slots := Slots{
		"argumentNames":    vm.NewCFunction(BlockArgumentNames, BlockTag),
		"asString":         vm.NewCFunction(BlockAsString, BlockTag),
		"call":             vm.NewCFunction(BlockCall, BlockTag),
		"message":          vm.NewCFunction(BlockMessage, BlockTag),
		"passStops":        vm.NewCFunction(BlockPassStops, BlockTag),
		"performOn":        vm.NewCFunction(BlockPerformOn, BlockTag),
		"scope":            vm.NewCFunction(BlockScope, BlockTag),
		"setArgumentNames": vm.NewCFunction(BlockSetArgumentNames, BlockTag),
		"setMessage":       vm.NewCFunction(BlockSetMessage, BlockTag),
		"setPassStops":     vm.NewCFunction(BlockSetPassStops, BlockTag),
		"setScope":         vm.NewCFunction(BlockSetScope, BlockTag),
		"type":             vm.NewString("Block"),
	}
	slots["code"] = slots["asString"]
	vm.coreInstall("Block", slots, &Block{}, BlockTag)
	// Call doesn't have anything special, so we'll set it up here.
	vm.coreInstall("Call", nil, nil, nil)
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
	// Don't use coreInstall because Locals have no protos.
	vm.Core.SetSlot("Locals", vm.ObjectWith(slots, nil, nil, nil))
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
func ObjectBlock(vm *VM, target, locals *Object, msg *Message) *Object {
	n := msg.ArgCount()
	if n == 0 {
		return vm.NewBlock(nil, locals)
	}
	args := make([]string, n-1)
	for i, arg := range msg.Args[:n-1] {
		args[i] = arg.Name()
	}
	return vm.NewBlock(msg.ArgAt(n-1), locals, args...)
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
func ObjectMethod(vm *VM, target, locals *Object, msg *Message) *Object {
	n := msg.ArgCount()
	if n == 0 {
		return vm.NewBlock(nil, nil)
	}
	args := make([]string, n-1)
	for i, arg := range msg.Args[:n-1] {
		args[i] = arg.Name()
	}
	return vm.NewBlock(msg.ArgAt(n-1), nil, args...)
}

// BlockArgumentNames is a Block method.
//
// argumentNames returns a list of the argument names of the block.
func BlockArgumentNames(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	l := make([]*Object, len(blk.ArgNames))
	for i, n := range blk.ArgNames {
		l[i] = vm.NewString(n)
	}
	return vm.NewList(l...)
}

// BlockAsString is a Block method.
//
// asString creates a string representation of an object.
func BlockAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
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
	return vm.NewString(b.String())
}

// BlockCall is a Block method.
//
// call activates a block.
func BlockCall(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.ActivateBlock(target, locals, locals, locals, msg)
}

// BlockMessage is a Block method.
//
// message returns the block's message.
func BlockMessage(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	return vm.MessageObject(blk.Message)
}

// BlockPassStops is a Block method.
//
// passStops returns whether the block returns control flow signals upward.
func BlockPassStops(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	return vm.IoBool(blk.PassStops)
}

// BlockPerformOn is a Block method.
//
// performOn executes the block in the context of the argument. Optional
// arguments may be supplied to give non-default locals and message.
func BlockPerformOn(vm *VM, target, locals *Object, msg *Message) *Object {
	nt, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(nt, stop)
	}
	nl := locals
	nm := msg
	nc := nt
	if msg.ArgCount() > 1 {
		nl, stop = msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(nl, stop)
		}
		if msg.ArgCount() > 2 {
			var exc *Object
			nm, exc, stop = msg.MessageArgAt(vm, locals, 2)
			if stop != NoStop {
				return vm.Stop(exc, stop)
			}
			if msg.ArgCount() > 3 {
				nc, stop = msg.EvalArgAt(vm, locals, 3)
				if stop != NoStop {
					return vm.Stop(nc, stop)
				}
			}
		}
	}
	return vm.ActivateBlock(target, nt, nl, nc, nm)
}

// BlockScope is a Block method.
//
// scope returns the scope of the block, or nil if the block is a method.
func BlockScope(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	return blk.Self
}

// BlockSetArgumentNames is a Block method.
//
// setArgumentNames changes the names of the arguments of the block. This does
// not modify the block code, so some arguments might change to context lookups
// and vice-versa.
func BlockSetArgumentNames(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	l := make([]string, len(msg.Args))
	for i := range msg.Args {
		s, exc, stop := msg.StringArgAt(vm, locals, i)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		l[i] = s
	}
	blk.ArgNames = l
	return target
}

// BlockSetMessage is a Block method.
//
// setMessage changes the message executed by the block.
func BlockSetMessage(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	m, exc, stop := msg.MessageArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	blk.Message = m
	return target
}

// BlockSetPassStops is a Block method.
//
// setPassStops changes whether the block allows control flow signals to
// propagate out to the block's caller.
func BlockSetPassStops(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	blk.PassStops = vm.AsBool(r)
	return target
}

// BlockSetScope is a Block method.
//
// setScope changes the context of the block. If nil, the block becomes a
// method (but whether it is activatable does not change).
func BlockSetScope(vm *VM, target, locals *Object, msg *Message) *Object {
	blk := target.Value.(*Block)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	if r == vm.Nil {
		blk.Self = nil
	} else {
		blk.Self = r
	}
	return target
}

// LocalsForward is a Locals method.
//
// forward handles messages to which the object does not respond.
func LocalsForward(vm *VM, target, locals *Object, msg *Message) *Object {
	self, ok := target.GetLocalSlot("self")
	if ok && self != target {
		return vm.Stop(vm.Perform(self, locals, msg))
	}
	return vm.Nil
}

// LocalsUpdateSlot is a Locals method.
//
// updateSlot changes the value of an existing slot.
func LocalsUpdateSlot(vm *VM, target, locals *Object, msg *Message) *Object {
	slot, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	_, proto := target.GetSlot(slot)
	if proto != nil {
		// The slot exists on the locals object.
		v, stop := msg.EvalArgAt(vm, locals, 1)
		if stop != NoStop {
			return vm.Stop(v, stop)
		}
		target.SetSlot(slot, v)
		return v
	}
	// If the slot doesn't exist on the locals, forward to self, which is the
	// block scope or method receiver.
	self, proto := target.GetSlot("self")
	if proto != nil {
		return vm.Stop(vm.Perform(self, locals, msg))
	}
	return vm.RaiseExceptionf("no slot named %s in %s", slot, vm.TypeName(target))
}
