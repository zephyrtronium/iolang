package iorange

import (
	"fmt"
	"github.com/zephyrtronium/iolang"
	"math"
)

// Range yields the terms of a linear sequence.
type Range struct {
	iolang.Object

	Index, Last int64
	Start, Step float64
}

// NewRange creates a new Range with the given start, stop, and step values.
func NewRange(vm *iolang.VM, start, stop, step float64) *Range {
	r := &Range{Object: *vm.AddonInstance("Range")}
	r.SetRange(start, stop, step)
	return r
}

// Activate returns the range.
func (r *Range) Activate(vm *iolang.VM, target, locals, context iolang.Interface, msg *iolang.Message) iolang.Interface {
	return r
}

// Clone creates a clone of the range with the same status.
func (r *Range) Clone() iolang.Interface {
	return &Range{
		Object: iolang.Object{Slots: iolang.Slots{}, Protos: []iolang.Interface{r}},
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
func At(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	n, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return err
	}
	k := int64(n.Value)
	if k < 0 || k > r.Last {
		return vm.RaiseException("index out of bounds")
	}
	return vm.NewNumber(r.Start + float64(k)*r.Step)
}

// Contains is a Range method.
//
// contains returns true if the given value occurs exactly in the range.
func Contains(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	v, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return err
	}
	var k float64
	if r.Step > 0 {
		k = (v.Value - r.Start) / r.Step
	} else {
		k = (v.Value + r.Start) / r.Step
	}
	if k < 0 || k > float64(r.Last) {
		return vm.False
	}
	return vm.IoBool(k == float64(int(k)))
}

// First is a Range method.
//
// first moves the range's cursor to the beginning and returns its value.
func First(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	r.Index = 0
	return vm.NewNumber(r.Value())
}

// Foreach is a Range method.
//
// foreach performs a loop for each element of the range.
func Foreach(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) (result iolang.Interface) {
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
			iolang.SetSlot(locals, vn, v)
			if hkn {
				iolang.SetSlot(locals, kn, vm.NewNumber(float64(r.Index)))
			}
			result = ev.Eval(vm, locals)
		} else {
			result = ev.Send(vm, v, locals)
		}
		if rr, ok := iolang.CheckStop(result, iolang.NoStop); !ok {
			switch s := rr.(iolang.Stop); s.Status {
			case iolang.ContinueStop:
				result = s.Result
			case iolang.BreakStop:
				return s.Result
			case iolang.ReturnStop, iolang.ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("range: invalid Stop: %#v", s))
			}
		}
	}
	return result
}

// Index is a Range method.
//
// index returns the number of terms yielded from the range.
func Index(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	return vm.NewNumber(float64(r.Index))
}

// Last is a Range method.
//
// last moves the range's cursor to the end and returns its value.
func Last(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	r.Index = r.Last
	return vm.NewNumber(r.Value())
}

// Next is a Range method.
//
// next increments the range's cursor. Returns self if the cursor is still in
// bounds and nil otherwise.
func Next(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	_, ok := r.Next()
	if ok {
		return target
	}
	return vm.Nil
}

// Previous is a Range method.
//
// previous decrements the range's cursor. Returns self if the cursor is still
// in bounds and nil otherwise.
func Previous(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	_, ok := r.Previous()
	if ok {
		return target
	}
	return vm.Nil
}

// Rewind is a Range method.
//
// rewind returns the range to its first value and returns the range.
func Rewind(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	r.Index = 0
	return target
}

// SetRange is a Range method.
//
// setRange sets the range to have the given start, stop, and step values.
func SetRange(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	var start, stop, step float64
	a, err := msg.NumberArgAt(vm, locals, 0)
	if err != nil {
		return err
	}
	b, err := msg.NumberArgAt(vm, locals, 1)
	if err != nil {
		return err
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
		c, err := msg.NumberArgAt(vm, locals, 2)
		if err != nil {
			return err
		}
		step = c.Value
	}
	r.SetRange(start, stop, step)
	return target
}

// Value is a Range method.
//
// value returns the range's current value.
func Value(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) iolang.Interface {
	r := target.(*Range)
	return vm.NewNumber(r.Value())
}
