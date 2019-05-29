package iolang

import (
	"fmt"
	"io"
	"math"
)

// A Stop is a pseudo-object used to implement control flow in Io.
type Stop struct {
	Status StopStatus
	Result Interface
}

// Stops are not objects, but let's pretend anyway so they can be returned
// from things like Message.Eval().

func (Stop) SP() *Object { panic("iolang: a Stop is not an Object!") }
func (Stop) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	panic("iolang: a Stop is not an Object!")
}
func (Stop) Clone() Interface { panic("iolang: a Stop is not an Object!") }
func (Stop) isIoObject()      {}

// CheckStop checks whether an object is a control flow stop pseudo-object,
// up to a given level: NoStop will only succeed if the value is not a Stop,
// ContinueStop will succeed if the value is a normal object or a Stop
// generated by ObjectContinue, and so forth. If the check succeeds, the
// returned value will be the Stop's result, and ok will be true; otherwise,
// the returned value will be the Stop, and ok will be false.
func CheckStop(v Interface, upto StopStatus) (r Interface, ok bool) {
	if s, ok := v.(Stop); ok {
		if s.Status <= NoStop {
			// NoStop is only allowed for remotely yielding.
			panic(fmt.Sprintf("iolang: invalid Stop: %#v", s))
		}
		if s.Status <= upto {
			return s.Result, true
		}
		return s, false
	}
	return v, true
}

// StopStatus represents the reason for flow control.
type StopStatus int

