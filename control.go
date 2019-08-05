package iolang

import "fmt"

// Stop represents the reason for flow control.
type Stop int

// Control flow reasons.
const (
	// NoStop indicates normal execution. Coroutines also use it to cause each
	// other to yield remotely.
	NoStop Stop = iota
	// ContinueStop should be interpreted by loops as a signal to restart the
	// loop immediately.
	ContinueStop
	// BreakStop should be interpreted by loops as a signal to exit the loop.
	BreakStop
	// ReturnStop should be interpreted by loops and blocks as a signal to
	// exit.
	ReturnStop
	// ExceptionStop should be interpreted by loops, blocks, and CFunctions as
	// a signal to exit.
	ExceptionStop

	// PauseStop tells a coroutine to stop execution until receiving a
	// ResumeStop. While paused, a coroutine is considered dead (the scheduler
	// will stop if all coroutines are paused), but it will still accept other
	// stops.
	PauseStop
	// ResumeStop tells a coroutine to unpause. If it is not paused, it will
	// instead yield.
	ResumeStop
)

var stopNames = [...]string{"normal", "continue", "break", "return", "exception", "pause", "resume"}

func (s Stop) String() string {
	if s < NoStop || s > ResumeStop {
		return fmt.Sprintf("Stop(%d)", s)
	}
	return stopNames[s]
}

// RemoteStop is a wrapped object and control flow status for sending to coros.
type RemoteStop struct {
	Result  Interface
	Control Stop
}

// ObjectFor is an Object method.
//
// for performs a loop with a counter. For example, to print each number from 1
// to 3 inclusive:
//
//   io> for(x, 1, 3, x println)
//
// Or, to print each third number from 10 to 25:
//
//   io> for(x, 10, 25, 3, x println)
func ObjectFor(vm *VM, target, locals Interface, msg *Message) (result Interface, stop Stop) {
	var (
		low, high float64
		err       Interface
		v         *Number
		step      = 1.0
		m         = msg.ArgAt(3)
	)
	switch msg.ArgCount() {
	case 5:
		if v, err, stop = msg.NumberArgAt(vm, locals, 3); stop != NoStop {
			return err, stop
		}
		step = v.Value
		m = msg.ArgAt(4)
		fallthrough
	case 4:
		if v, err, stop = msg.NumberArgAt(vm, locals, 1); stop != NoStop {
			return err, stop
		}
		low = v.Value
		if v, err, stop = msg.NumberArgAt(vm, locals, 2); stop != NoStop {
			return err, stop
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
			locals.SetSlot(ctrname, i)
			result, stop = m.Eval(vm, locals)
			switch stop {
			case NoStop, ContinueStop: // do nothing
			case BreakStop:
				return result, NoStop
			case ReturnStop, ExceptionStop:
				return result, stop
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", stop))
			}
			i = vm.NewNumber(i.Value + step)
		}
	default:
		return vm.RaiseExceptionf("Object for requires 4 or 5 arguments")
	}
	return result, NoStop
}

// ObjectWhile is an Object method.
//
// while performs a loop as long as a condition, its first argument, evaluates
// to true.
func ObjectWhile(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	if err := msg.AssertArgCount("Object while", 2); err != nil {
		return vm.IoError(err)
	}
	cond := msg.ArgAt(0)
	m := msg.ArgAt(1)
	for {
		c, stop := cond.Eval(vm, locals)
		if stop != NoStop {
			return c, stop
		}
		if !vm.AsBool(c) {
			return result, NoStop
		}
		result, stop = m.Eval(vm, locals)
		switch stop {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, stop
		}
	}
}

// ObjectLoop is an Object method.
//
// loop performs a loop.
func ObjectLoop(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	if err := msg.AssertArgCount("Object loop", 1); err != nil {
		return vm.IoError(err)
	}
	m := msg.ArgAt(0)
	for {
		result, control = m.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result, NoStop
		case ReturnStop, ExceptionStop:
			return result, control
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
}

// ObjectContinue is an Object method.
//
// continue immediately returns to the beginning of the current loop.
func ObjectContinue(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	return v, ContinueStop
}

// ObjectBreak is an Object method.
//
// break ceases execution of a loop and returns a value from it.
func ObjectBreak(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	// We still want to check against only NoStop so that an expression like
	// break(continue) uses the first-evaluated control flow first.
	if stop != NoStop {
		return v, stop
	}
	return v, BreakStop
}

// ObjectReturn is an Object method.
//
// return ceases execution of a block and returns a value from it.
func ObjectReturn(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	return v, ReturnStop
}

// ObjectIf is an Object method.
//
// if evaluates its first argument, then evaluates the second if the first was
// true or the third if it was false.
func ObjectIf(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	c, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return c, stop
	}
	if vm.AsBool(c) {
		if msg.ArgCount() < 2 {
			// Return true to support `if(condition) then(action)`.
			return vm.True, NoStop
		}
		return msg.EvalArgAt(vm, locals, 1)
	}
	if msg.ArgCount() < 3 {
		// Return false to support `if(c, message) elseif(c, message)`.
		return vm.False, NoStop
	}
	// Even if only two arguments are supplied, this will evaluate to vm.Nil.
	return msg.EvalArgAt(vm, locals, 2)
}

// ForeachArgs gets the arguments for a foreach method utilizing the standard
// foreach([[key,] value,] message) syntax.
func ForeachArgs(msg *Message) (kn, vn string, hkn, hvn bool, ev *Message) {
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
	}
	return
}
