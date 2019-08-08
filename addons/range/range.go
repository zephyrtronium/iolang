package iorange

import (
	"fmt"
	"github.com/zephyrtronium/iolang"
	"math"

	. "github.com/zephyrtronium/iolang/addons"
)

// Range yields the terms of a linear sequence.
type Range struct {
	Object

	Index, Last int64
	Start, Step float64
}

// NewRange creates a new Range with the given start, stop, and step values.
func NewRange(vm *VM, start, stop, step float64) *Range {
	r := &Range{Object: *vm.AddonInstance("Range")}
	r.SetRange(start, stop, step)
	return r
}

// Activate returns the range.
func (r *Range) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return r, NoStop
}

// Clone creates a clone of the range with the same status.
func (r *Range) Clone() Interface {
	return &Range{
		Object: Object{Slots: Slots{}, Protos: []Interface{r}},
		Index:  r.Index,
		Last:   r.Last,
		Start:  r.Start,
		Step:   r.Step,
	}
}

// SetRange sets up the range with the given start, stop, and step.
func (r *Range) SetRange(start, stop, step float64) {
	var last int64
	if step > 0 {
		v := math.Ceil((stop - start) / step)
		if math.IsInf(v, 0) || math.IsNaN(v) {
			last = math.MaxInt64
		} else {
			last = int64(v)
		}
	} else {
		v := math.Floor((stop - start) / step)
		if math.IsInf(v, 0) || math.IsNaN(v) {
			last = math.MaxInt64
		} else {
			last = int64(v)
		}
	}
	r.Index = 0
	r.Last = last
	r.Start = start
	r.Step = step
}

// Value returns the current value of the range. This succeeds regardless of
// whether the range cursor is in-bounds.
func (r *Range) Value() float64 {
	return r.Start + float64(r.Index)*r.Step
}

// Next computes the current range value and increments the cursor. If the value
// is in the range, then ok is true.
func (r *Range) Next() (v float64, ok bool) {
	v = r.Value()
	ok = r.Index <= r.Last
	if ok {
		r.Index++
	}
	return v, ok
}

// Previous computes the decrements the cursor and computes the range's new
// value. If the value is in the range, then ok is true.
func (r *Range) Previous() (v float64, ok bool) {
	ok = r.Index > 0
	if ok {
		r.Index--
	}
	return r.Value(), ok
}

// At is a Range method.
//
// at returns the nth value of the range.
func At(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	k := int64(n.Value)
	if k < 0 || k > r.Last {
		return vm.RaiseException("index out of bounds")
	}
	return vm.NewNumber(r.Start + float64(k)*r.Step), NoStop
}

// Contains is a Range method.
//
// contains returns true if the given value occurs exactly in the range.
func Contains(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	var k float64
	if r.Step > 0 {
		k = (v.Value - r.Start) / r.Step
	} else {
		k = (v.Value + r.Start) / r.Step
	}
	if k < 0 || k > float64(r.Last) {
		return vm.False, NoStop
	}
	return vm.IoBool(k == float64(int(k))), NoStop
}

// First is a Range method.
//
// first moves the range's cursor to the beginning and returns its value.
func First(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	r.Index = 0
	return vm.NewNumber(r.Value()), NoStop
}

// Foreach is a Range method.
//
// foreach performs a loop for each element of the range.
func Foreach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	r := target.(*Range)
	kn, vn, hkn, hvn, ev := iolang.ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("Range foreach requires 1, 2, or 3 args")
	}
	pk := r.Index
	defer func() { r.Index = pk }()
	r.Index = 0
	for x, ok := r.Next(); ok; x, ok = r.Next() {
		v := vm.NewNumber(x)
		if hvn {
			vm.SetSlot(locals, vn, v)
			if hkn {
				vm.SetSlot(locals, kn, vm.NewNumber(float64(r.Index)))
			}
			result, control = ev.Eval(vm, locals)
		} else {
			result, control = ev.Send(vm, v, locals)
		}
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, control
		default:
			panic(fmt.Sprintf("iolang/range: invalid Stop: %v", control))
		}
	}
	return result, control
}

// Index is a Range method.
//
// index returns the number of terms yielded from the range.
func Index(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	return vm.NewNumber(float64(r.Index)), NoStop
}

// IndexOf is a Range method.
//
// indexOf returns the index of the range that would produce a particular value,
// or nil if none would.
func IndexOf(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	v, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	var k float64
	if r.Step > 0 {
		k = (v.Value - r.Start) / r.Step
	} else {
		k = (v.Value + r.Start) / r.Step
	}
	if k < 0 || k > float64(r.Last) || k != float64(int64(k)) {
		return vm.Nil, NoStop
	}
	return vm.NewNumber(k), NoStop
}

// Last is a Range method.
//
// last moves the range's cursor to the end and returns its value.
func Last(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	r.Index = r.Last
	return vm.NewNumber(r.Value()), NoStop
}

// Next is a Range method.
//
// next increments the range's cursor. Returns self if the cursor is still in
// bounds and nil otherwise.
func Next(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	_, ok := r.Next()
	if ok {
		return target, NoStop
	}
	return vm.Nil, NoStop
}

// Previous is a Range method.
//
// previous decrements the range's cursor. Returns self if the cursor is still
// in bounds and nil otherwise.
func Previous(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	_, ok := r.Previous()
	if ok {
		return target, NoStop
	}
	return vm.Nil, NoStop
}

// Rewind is a Range method.
//
// rewind returns the range to its first value and returns the range.
func Rewind(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	r.Index = 0
	return target, NoStop
}

// setIndex is a Range method.
//
// setIndex seeks the range to a new index.
func SetIndex(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	k, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	r.Index = int64(k.Value)
	return target, NoStop
}

// SetRange is a Range method.
//
// setRange sets the range to have the given start, stop, and step values.
func SetRange(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	var start, stop, step float64
	a, err, control := msg.NumberArgAt(vm, locals, 0)
	if control != NoStop {
		return err, control
	}
	b, err, control := msg.NumberArgAt(vm, locals, 1)
	if control != NoStop {
		return err, control
	}
	start = a.Value
	stop = b.Value
	if msg.ArgCount() == 2 {
		if stop >= start {
			step = 1
		} else {
			step = -1
		}
	} else {
		c, err, control := msg.NumberArgAt(vm, locals, 2)
		if control != NoStop {
			return err, control
		}
		step = c.Value
	}
	r.SetRange(start, stop, step)
	return target, NoStop
}

// Size is a Range method.
//
// size returns the number of steps the range can take.
func Size(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	return vm.NewNumber(float64(r.Last)), NoStop
}

// Value is a Range method.
//
// value returns the range's current value.
func Value(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	r := target.(*Range)
	return vm.NewNumber(r.Value()), NoStop
}
