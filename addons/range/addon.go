package iorange

import (
	"github.com/zephyrtronium/iolang"
	"strings"
)

type rangeAddon struct{}

func (rangeAddon) AddonName() string {
	return "Range"
}

func (rangeAddon) Instance(vm *iolang.VM) iolang.Interface {
	var r *Range
	slots := iolang.Slots{
		"at":       vm.NewTypedCFunction(At, r),
		"contains": vm.NewTypedCFunction(Contains, r),
		"first":    vm.NewTypedCFunction(First, r),
		"foreach":  vm.NewTypedCFunction(Foreach, r),
		"index":    vm.NewTypedCFunction(Index, r),
		"indexOf":  vm.NewTypedCFunction(IndexOf, r),
		"last":     vm.NewTypedCFunction(Last, r),
		"next":     vm.NewTypedCFunction(Next, r),
		"previous": vm.NewTypedCFunction(Previous, r),
		"rewind":   vm.NewTypedCFunction(Rewind, r),
		"setIndex": vm.NewTypedCFunction(SetIndex, r),
		"setRange": vm.NewTypedCFunction(SetRange, r),
		"size":     vm.NewTypedCFunction(Size, r),
		"type":     vm.NewString("Range"),
		"value":    vm.NewTypedCFunction(Value, r),
	}
	return &Range{Object: *vm.ObjectWith(slots)}
}

func (rangeAddon) Script(vm *iolang.VM) *iolang.Message {
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
func OpenAddon(vm *iolang.VM) iolang.Addon {
	return rangeAddon{}
}
