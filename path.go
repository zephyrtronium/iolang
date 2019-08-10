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
	vm.SetSlot(vm.Core, "Path", vm.ObjectWith(slots))
}

// PathAbsolute is a Path method.
//
// absolute returns an absolute version of the argument path.
func PathAbsolute(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	abs, err := filepath.Abs(filepath.FromSlash(s.String()))
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewString(filepath.ToSlash(abs)), NoStop
}

// PathIsPathAbsolute is a Path method.
//
// isPathAbsolute returns whether the argument is an absolute path. The path
// may be operating system- or Io-style.
func PathIsPathAbsolute(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	return vm.IoBool(filepath.IsAbs(filepath.FromSlash(s.String()))), NoStop
}
