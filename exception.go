package iolang

import "fmt"

// An Exception is an Io exception.
type Exception struct {
	Object
	Stack []*Message
}

// Activate returns the exception.
func (e *Exception) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return e
}

// Clone creates a clone of the exception.
func (e *Exception) Clone() Interface {
	return &Exception{Object: Object{Slots: Slots{}, Protos: []Interface{e}}}
}

// NewException creates a new Io Exception with the given error message.
func (vm *VM) NewException(msg string) *Exception {
	e := Exception{Object: *vm.CoreInstance("Exception")}
	e.Slots["error"] = vm.NewString(msg)
	return &e
}

// NewExceptionf creates a new Io Exception with the given formatted error
// message.
func (vm *VM) NewExceptionf(format string, args ...interface{}) *Exception {
	return vm.NewException(fmt.Sprintf(format, args...))
}

// RaiseException returns NewException(msg).Raise().
func (vm *VM) RaiseException(msg string) Interface {
	return vm.NewException(msg).Raise()
}

// RaiseExceptionf returns NewExceptionf(format, args...).Raise().
func (vm *VM) RaiseExceptionf(format string, args ...interface{}) Interface {
	return vm.NewExceptionf(format, args...).Raise()
}

// String returns the error message.
func (e *Exception) String() string {
	s, _ := GetSlot(e, "error")
	return fmt.Sprint(s)
}

// Error returns the error message.
func (e *Exception) Error() string {
	s, _ := GetSlot(e, "error")
	return fmt.Sprint(s)
}

// Raise returns the exception in a Stop, so that the interpreter will treat it
// as an exception, rather than as an object.
func (e *Exception) Raise() Interface {
	return Stop{Status: ExceptionStop, Result: e}
}

// IsIoError returns true if the given object is an exception produced by Io.
func IsIoError(err interface{}) bool {
	switch e := err.(type) {
	case *Exception:
		return true
	case Stop:
		return e.Status == ExceptionStop
	default:
		return false
	}
}

// IoError converts an error to a raising Io exception. If it is already an Io
// exception, it will be used unchanged. If it is already an Io exception being
// raised, the same object will be returned. If it is not an error, panic.
func (vm *VM) IoError(err interface{}) Interface {
	switch e := err.(type) {
	case *Exception:
		return e.Raise()
	case Stop:
		if e.Status != ExceptionStop {
			panic(fmt.Sprintf("iolang.IoError: not an error: %#v", err))
		}
		return e
	default:
		return vm.RaiseException(e.(error).Error())
	}
}

// Must panics if the argument is an error and otherwise returns it.
func Must(v Interface) Interface {
	if e, ok := v.(error); ok {
		panic(e)
	} else if s, _ := v.(Stop); s.Status == ExceptionStop {
		panic(s.Result)
	}
	return v
}

func (vm *VM) initException() {
	var exemplar *Exception
	slots := Slots{
		"caughtMessage":   vm.Nil,
		"error":           vm.Nil,
		"nestedException": vm.Nil,
		"originalCall":    vm.Nil,
		"pass":            vm.NewTypedCFunction(ExceptionPass, exemplar),
		"raise":           vm.NewCFunction(ExceptionRaise),
		"raiseFrom":       vm.NewCFunction(ExceptionRaiseFrom),
		"stack":           vm.NewTypedCFunction(ExceptionStack, exemplar),
		"type":            vm.NewString("Exception"),
	}
	SetSlot(vm.Core, "Exception", &Exception{Object: *vm.ObjectWith(slots)})
}

// ExceptionPass is an Exception method.
//
// pass re-raises a caught exception.
func ExceptionPass(vm *VM, target, locals Interface, msg *Message) Interface {
	return target.(*Exception).Raise()
}

// ExceptionRaise is an Exception method.
//
// raise creates an exception with the given error message and raises it.
func ExceptionRaise(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	nested, ok := CheckStop(msg.EvalArgAt(vm, locals, 1), LoopStops)
	if !ok {
		return nested
	}
	e := &Exception{Object: *vm.CoreInstance("Exception")}
	SetSlot(e, "error", s)
	SetSlot(e, "nestedException", nested)
	return e.Raise()
}

// ExceptionRaiseFrom is an Exception method.
//
// raiseFrom raises an exception from the given call site.
func ExceptionRaiseFrom(vm *VM, target, locals Interface, msg *Message) Interface {
	call, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return call
	}
	s, stop := msg.StringArgAt(vm, locals, 1)
	if stop != nil {
		return stop
	}
	nested, ok := CheckStop(msg.EvalArgAt(vm, locals, 2), LoopStops)
	if !ok {
		return nested
	}
	e := &Exception{Object: *vm.CoreInstance("Exception")}
	SetSlot(e, "error", s)
	SetSlot(e, "nestedException", nested)
	SetSlot(e, "originalCall", call)
	return e.Raise()
}

// ExceptionStack is an Exception method.
//
// stack returns the message stack of the exception. The Object try method
// transfers the control flow stack to the exception object, so the result will
// be an empty list except for caught exceptions.
func ExceptionStack(vm *VM, target, locals Interface, msg *Message) Interface {
	e := target.(*Exception)
	l := make([]Interface, len(e.Stack))
	for i, m := range e.Stack {
		l[i] = m
	}
	return vm.NewList(l...)
}
