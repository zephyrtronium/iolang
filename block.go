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
func (b *Block) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	// If this block isn't actually activatable, then it should be the result
	// of activation.
	if !b.Activatable {
		return b
	}
	return b.reallyActivate(vm, target, locals, msg)
}

func (b *Block) reallyActivate(vm *VM, target, locals Interface, msg *Message) Interface {
	scope := b.Self
	if scope == nil {
		scope = target
	}
	call := vm.NewCall(locals, b, msg, target)
	blkLocals := vm.NewLocals(scope, call)
	for i, arg := range b.ArgNames {
		if x := msg.EvalArgAt(vm, locals, i); x != nil {
			if r, ok := CheckStop(x, LoopStops); ok {
				x = r
			} else {
				return r
			}
			SetSlot(blkLocals, arg, x)
		} else {
			SetSlot(blkLocals, arg, vm.Nil)
		}
	}
	upto := ReturnStop
	if b.PassStops {
		upto = NoStop
	}
	result, _ := CheckStop(b.Message.Eval(vm, blkLocals), upto)
	return result
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
	SetSlot(lc, "self", self)
	SetSlot(lc, "call", call)
	return lc
}

func (vm *VM) initBlock() {
	var exemplar *Block
	slots := Slots{
		"argumentNames":    vm.NewTypedCFunction(BlockArgumentNames, exemplar),
		"asString":         vm.NewTypedCFunction(BlockAsString, exemplar),
		"call":             vm.NewTypedCFunction(BlockCall, exemplar),
		"message":          vm.NewTypedCFunction(BlockMessage, exemplar),
		"passStops":        vm.NewTypedCFunction(BlockPassStops, exemplar),
		"performOn":        vm.NewTypedCFunction(BlockPerformOn, exemplar),
		"scope":            vm.NewTypedCFunction(BlockScope, exemplar),
		"setArgumentNames": vm.NewTypedCFunction(BlockSetArgumentNames, exemplar),
		"setMessage":       vm.NewTypedCFunction(BlockSetMessage, exemplar),
		"setPassStops":     vm.NewTypedCFunction(BlockSetPassStops, exemplar),
		"setScope":         vm.NewTypedCFunction(BlockSetScope, exemplar),
		"type":             vm.NewString("Block"),
	}
	slots["code"] = slots["asString"]
	SetSlot(vm.Core, "Block", &Block{Object: *vm.ObjectWith(slots)})
}

func (vm *VM) initLocals() {
	slots := Slots{
		"forward": vm.NewCFunction(LocalsForward),
	}
	SetSlot(vm.Core, "Locals", vm.ObjectWith(slots))
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
func ObjectBlock(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := Block{
		Object:   *vm.CoreInstance("Block"),
		Message:  msg.ArgAt(len(msg.Args) - 1),
		Self:     locals,
		ArgNames: make([]string, len(msg.Args)-1),
	}
	for i, arg := range msg.Args[:len(msg.Args)-1] {
		blk.ArgNames[i] = arg.Name()
	}
	return &blk
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
func ObjectMethod(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := ObjectBlock(vm, target, locals, msg).(*Block)
	blk.Activatable = true
	blk.Self = nil
	return blk
}

// BlockArgumentNames is a Block method.
//
// argumentNames returns a list of the argument names of the block.
func BlockArgumentNames(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	l := make([]Interface, len(blk.ArgNames))
	for i, n := range blk.ArgNames {
		l[i] = vm.NewString(n)
	}
	return vm.NewList(l...)
}

// BlockAsString is a Block method.
//
// asString creates a string representation of an object.
func BlockAsString(vm *VM, target, locals Interface, msg *Message) Interface {
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
	return vm.NewString(b.String())
}

// BlockCall is a Block method.
//
// call activates a block.
func BlockCall(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Block).reallyActivate(vm, target, locals, msg)
}

// BlockMessage is a Block method.
//
// message returns the block's message.
func BlockMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Block).Message
}

// BlockPassStops is a Block method.
//
// passStops returns whether the block returns control flow signals upward.
func BlockPassStops(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.IoBool(target.(*Block).PassStops)
}

// BlockPerformOn is a Block method.
//
// performOn executes the block in the context of the argument. Optional
// arguments may be supplied to give non-default locals and message.
func BlockPerformOn(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	nt, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return nt
	}
	nl := locals
	nm := msg
	if len(msg.Args) > 1 {
		nl, ok = CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
		if !ok {
			return nl
		}
		if len(msg.Args) > 2 {
			r, ok := CheckStop(msg.EvalArgAt(vm, locals, 2), LoopStops)
			if !ok {
				return nm
			}
			if nm, ok = r.(*Message); !ok {
				return vm.RaiseException("argument 2 to performOn must evaluate to a Message")
			}
		}
	}
	return blk.reallyActivate(vm, nt, nl, nm)
}

// BlockScope is a Block method.
//
// scope returns the scope of the block, or nil if the block is a method.
func BlockScope(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	if blk.Self == nil {
		return vm.Nil
	}
	return blk.Self
}

// BlockSetArgumentNames is a Block method.
//
// setArgumentNames changes the names of the arguments of the block. This does
// not modify the block code, so some arguments might change to context lookups
// and vice-versa.
func BlockSetArgumentNames(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	l := make([]string, len(msg.Args))
	for i := range msg.Args {
		s, err := msg.StringArgAt(vm, locals, i)
		if err != nil {
			return vm.RaiseException("all arguments to setArgumentNames must be strings")
		}
		l[i] = s.String()
	}
	blk.ArgNames = l
	return target
}

// BlockSetMessage is a Block method.
//
// setMessage changes the message executed by the block.
func BlockSetMessage(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	m, ok := r.(*Message)
	if !ok {
		return vm.RaiseException("argument to setMessage must evaluate to a Message")
	}
	blk.Message = m
	return target
}

// BlockSetPassStops is a Block method.
//
// setPassStops changes whether the block allows control flow signals to
// propagate out to the block's caller.
func BlockSetPassStops(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
	}
	blk.PassStops = vm.AsBool(r)
	return target
}

// BlockSetScope is a Block method.
//
// setScope changes the context of the block. If nil, the block becomes a
// method.
func BlockSetScope(vm *VM, target, locals Interface, msg *Message) Interface {
	blk := target.(*Block)
	r, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return r
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
func LocalsForward(vm *VM, target, locals Interface, msg *Message) Interface {
	// We do not want a proto's self, so do the lookup manually.
	lc := target.SP()
	lc.L.Lock()
	self, ok := lc.Slots["self"]
	lc.L.Unlock()
	if ok && self != target {
		return msg.Send(vm, self, locals)
	}
	return vm.Nil
}
