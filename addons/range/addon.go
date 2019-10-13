package iorange

import (
	"strings"

	"github.com/zephyrtronium/iolang"

	. "github.com/zephyrtronium/iolang/addons"
)

type rangeAddon struct{}

func (rangeAddon) AddonName() string {
	return "Range"
}

func (rangeAddon) Instance(vm *VM) Interface {
	slots := iolang.Slots{
		"at":       vm.NewCFunction(At, RangeTag),
		"contains": vm.NewCFunction(Contains, RangeTag),
		"first":    vm.NewCFunction(First, RangeTag),
		"foreach":  vm.NewCFunction(Foreach, RangeTag),
		"index":    vm.NewCFunction(Index, RangeTag),
		"indexOf":  vm.NewCFunction(IndexOf, RangeTag),
		"last":     vm.NewCFunction(Last, RangeTag),
		"next":     vm.NewCFunction(Next, RangeTag),
		"previous": vm.NewCFunction(Previous, RangeTag),
		"rewind":   vm.NewCFunction(Rewind, RangeTag),
		"setIndex": vm.NewCFunction(SetIndex, RangeTag),
		"setRange": vm.NewCFunction(SetRange, RangeTag),
		"size":     vm.NewCFunction(Size, RangeTag),
		"type":     vm.NewString("Range"),
		"value":    vm.NewCFunction(Value, RangeTag),
	}
	return &Object{
		Slots:  slots,
		Protos: []*iolang.Object{vm.BaseObject},
		Value:  Range{},
		Tag:    RangeTag,
	}
}

func (rangeAddon) Script(vm *VM) *Message {
	msg, err := vm.Parse(strings.NewReader(script), "<init Range>")
	if err != nil {
		panic(err)
	}
	return msg
}

const script = `
asList := method(
	l := list()
	self foreach(v, l append(v))
)

map := method(call delegateToMethod(self asList, "mapInPlace"))

select := List getSlot("select")

slice := method(start, stop, step,
	l := list()
	step = step ifNilEval(1)
	for(i, start, stop, step, l append(self at(i)))
)

Core Number do(
	to := method(end, self toBy(end, 1))
	toBy := method(end, step, Range clone setRange(self, end, step))
)
`

// OpenAddon returns an object to load the addon.
func OpenAddon(vm *VM) iolang.Addon {
	return rangeAddon{}
}
