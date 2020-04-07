//go:generate go run ../../cmd/gencore path_init.go path ./io
//go:generate gofmt -s -w path_init.go

package path

import (
	"path/filepath"
	"runtime"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"
)

func init() {
	internal.Register(initPath)
}

func initPath(vm *iolang.VM) {
	slots := iolang.Slots{
		"absolute":        vm.NewCFunction(absolute, nil),
		"hasDriveLetters": vm.IoBool(runtime.GOOS == "windows"),
		"isPathAbsolute":  vm.NewCFunction(isPathAbsolute, nil),
		"listSeparator":   vm.NewString(string(filepath.ListSeparator)),
		"separator":       vm.NewString(string(filepath.Separator)),
	}
	internal.CoreInstall(vm, "Path", slots, nil, nil)
	internal.Ioz(vm, coreIo, coreFiles)
}

// absolute is a Path method.
//
// absolute returns an absolute version of the argument path.
func absolute(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	abs, err := filepath.Abs(filepath.FromSlash(s))
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewString(filepath.ToSlash(abs))
}

// isPathAbsolute is a Path method.
//
// isPathAbsolute returns whether the argument is an absolute path. The path
// may be operating system- or Io-style.
func isPathAbsolute(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	return vm.IoBool(filepath.IsAbs(filepath.FromSlash(s)))
}
