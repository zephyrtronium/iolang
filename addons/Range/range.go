//go:generate go run github.com/zephyrtronium/iolang/cmd/mkaddon addon.yaml addon.go
//go:generate gofmt -s -w addon.go

// Package Range provides an efficient iterator over linear numeric sequences.
package Range

import (
	"fmt"
	"math"

	"github.com/zephyrtronium/iolang"

	. "github.com/zephyrtronium/iolang/addons"
)

// Range yields the terms of a linear sequence.
type Range struct {
	Index, Last int64
	Start, Step float64
}

// RangeTag is the Tag for Range objects.
const RangeTag = iolang.BasicTag("github.com/zephyrtronium/iolang/addons/range")

// New creates a new Range object with the given start, stop, and step
// values.
func New(vm *VM, start, stop, step float64) *Object {
	return vm.ObjectWith(nil, vm.AddonProto("Range"), With(start, stop, step), RangeTag)
}

// With creates a Range value with the given start, stop, and step values.
func With(start, stop, step float64) Range {
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
	return Range{Last: last, Start: start, Step: step}
}

// Value returns the current value of the range. This succeeds regardless of
// whether the range cursor is in-bounds.
func (r Range) Value() float64 {
	return r.Start + float64(r.Index)*r.Step
}

// Next computes the current range value and increments the cursor. If the value
// is in the range, then ok is true. Returns the new range value.
func (r Range) Next() (v float64, ok bool, s Range) {
	v = r.Value()
	ok = r.Index <= r.Last
	if ok {
		r.Index++
	}
	return v, ok, r
}

// Previous decrements the cursor and computes the range's new value. If the
// value is in the range, then ok is true. Returns the new range value.
func (r Range) Previous() (v float64, ok bool, s Range) {
	ok = r.Index > 0
	if ok {
		r.Index--
	}
	return r.Value(), ok, r
}

// At is a Range method.
//
// at returns the nth value of the range.
func At(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	k := int64(n)
	if k < 0 || k > r.Last {
		return vm.RaiseExceptionf("index %d out of bounds", k)
	}
	return vm.NewNumber(r.Start + float64(k)*r.Step)
}

// Contains is a Range method.
//
// contains returns true if the given value occurs exactly in the range.
func Contains(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	v, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	var k float64
	if r.Step > 0 {
		k = (v - r.Start) / r.Step
	} else {
		k = (v + r.Start) / r.Step
	}
	if k < 0 || k > float64(r.Last) {
		return vm.False
	}
	return vm.IoBool(k == float64(int(k)))
}

// First is a Range method.
//
// first moves the range's cursor to the beginning and returns its value.
func First(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	r.Index = 0
	target.Value = r
	target.Unlock()
	return vm.NewNumber(r.Value())
}

// Foreach is a Range method.
//
// foreach performs a loop for each element of the range.
func Foreach(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	kn, vn, hkn, hvn, ev := iolang.ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseExceptionf("Range foreach requires 1, 2, or 3 args")
	}
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	r.Index = 0
	var control Stop
	for x, ok, r := r.Next(); ok; x, ok, r = r.Next() {
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
			return result
		case ReturnStop, ExceptionStop, ExitStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang/range: invalid Stop: %v", control))
		}
	}
	return vm.Stop(result, control)
}

// Index is a Range method.
//
// index returns the number of terms yielded from the range.
func Index(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	return vm.NewNumber(float64(r.Index))
}

// IndexOf is a Range method.
//
// indexOf returns the index of the range that would produce a particular value,
// or nil if none would.
func IndexOf(vm *VM, target, locals *Object, msg *Message) *Object {
	v, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	var k float64
	if r.Step > 0 {
		k = (v - r.Start) / r.Step
	} else {
		k = (v + r.Start) / r.Step
	}
	if k < 0 || k > float64(r.Last) || k != float64(int64(k)) {
		return vm.Nil
	}
	return vm.NewNumber(k)
}

// Last is a Range method.
//
// last moves the range's cursor to the end and returns its value.
func Last(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	r.Index = r.Last
	target.Value = r
	target.Unlock()
	return vm.NewNumber(r.Value())
}

// Next is a Range method.
//
// next increments the range's cursor. Returns self if the cursor is still in
// bounds and nil otherwise.
func Next(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	_, ok, r := r.Next()
	target.Value = r
	target.Unlock()
	if ok {
		return target
	}
	return vm.Nil
}

// Previous is a Range method.
//
// previous decrements the range's cursor. Returns self if the cursor is still
// in bounds and nil otherwise.
func Previous(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	_, ok, r := r.Previous()
	target.Value = r
	target.Unlock()
	if ok {
		return target
	}
	return vm.Nil
}

// Rewind is a Range method.
//
// rewind returns the range to its first value and returns the range.
func Rewind(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	r.Index = 0
	target.Value = r
	target.Unlock()
	return target
}

// SetIndex is a Range method.
//
// setIndex seeks the range to a new index.
func SetIndex(vm *VM, target, locals *Object, msg *Message) *Object {
	k, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	r := target.Value.(Range)
	r.Index = int64(k)
	target.Value = r
	target.Unlock()
	return target
}

// SetRange is a Range method.
//
// setRange sets the range to have the given start, stop, and step values.
func SetRange(vm *VM, target, locals *Object, msg *Message) *Object {
	var start, stop, step float64
	start, exc, control := msg.NumberArgAt(vm, locals, 0)
	if control != NoStop {
		return vm.Stop(exc, control)
	}
	stop, exc, control = msg.NumberArgAt(vm, locals, 1)
	if control != NoStop {
		return vm.Stop(exc, control)
	}
	if msg.ArgCount() == 2 {
		if stop >= start {
			step = 1
		} else {
			step = -1
		}
	} else {
		c, exc, control := msg.NumberArgAt(vm, locals, 2)
		if control != NoStop {
			return vm.Stop(exc, control)
		}
		step = c
	}
	target.Lock()
	target.Value = With(start, stop, step)
	target.Unlock()
	return target
}

// Size is a Range method.
//
// size returns the number of steps the range can take.
func Size(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	return vm.NewNumber(float64(r.Last))
}

// Value is a Range method.
//
// value returns the range's current value.
func Value(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	r := target.Value.(Range)
	target.Unlock()
	return vm.NewNumber(r.Value())
}
