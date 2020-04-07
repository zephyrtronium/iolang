package internal

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

	// ExitStop indicates that the VM has been force-closed and all coroutines
	// must exit.
	ExitStop

	// PauseStop tells a coroutine to stop execution until receiving a
	// ResumeStop. While paused, a coroutine is considered dead (the scheduler
	// will stop if all coroutines are paused), but it will still accept other
	// stops.
	PauseStop
	// ResumeStop tells a coroutine to unpause. If it is not paused, it will
	// instead yield.
	ResumeStop
)

var stopNames = [...]string{"normal", "continue", "break", "return", "exception", "exit", "pause", "resume"}

// String returns a string representation of the Stop.
func (s Stop) String() string {
	if s < NoStop || s > ResumeStop {
		return fmt.Sprintf("Stop(%d)", s)
	}
	return stopNames[s]
}

// Err returns nil if s is NoStop or an error value if s is ContinueStop,
// BreakStop, ReturnStop, ExceptionStop, or ExitStop. Panics otherwise.
func (s Stop) Err() error {
	switch s {
	case NoStop:
		return nil
	case ContinueStop, BreakStop, ReturnStop, ExceptionStop, ExitStop:
		return stopError(s)
	default:
		panic(fmt.Sprintf("iolang: invalid Stop: %v", s))
	}
}

type stopError Stop

func (err stopError) Error() string {
	return Stop(err).String()
}

// RemoteStop is a wrapped object and control flow status for sending to coros.
type RemoteStop struct {
	Result  *Object
	Control Stop
}

// Stop sends a Stop to the VM, causing the innermost message evaluation loop
// to exit with the given value and Stop. Stop does nothing if stop is NoStop,
// and it panics if stop is not ContinueStop, BreakStop, ReturnStop,
// ExceptionStop, or ExitStop. Returns result. If stop is not ExceptionStop or
// ExitStop and a Stop is already pending, e.g. from another coroutine, then
// this method instead does nothing and returns nil.
func (vm *VM) Stop(result *Object, stop Stop) *Object {
	switch stop {
	case NoStop:
		return result
	case ContinueStop, BreakStop, ReturnStop:
		select {
		case vm.Control <- RemoteStop{result, stop}:
			return result
		default:
			return nil
		}
	case ExceptionStop, ExitStop:
		// Always exit or raise exceptions. If there is already a value in
		// vm.Control, then simply sending the stop will block; then, if the
		// stop is being raised from within the currently executing VM (which
		// is the usual case), then the existing signal won't be drained, so
		// the goroutine is deadlocked against itself. Launching a new
		// goroutine to send the stop may cause the stop to actually be
		// registered at any random time if the number of existing goroutines
		// exceeds GOMAXPROCS, which would make debugging exceedingly
		// difficult. To ensure consistently valid behavior, we have to remove
		// any existing value from vm.Control and then send the stop.
		select {
		case s := <-vm.Control:
			if s.Control == ExitStop {
				// Never replace an ExitStop. Re-send it.
				result, stop = s.Result, s.Control
			}
			vm.Control <- RemoteStop{result, stop}
		case vm.Control <- RemoteStop{result, stop}: // do nothing
		}
		return result
	default:
		panic(fmt.Errorf("iolang: invalid Stop: %v", stop))
	}
}

