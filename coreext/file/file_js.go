// +build js,wasm

package file

import (
	"os"
	"syscall"
	"time"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/date"
)

// groupID is a File method.
//
// groupId returns the group ID owning the file.
func groupID(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	// No such thing on Wasm, or at least not readily available.
	return vm.NewNumber(-1)
}

// lastAccessDate is a File method.
//
// lastAccessDate returns the date at which the file was last accessed.
func lastAccessDate(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Stat_t)
	return date.New(vm, time.Unix(si.Atime, si.AtimeNsec))
}

// lastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the modification time of the file.
func lastInfoChangeDate(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Stat_t)
	return date.New(vm, time.Unix(si.Ctime, si.CtimeNsec))
}

// userID is a File method.
//
// userId returns the user ID owning the file.
func userID(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	// No such thing on Wasm, or at least not readily available.
	return vm.NewNumber(-1)
}
