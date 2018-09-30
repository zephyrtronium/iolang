package iolang

import "fmt"

// An Exception is an Io exception.
type Exception struct {
	Object
}

// NewException creates a new Io Exception with the given error message.
func (vm *VM) NewException(msg string) *Exception {
	e := Exception{Object{Slots: vm.DefaultSlots["Exception"], Protos: []Interface{vm.BaseObject}}}
	e.Slots["error"] = vm.NewString(msg)
	return &e
}

// NewExceptionf creates a new Io Exception with the given formatted error
// message.
func (vm *VM) NewExceptionf(format string, args ...interface{}) *Exception {
	return vm.NewException(fmt.Sprintf(format, args...))
}

// String returns the error message.
func (e *Exception) String() string {
	return e.Slots["error"].(*String).Value
}

// Error returns the error message.
func (e *Exception) Error() string {
	return e.Slots["error"].(*String).Value
}

// Error is an Io error. I'm not sure why I implemented this separately,
// because the original Io doesn't.
type Error struct {
	Object
}

func (vm *VM) NewError(msg string) *Error {
	e := Error{Object{Slots: vm.DefaultSlots["Error"], Protos: []Interface{vm.BaseObject}}}
	e.Slots["error"] = vm.NewString(msg)
	return &e
}

func (vm *VM) NewErrorf(format string, args ...interface{}) *Error {
	return vm.NewError(fmt.Sprintf(format, args...))
}

func (e *Error) String() string {
	return e.Slots["error"].(*String).Value
}

func (e *Error) Error() string {
	return e.Slots["error"].(*String).Value
}

// Determine whether an error is an Exception or Error from Io.
func IsIoError(e interface{}) bool {
	switch e.(type) {
	case Exception:
		return true
	case Error:
		return true
	default:
		return false
	}
}

// IoError converts an error to an Io Exception. If it is already an Io Error
// or Exception, it will be returned unchanged.
func (vm *VM) IoError(e error) Interface {
	if IsIoError(e) {
		return e.(Interface)
	}
	return vm.NewException(e.Error())
}

// Panic if the argument is an error; otherwise, return it.
func Must(v Interface) Interface {
	if e, ok := v.(error); ok {
		panic(e)
	}
	return v
}
