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
}

// Activate performs the messages in this block if the block is activatable.
// Otherwise, this block is returned.
func (b *Block) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	// If this block isn't actually activatable, then it should be the result
	// of evaluation.
	if !b.Activatable {
		return b
	}
	return b.reallyActivate(vm, target, locals, msg)
}

func (b *Block) reallyActivate(vm *VM, target, locals Interface, msg *Message) Interface {
	blkLocals := &Object{Slots: Slots{}, Protos: []Interface{vm.BaseObject}}
	for i, arg := range b.ArgNames {
		if x := msg.ArgAt(i); x != nil {
			SetSlot(blkLocals, arg, x)
		} else {
			SetSlot(blkLocals, arg, vm.Nil)
		}
	}
	scope := b.Self
	if scope == nil {
		scope = target
	}
	call := vm.NewCall(locals, b, msg, target)
	SetSlot(blkLocals, "self", scope)
	SetSlot(blkLocals, "call", call)
	result := b.Message.Eval(vm, blkLocals)
	if stop, ok := result.(Stop); ok {
		return stop.Result
	}
	return result
}

func (vm *VM) initBlock() {
	slots := Slots{
		"asString": vm.NewCFunction(BlockAsString, "BlockAsString()"),
		"call":     vm.NewCFunction(BlockCall, "BlockCall(...)"),
	}
	vm.DefaultSlots["Block"] = slots
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
		Object:   Object{Slots: vm.DefaultSlots["Block"], Protos: []Interface{vm.BaseObject}},
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
