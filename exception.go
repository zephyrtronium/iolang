package iolang

import "fmt"

// An Exception is an Io exception.
type Exception struct {
	Object
	Stack []*Message
}

// Activate returns the exception.
func (e *Exception) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return e, NoStop
}

// Clone creates a clone of the exception.
func (e *Exception) Clone() Interface {
	return &Exception{Object: Object{Slots: Slots{}, Protos: []Interface{e}}}
}

// NewException creates a new Io Exception with the given error message.
func (vm *VM) NewException(msg string) *Exception {
	e := Exception{Object: *vm.CoreInstance("Exception")}
	e.Slots["error"] = vm.NewString(msg)
	e.Slots["coroutine"] = vm
	return &e
}

// NewExceptionf creates a new Io Exception with the given formatted error
// message.
func (vm *VM) NewExceptionf(format string, args ...interface{}) *Exception {
	return vm.NewException(fmt.Sprintf(format, args...))
}

// RaiseException returns NewException(msg).Raise().
func (vm *VM) RaiseException(msg string) (Interface, Stop) {
	return vm.NewException(msg).Raise()
}

// RaiseExceptionf returns NewExceptionf(format, args...).Raise().
func (vm *VM) RaiseExceptionf(format string, args ...interface{}) (Interface, Stop) {
	return vm.NewExceptionf(format, args...).Raise()
}

// String returns the error message.
func (e *Exception) String() string {
	e.Lock()
	s := e.Slots["error"]
	e.Unlock()
	return fmt.Sprint(s)
}

// Error returns the error message.
func (e *Exception) Error() string {
	e.Lock()
	s := e.Slots["error"]
	e.Unlock()
	return fmt.Sprint(s)
}

// Raise returns the exception in a Stop, so that the interpreter will treat it
// as an exception, rather than as an object.
func (e *Exception) Raise() (Interface, Stop) {
	return e, ExceptionStop
}

// IoError converts an error to a raising Io exception. If it is already an Io
// object, it will be used unchanged. Otherwise, if it is not an error, panic.
func (vm *VM) IoError(err interface{}) (Interface, Stop) {
	switch e := err.(type) {
	case Interface:
		return e, ExceptionStop
	case error:
		return vm.RaiseException(e.Error())
	}
	panic(fmt.Sprintf("iolang.IoError: not an error: %#v", err))
}

// Must panics if the argument is an error and otherwise returns it.
func Must(v Interface) Interface {
	if _, ok := v.(error); ok {
		panic(v)
	}
	return v
}

func (vm *VM) initException() {
	var kind *Exception
	slots := Slots{
		"caughtMessage":   vm.Nil,
		"error":           vm.Nil,
		"nestedException": vm.Nil,
		"originalCall":    vm.Nil,
		"pass":            vm.NewCFunction(ExceptionPass, kind),
		"raise":           vm.NewCFunction(ExceptionRaise, nil),
		"raiseFrom":       vm.NewCFunction(ExceptionRaiseFrom, nil),
		"stack":           vm.NewCFunction(ExceptionStack, kind),
		"type":            vm.NewString("Exception"),
	}
	vm.Core.SetSlot("Exception", &Exception{Object: *vm.ObjectWith(slots)})
}

// ExceptionPass is an Exception method.
//
// pass re-raises a caught exception.
func ExceptionPass(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return target.(*Exception).Raise()
}

// ExceptionRaise is an Exception method.
//
// raise creates an exception with the given error message and raises it.
func ExceptionRaise(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	nested, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		return nested, stop
	}
	e := &Exception{Object: *vm.CoreInstance("Exception")}
	e.SetSlot("error", s)
	e.SetSlot("nestedException", nested)
	return e.Raise()
}

// ExceptionRaiseFrom is an Exception method.
//
// raiseFrom raises an exception from the given call site.
func ExceptionRaiseFrom(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	call, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return call, stop
	}
	s, err, stop := msg.StringArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	nested, stop := msg.EvalArgAt(vm, locals, 2)
	if stop != NoStop {
		return nested, stop
	}
	e := &Exception{Object: *vm.CoreInstance("Exception")}
	e.SetSlot("error", s)
	e.SetSlot("nestedException", nested)
	e.SetSlot("originalCall", call)
	return e.Raise()
}

// ExceptionStack is an Exception method.
//
// stack returns the message stack of the exception.
func ExceptionStack(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	e := target.(*Exception)
	l := make([]Interface, len(e.Stack))
	for i, m := range e.Stack {
		l[i] = m
	}
	return vm.NewList(l...), NoStop
}
