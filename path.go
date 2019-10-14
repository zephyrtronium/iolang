package iolang

import (
	"path/filepath"
	"runtime"
)

func (vm *VM) initPath() {
	slots := Slots{
		"absolute":        vm.NewCFunction(PathAbsolute, nil),
		"hasDriveLetters": vm.IoBool(runtime.GOOS == "windows"),
		"isPathAbsolute":  vm.NewCFunction(PathIsPathAbsolute, nil),
		"listSeparator":   vm.NewString(string(filepath.ListSeparator)),
		"separator":       vm.NewString(string(filepath.Separator)),
	}
	vm.Core.SetSlot("Path", vm.ObjectWith(slots))
}

// PathAbsolute is a Path method.
//
// absolute returns an absolute version of the argument path.
func PathAbsolute(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	abs, err := filepath.Abs(filepath.FromSlash(s))
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewString(filepath.ToSlash(abs))
}

// PathIsPathAbsolute is a Path method.
//
// isPathAbsolute returns whether the argument is an absolute path. The path
// may be operating system- or Io-style.
func PathIsPathAbsolute(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.IoBool(filepath.IsAbs(filepath.FromSlash(s)))
}
