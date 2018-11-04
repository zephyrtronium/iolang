package iolang

import "fmt"

// An Exception is an Io exception.
type Exception struct {
	Object
}

// Activate returns the exception.
func (e *Exception) Activate(vm *VM, target, locals Interface, msg *Message) Interface {
	return e
}

// Clone creates a clone of the exception.
func (e *Exception) Clone() Interface {
	return &Exception{Object: Object{Slots: Slots{}, Protos: []Interface{e}}}
}

// NewException creates a new Io Exception with the given error message.
func (vm *VM) NewException(msg string) *Exception {
	e := Exception{*vm.CoreInstance("Exception")}
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
	e.L.Lock()
	defer e.L.Unlock()
	return e.Slots["error"].(*Sequence).String()
}

// Error returns the error message.
func (e *Exception) Error() string {
	e.L.Lock()
	defer e.L.Unlock()
	return e.Slots["error"].(*Sequence).String()
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
	// TODO: Exception slots
	slots := Slots{
		"type": vm.NewString("Exception"),
	}
	SetSlot(vm.Core, "Exception", &Exception{*vm.ObjectWith(slots)})
}
