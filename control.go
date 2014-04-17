package iolang

import (
	"fmt"
	"math"
)

type Stop struct {
	Status StopStatus
	// This is really ugly, but I couldn't think of a better way to return
	// the result of a broken loop.
	Result Interface
}

// Stops are not objects, but let's pretend anyway so they can be returned
// from things like Message.Eval().
func (Stop) SP() *Object      { panic("iolang: a Stop is not an Object!") }
func (Stop) Clone() Interface { panic("iolang: a Stop is not an Object!") }
func (Stop) isIoObject()      {}

type StopStatus int

const (
	// I have resisted the urge to name this DontStopBelieving.
	// Unfortunately, I have just realized it is now a rape joke.
	NoStop StopStatus = iota
	ReturnStop
	BreakStop
	ContinueStop
)

func ObjectFor(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	var (
		low, high float64
		err       error
		v         *Number
		step      = 1.0
		m         = msg.ArgAt(3)
	)
	switch len(msg.Args) {
	case 4:
		if v, err = msg.NumberArgAt(vm, locals, 3); err != nil {
			return vm.IoError(err)
		}
		step = v.Value
		m = msg.ArgAt(4)
		fallthrough
	case 3:
		if v, err = msg.NumberArgAt(vm, locals, 1); err != nil {
			return vm.IoError(err)
		}
		low = v.Value
		if v, err = msg.NumberArgAt(vm, locals, 2); err != nil {
			return vm.IoError(err)
		}
		high = v.Value
		ctrname := msg.ArgAt(0).Name()
		i := vm.NewNumber(low)
		for {
			if step > 0 {
				if i.Value > high {
					break
				}
			} else {
				if i.Value < high {
					break
				}
			}
			SetSlot(locals, ctrname, i)
			result = m.Eval(vm, locals)
			switch rr := result.(type) {
			case Stop:
				switch rr.Status {
				case ReturnStop:
					// We have to return the stop so the outer Block knows.
					return rr
				case BreakStop:
					// Here, the stop only affects this loop, so we can return
					// the actual result.
					return rr.Result
				case ContinueStop:
					// do nothing
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
				}
			case error:
				return result
			}
			i = vm.NewNumber(i.Value + step)
		}
	default:
		return vm.NewExceptionf("Object for requires 3 or 4 arguments")
	}
	return result
}

func ObjectWhile(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	if err := msg.AssertArgCount("Object while", 2); err != nil {
		return vm.IoError(err)
	}
	cond := msg.ArgAt(0)
	m := msg.ArgAt(1)
	for {
		c := cond.Eval(vm, locals)
		if IsIoError(c) {
			return c
		}
		if vm.AsBool(c) {
			return result
		}
		result = m.Eval(vm, locals)
		if IsIoError(result) {
			return result
		}
		if stop, ok := result.(Stop); ok {
			switch stop.Status {
			case ReturnStop:
				return stop
			case BreakStop:
				return stop.Result
			case ContinueStop:
				// do nothing
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", stop))
			}
		}
	}
}

func ObjectLoop(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	if err := msg.AssertArgCount("Object loop", 1); err != nil {
		return vm.IoError(err)
	}
	m := msg.ArgAt(0)
	for {
		result = m.Eval(vm, locals)
		if err, ok := result.(error); ok {
			return vm.IoError(err)
		}
		if stop, ok := result.(Stop); ok {
			switch stop.Status {
			case ReturnStop:
				return stop
			case BreakStop:
				return stop.Result
			case ContinueStop:
				// do nothing
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", stop))
			}
		}
	}
}

func ObjectReturn(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: ReturnStop, Result: msg.ArgAt(0).Eval(vm, locals)}
}

func ObjectBreak(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: BreakStop, Result: msg.ArgAt(0).Eval(vm, locals)}
}

func ObjectContinue(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: ContinueStop}
}

func ObjectIf(vm *VM, target, locals Interface, msg *Message) Interface {
	// The behavior of this does not exactly mimic that of the original Io
	// implementation in strange cases:
	// expression	Io		iolang
	// if()			false	nil
	// if(false)	false	nil
	// if(true)		true	nil
	// (The behavior implemented here is actually the documented behavior of
	// Io, though, and frankly, it makes more sense.)
	if vm.AsBool(msg.ArgAt(0).Eval(vm, locals)) {
		return msg.ArgAt(1).Eval(vm, locals)
	}
	// Even if only two arguments are supplied, this will evaluate to vm.Nil.
	return msg.ArgAt(2).Eval(vm, locals)
}

func NumberRepeat(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	if len(msg.Args) < 1 {
		return vm.NewException("Number repeat requires 1 or 2 arguments")
	}
	counter, eval := msg.ArgAt(0), msg.ArgAt(1)
	c := counter.Symbol.Text
	if eval == nil {
		// One argument was supplied.
		counter, eval = nil, counter
	}
	max := int(math.Ceil(target.(*Number).Value))
	for i := 0; i < max; i++ {
		if counter != nil {
			SetSlot(locals, c, vm.NewNumber(float64(i)))
		}
		result = eval.Eval(vm, locals)
		if IsIoError(result) {
			return result
		}
		if stop, ok := result.(Stop); ok {
			switch stop.Status {
			case ReturnStop:
				return stop
			case BreakStop:
				return stop.Result
			case ContinueStop:
				// do nothing
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", stop))
			}
		}
	}
	return result
}
