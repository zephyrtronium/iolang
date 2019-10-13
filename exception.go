package iolang

import "fmt"

// An Exception is an Io exception.
type Exception struct {
	Err   error
	Stack []*Message
}

// String returns the error message.
func (e Exception) String() string {
	return e.Err.Error()
}

// Error returns the error message.
func (e Exception) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error.
func (e Exception) Unwrap() error {
	return e.Err
}

// tagException is the Tag type for Exception objects.
type tagException struct{}

func (tagException) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	return self
}

func (tagException) CloneValue(value interface{}) interface{} {
	v := value.(Exception)
	if v.Stack == nil {
		return Exception{Err: v.Err}
	}
	s := make([]*Message, len(v.Stack))
	copy(s, v.Stack)
	return Exception{Err: v.Err, Stack: s}
}

func (tagException) String() string {
	return "Exception"
}

// ExceptionTag is the Tag for Exception objects. Activate returns the
// exception. CloneValue creates an exception with the same error and a copy of
// the parent exception's stack.
var ExceptionTag tagException

// NewException creates a new Exception object with the given error.
func (vm *VM) NewException(err error) *Object {
	return &Object{
		Slots: Slots{
			"coroutine": vm.Coro,
		},
		Protos: vm.CoreProto("Exception"),
		Value:  Exception{Err: err},
		Tag:    ExceptionTag,
	}
}

// NewExceptionf creates a new Io Exception with the given formatted error
// message.
func (vm *VM) NewExceptionf(format string, args ...interface{}) *Object {
	return vm.NewException(fmt.Errorf(format, args...))
}

// RaiseException returns vm.Raise(vm.NewException(msg)).
func (vm *VM) RaiseException(msg error) *Object {
	return vm.Raise(vm.NewException(msg))
}

// RaiseExceptionf returns vm.Raise(vm.NewExceptionf(format, args...)).
func (vm *VM) RaiseExceptionf(format string, args ...interface{}) *Object {
	return vm.Raise(vm.NewExceptionf(format, args...))
}

// Raise raises an exception and returns the object.
func (vm *VM) Raise(exc *Object) *Object {
	return vm.Stop(exc, ExceptionStop)
}

// IoError converts an error to an Io exception and raises it. If it is already
// an Io object, it will be used unchanged. Otherwise, if it is not an error,
// panic.
func (vm *VM) IoError(err interface{}) *Object {
	switch e := err.(type) {
	case *Object:
		return vm.Raise(e)
	case error:
		return vm.RaiseException(e)
	}
	panic(fmt.Sprintf("iolang.IoError: not an error: %#v", err))
}

func (vm *VM) initException() {
	slots := Slots{
		"caughtMessage":   vm.Nil,
		"error":           vm.NewCFunction(ExceptionError, ExceptionTag),
		"nestedException": vm.Nil,
		"originalCall":    vm.Nil,
		"pass":            vm.NewCFunction(ExceptionPass, ExceptionTag),
		"raise":           vm.NewCFunction(ExceptionRaise, nil),
		"raiseFrom":       vm.NewCFunction(ExceptionRaiseFrom, nil),
		"stack":           vm.NewCFunction(ExceptionStack, ExceptionTag),
		"type":            vm.NewString("Exception"),
	}
	vm.Core.SetSlot("Exception", &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
		Value:  Exception{Err: fmt.Errorf("no error")},
		Tag:    ExceptionTag,
	})
}

// ExceptionError is an Exception method.
//
// error returns the exception's error message.
func ExceptionError(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	s := target.Value.(Exception).Err.Error()
	target.Unlock()
	return vm.NewString(s)
}

// ExceptionPass is an Exception method.
//
// pass re-raises a caught exception.
func ExceptionPass(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.Raise(target)
}

// ExceptionRaise is an Exception method.
//
// raise creates an exception with the given error message and raises it.
func ExceptionRaise(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	nested, stop := msg.EvalArgAt(vm, locals, 1)
	if stop != NoStop {
		vm.Stop(nested, stop)
	}
	e := vm.NewExceptionf("%v", s)
	e.SetSlot("nestedException", nested)
	return vm.Raise(e)
}

// ExceptionRaiseFrom is an Exception method.
//
// raiseFrom raises an exception from the given call site.
func ExceptionRaiseFrom(vm *VM, target, locals *Object, msg *Message) *Object {
	call, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(call, stop)
	}
	s, exc, stop := msg.StringArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	nested, stop := msg.EvalArgAt(vm, locals, 2)
	if stop != NoStop {
		return vm.Stop(nested, stop)
	}
	e := vm.NewExceptionf("%v", s)
	e.SetSlots(Slots{"nestedException": nested, "originalCall": call})
	return vm.Raise(e)
}

// ExceptionSetError is an Exception method.
//
// setError sets the exception's error message.
func ExceptionSetError(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	e := target.Value.(Exception)
	e.Err = fmt.Errorf("%s", s)
	target.Value = e
	target.Unlock()
	return target
}

// ExceptionStack is an Exception method.
//
// stack returns the message stack of the exception.
func ExceptionStack(vm *VM, target, locals *Object, msg *Message) *Object {
	e := target.Value.(*Exception)
	target.Lock()
	l := make([]*Object, len(e.Stack))
	for i, m := range e.Stack {
		l[i] = vm.MessageObject(m)
	}
	target.Unlock()
	return vm.NewList(l...)
}