// Status checks the VM's control flow channel and returns any pending signal,
// or (result, NoStop) if there is none.
func (vm *VM) Status(result *Object) (*Object, Stop) {
	select {
	case r := <-vm.Control:
		return r.Result, r.Control
	default:
		return result, NoStop
	}
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
func ObjectFor(vm *VM, target, locals *Object, msg *Message) (result *Object) {
	var (
		low, high float64
		exc       *Object
		v         float64
		step      = 1.0
		m         = msg.ArgAt(3)
		stop      Stop
	)
	switch msg.ArgCount() {
	case 5:
		if v, exc, stop = msg.NumberArgAt(vm, locals, 3); stop != NoStop {
			return vm.Stop(exc, stop)
		}
		step = v
		m = msg.ArgAt(4)
		fallthrough
	case 4:
		if v, exc, stop = msg.NumberArgAt(vm, locals, 1); stop != NoStop {
			return vm.Stop(exc, stop)
		}
		low = v
		if v, exc, stop = msg.NumberArgAt(vm, locals, 2); stop != NoStop {
			return vm.Stop(exc, stop)
		}
		high = v
		ctrname := msg.ArgAt(0).Name()
		i := low
		for {
			if step > 0 {
				if i > high {
					break
				}
			} else {
				if i < high {
					break
				}
			}
			vm.SetSlot(locals, ctrname, vm.NewNumber(i))
			result, stop = m.Eval(vm, locals)
			switch stop {
			case NoStop, ContinueStop: // do nothing
			case BreakStop:
				return result
			case ReturnStop, ExceptionStop, ExitStop:
				return vm.Stop(result, stop)
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", stop))
			}
			i += step
		}
	default:
		return vm.RaiseExceptionf("Object for requires 4 or 5 arguments")
	}
	return result
}

// ObjectWhile is an Object method.
//
// while performs a loop as long as a condition, its first argument, evaluates
// to true.
func ObjectWhile(vm *VM, target, locals *Object, msg *Message) *Object {
	if err := msg.AssertArgCount("Object while", 2); err != nil {
		return vm.IoError(err)
	}
	cond := msg.ArgAt(0)
	m := msg.ArgAt(1)
	result := vm.Nil
	for {
		c, stop := cond.Eval(vm, locals)
		if stop != NoStop {
			return vm.Stop(c, stop)
		}
		if !vm.AsBool(c) {
			return result
		}
		result, stop = m.Eval(vm, locals)
		switch stop {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop, ExitStop:
			return vm.Stop(result, stop)
		default:
			panic(fmt.Errorf("iolang: invalid Stop: %w", stop.Err()))
		}
	}
}

// ObjectLoop is an Object method.
//
// loop performs a loop.
func ObjectLoop(vm *VM, target, locals *Object, msg *Message) *Object {
	if err := msg.AssertArgCount("Object loop", 1); err != nil {
		return vm.IoError(err)
	}
	m := msg.ArgAt(0)
	for {
		result, control := m.Eval(vm, locals)
		switch control {
		case NoStop, ContinueStop: // do nothing
		case BreakStop:
			return result
		case ReturnStop, ExceptionStop, ExitStop:
			return vm.Stop(result, control)
		default:
			panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
		}
	}
}

// ObjectContinue is an Object method.
//
// continue immediately returns to the beginning of the current loop.
func ObjectContinue(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	return vm.Stop(v, ContinueStop)
}

// ObjectBreak is an Object method.
//
// break ceases execution of a loop and returns a value from it.
func ObjectBreak(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	// We still want to check against only NoStop so that an expression like
	// break(continue) uses the first-evaluated control flow first.
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	return vm.Stop(v, BreakStop)
}

// ObjectReturn is an Object method.
//
// return ceases execution of a block and returns a value from it.
func ObjectReturn(vm *VM, target, locals *Object, msg *Message) *Object {
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(v, stop)
	}
	return vm.Stop(v, ReturnStop)
}

// ObjectIf is an Object method.
//
// if evaluates its first argument, then evaluates the second if the first was
// true or the third if it was false.
func ObjectIf(vm *VM, target, locals *Object, msg *Message) *Object {
	c, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(c, stop)
	}
	if vm.AsBool(c) {
		if msg.ArgCount() < 2 {
			// Return true to support `if(condition) then(action)`.
			return vm.True
		}
		return vm.Stop(msg.EvalArgAt(vm, locals, 1))
	}
	if msg.ArgCount() < 3 {
		// Return false to support `if(c, message) elseif(c, message)`.
		return vm.False
	}
	// Even if only two arguments are supplied, this will evaluate to vm.Nil.
	return vm.Stop(msg.EvalArgAt(vm, locals, 2))
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
