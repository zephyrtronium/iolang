package iolang

import (
	"fmt"
	"math"
)

// A Stop is a pseudo-object used to implement control flow in Io.
type Stop struct {
	Status StopStatus
	Result Interface
}

// Stops are not objects, but let's pretend anyway so they can be returned
// from things like Message.Eval().

func (Stop) SP() *Object      { panic("iolang: a Stop is not an Object!") }
func (Stop) Clone() Interface { panic("iolang: a Stop is not an Object!") }
func (Stop) isIoObject()      {}

// StopStatus represents the reason for flow control.
type StopStatus int

// Control flow reasons.
const (
	// I have resisted the urge to name this DontStopBelieving.
	NoStop StopStatus = iota
	ReturnStop
	BreakStop
	ContinueStop
)

// ObjectFor is an Object method.
//
// for performs a loop with a counter. For example, to print each number from 1
// to 3 inclusive:
//
//   io> for(x, 1, 3, x println)
//
// Or, to print each third number from 10 to 25:
//
//   for(x, 10, 25, 3, x println)
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

// ObjectWhile is an Object method.
//
// while performs a loop as long as a condition, its first argument, evaluates
// to true.
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

// ObjectLoop is an Object method.
//
// loop performs a loop.
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

// ObjectReturn is an Object method.
//
// return ceases execution of a block and returns a value from it.
func ObjectReturn(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: ReturnStop, Result: msg.ArgAt(0).Eval(vm, locals)}
}

// ObjectBreak is an Object method.
//
// break ceases execution of a loop and returns a value from it.
func ObjectBreak(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: BreakStop, Result: msg.ArgAt(0).Eval(vm, locals)}
}

// ObjectContinue is an Object method.
//
// continue immediately returns to the beginning of the current loop.
func ObjectContinue(vm *VM, target, locals Interface, msg *Message) Interface {
	return Stop{Status: ContinueStop}
}

// ObjectIf is an Object method.
//
// if evaluates its first argument, then evaluates the second if the first was
// true or the third if it was false.
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

// NumberRepeat is a Number method.
//
// repeat performs a loop the given number of times.
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

// ListForeach is a List method.
//
// foreach performs a loop on each item of a list in order, optionally setting
// index and value variables.
func ListForeach(vm *VM, target, locals Interface, msg *Message) Interface {
	var kn, vn string
	var hkn, hvn bool
	var ev *Message
	if len(msg.Args) == 3 {
		kn = msg.ArgAt(0).Name()
		vn = msg.ArgAt(1).Name()
		ev = msg.ArgAt(2)
		hkn, hvn = true, true
	} else if len(msg.Args) == 2 {
		vn = msg.ArgAt(0).Name()
		ev = msg.ArgAt(1)
		hvn = true
	} else if len(msg.Args) == 1 {
		ev = msg.ArgAt(0)
	} else {
		return vm.NewException("foreach requires 1, 2, or 3 arguments")
	}
	l := target.(*List)
	var result Interface
	for k, v := range l.Value {
		if hvn {
			SetSlot(locals, vn, v)
			if hkn {
				SetSlot(locals, kn, vm.NewNumber(float64(k)))
			}
		}
		result = ev.Eval(vm, locals)
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