// Control flow reasons.
const (
	// NoStop is a constant to allow loops to use CheckStop to check for both
	// continues and breaks. Coroutines also use it to tell each other to yield
	// remotely. It should not be used for normal execution.
	NoStop StopStatus = iota
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

	// LoopStops is a synonym for BreakStop provided to allow a more
	// descriptive name for non-loops to check stops intended for loops.
	LoopStops = BreakStop
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
//   io> for(x, 10, 25, 3, x println)
func ObjectFor(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	var (
		low, high float64
		stop      Interface
		v         *Number
		step      = 1.0
		m         = msg.ArgAt(3)
	)
	switch len(msg.Args) {
	case 5:
		if v, stop = msg.NumberArgAt(vm, locals, 3); stop != nil {
			return stop
		}
		step = v.Value
		m = msg.ArgAt(4)
		fallthrough
	case 4:
		if v, stop = msg.NumberArgAt(vm, locals, 1); stop != nil {
			return stop
		}
		low = v.Value
		if v, stop = msg.NumberArgAt(vm, locals, 2); stop != nil {
			return stop
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
			if rr, ok := CheckStop(result, NoStop); !ok {
				// Due to the implementation of CheckStop, we still have to
				// check the stop status manually, but we are certain that it
				// is a Stop.
				switch s := rr.(Stop); s.Status {
				case ContinueStop:
					result = s.Result
				case BreakStop:
					return s.Result
				case ReturnStop, ExceptionStop:
					return rr
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
				}
			}
			i = vm.NewNumber(i.Value + step)
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
func ObjectWhile(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	if err := msg.AssertArgCount("Object while", 2); err != nil {
		return vm.IoError(err)
	}
	cond := msg.ArgAt(0)
	m := msg.ArgAt(1)
	for {
		// It's possible for a loop condition to evaluate to a Stop. Io's
		// behavior, due to the way it implements control flow as a coroutine
		// attribute, is to respect them wherever they occur, but I believe
		// that is more likely to lead to unexplained stalling. We will instead
		// accept continues and breaks as normal values.
		c, ok := CheckStop(cond.Eval(vm, locals), LoopStops)
		if !ok {
			return c
		}
		if !vm.AsBool(c) {
			return result
		}
		result = m.Eval(vm, locals)
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
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
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				// result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
}

// ObjectContinue is an Object method.
//
// continue immediately returns to the beginning of the current loop.
func ObjectContinue(vm *VM, target, locals Interface, msg *Message) Interface {
	v := msg.EvalArgAt(vm, locals, 0)
	if rr, ok := CheckStop(v, ContinueStop); ok {
		return Stop{Status: ContinueStop, Result: rr}
	}
	return v
}

// ObjectBreak is an Object method.
//
// break ceases execution of a loop and returns a value from it.
func ObjectBreak(vm *VM, target, locals Interface, msg *Message) Interface {
	v := msg.EvalArgAt(vm, locals, 0)
	if rr, ok := CheckStop(v, BreakStop); ok {
		return Stop{Status: BreakStop, Result: rr}
	}
	return v
}

// ObjectReturn is an Object method.
//
// return ceases execution of a block and returns a value from it.
func ObjectReturn(vm *VM, target, locals Interface, msg *Message) Interface {
	v := msg.EvalArgAt(vm, locals, 0)
	if rr, ok := CheckStop(v, ReturnStop); ok {
		return Stop{Status: ReturnStop, Result: rr}
	}
	return v
}

// ObjectIf is an Object method.
//
// if evaluates its first argument, then evaluates the second if the first was
// true or the third if it was false.
func ObjectIf(vm *VM, target, locals Interface, msg *Message) Interface {
	c, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), NoStop)
	if !ok {
		return c
	}
	if vm.AsBool(c) {
		if len(msg.Args) < 2 {
			// Return true to support `if(condition) then(action)`.
			return vm.True
		}
		return msg.EvalArgAt(vm, locals, 1)
	}
	if len(msg.Args) < 3 {
		// Return false to support `if(c, message) elseif(c, message)`.
		return vm.False
	}
	// Even if only two arguments are supplied, this will evaluate to vm.Nil.
	return msg.EvalArgAt(vm, locals, 2)
}

// NumberRepeat is a Number method.
//
// repeat performs a loop the given number of times.
func NumberRepeat(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	if len(msg.Args) < 1 {
		return vm.RaiseException("Number repeat requires 1 or 2 arguments")
	}
	counter, eval := msg.ArgAt(0), msg.ArgAt(1)
	c := counter.Name()
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
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
	return result
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

// ListForeach is a List method.
//
// foreach performs a loop on each item of a list in order, optionally setting
// index and value variables.
func ListForeach(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 1, 2, or 3 arguments")
	}
	l := target.(*List)
	for k, v := range l.Value {
		if hvn {
			SetSlot(locals, vn, v)
			if hkn {
				SetSlot(locals, kn, vm.NewNumber(float64(k)))
			}
			result = ev.Eval(vm, locals)
		} else {
			result = ev.Send(vm, v, locals)
		}
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
	return result
}

// ListReverseForeach is a List method.
//
// reverseForeach performs a loop on each item of a list in order, optionally
// setting index and value variables, proceeding from the end of the list to
// the start.
func ListReverseForeach(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if !hvn {
		return vm.RaiseException("reverseForeach requires 2 or 3 arguments")
	}
	l := target.(*List)
	for k := len(l.Value) - 1; k >= 0; k-- {
		v := l.Value[k]
		SetSlot(locals, vn, v)
		if hkn {
			SetSlot(locals, kn, vm.NewNumber(float64(k)))
		}
		result = ev.Eval(vm, locals)
		if rr, ok := CheckStop(result, NoStop); !ok {
			switch s := rr.(Stop); s.Status {
			case ContinueStop:
				result = s.Result
			case BreakStop:
				return s.Result
			case ReturnStop, ExceptionStop:
				return rr
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
			}
		}
	}
	return result
}

// FileForeach is a File method.
//
// foreach executes a message for each byte of the file.
func FileForeach(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 2 or 3 arguments")
	}
	f := target.(*File)
	if info, err := f.File.Stat(); err == nil && info.Mode().IsRegular() {
		// Regular file, so we can read into a buffer and then seek back if we
		// encounter a Stop.
		k, j, n := 0, 0, 0
		b := make([]byte, 4096)
		defer func() {
			f.File.Seek(int64(j-n), io.SeekCurrent)
		}()
		for {
			n, err = f.File.Read(b)
			j = 0
			for _, c := range b[:n] {
				v := vm.NewNumber(float64(c))
				if hvn {
					SetSlot(locals, vn, v)
					if hkn {
						SetSlot(locals, kn, vm.NewNumber(float64(k)))
					}
				}
				result = ev.Send(vm, v, locals)
				if rr, ok := CheckStop(result, NoStop); !ok {
					switch s := rr.(Stop); s.Status {
					case ContinueStop:
						result = s.Result
					case BreakStop:
						return s.Result
					case ReturnStop, ExceptionStop:
						return rr
					default:
						panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
					}
				}
				k++
				j++
			}
			if err == io.EOF {
				f.EOF = true
				break
			}
			if err != nil {
				return vm.IoError(err)
			}
		}
	} else {
		// Other than a regular file. We can't necessarily seek around, so we
		// have to read one byte at a time.
		b := []byte{0}
		for k := 0; err != io.EOF; k++ {
			_, err = f.File.Read(b)
			if err != nil {
				if err == io.EOF {
					f.EOF = true
					break
				}
				return vm.IoError(err)
			}
			v := vm.NewNumber(float64(b[0]))
			if hvn {
				SetSlot(locals, vn, v)
				if hkn {
					SetSlot(locals, kn, vm.NewNumber(float64(k)))
				}
			}
			result = ev.Send(vm, v, locals)
			if rr, ok := CheckStop(result, NoStop); !ok {
				switch s := rr.(Stop); s.Status {
				case ContinueStop:
					result = s.Result
				case BreakStop:
					return s.Result
				case ReturnStop, ExceptionStop:
					return rr
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
				}
			}
		}
	}
	return result
}

// FileForeachLine is a File method.
//
// foreachLine executes a message for each line of the file.
func FileForeachLine(vm *VM, target, locals Interface, msg *Message) (result Interface) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 1, 2, or 3 arguments")
	}
	f := target.(*File)
	k := 0
	// f.ReadLine implements the same logic as FileForeach above.
	for {
		line, err := f.ReadLine()
		if line != nil {
			v := vm.NewSequence(line, true, "latin1")
			if hvn {
				SetSlot(locals, vn, v)
				if hkn {
					SetSlot(locals, kn, vm.NewNumber(float64(k)))
				}
			}
			result = ev.Send(vm, v, locals)
			if rr, ok := CheckStop(result, NoStop); !ok {
				switch s := rr.(Stop); s.Status {
				case ContinueStop:
					result = s.Result
				case BreakStop:
					return s.Result
				case ReturnStop, ExceptionStop:
					return rr
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %#v", rr))
				}
			}
			if err != nil {
				break
			}
			k++
		}
	}
	return result
}
