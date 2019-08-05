package iorange

import (
	"github.com/zephyrtronium/iolang"
	"strings"

	. "github.com/zephyrtronium/iolang/addons"
)

type rangeAddon struct{}

func (rangeAddon) AddonName() string {
	return "Range"
}

func (rangeAddon) Instance(vm *VM) Interface {
	var kind *Range
	slots := iolang.Slots{
		"at":       vm.NewCFunction(At, kind),
		"contains": vm.NewCFunction(Contains, kind),
		"first":    vm.NewCFunction(First, kind),
		"foreach":  vm.NewCFunction(Foreach, kind),
		"index":    vm.NewCFunction(Index, kind),
		"indexOf":  vm.NewCFunction(IndexOf, kind),
		"last":     vm.NewCFunction(Last, kind),
		"next":     vm.NewCFunction(Next, kind),
		"previous": vm.NewCFunction(Previous, kind),
		"rewind":   vm.NewCFunction(Rewind, kind),
		"setIndex": vm.NewCFunction(SetIndex, kind),
		"setRange": vm.NewCFunction(SetRange, kind),
		"size":     vm.NewCFunction(Size, kind),
		"type":     vm.NewString("Range"),
		"value":    vm.NewCFunction(Value, kind),
	}
	return &Range{Object: *vm.ObjectWith(slots)}
}

func (rangeAddon) Script(vm *VM) *Message {
	msg, err := vm.Parse(strings.NewReader(script), "<init Range>")
	if err != nil {
		panic(err)
	}
	if err := vm.OpShuffle(msg); err != nil {
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
