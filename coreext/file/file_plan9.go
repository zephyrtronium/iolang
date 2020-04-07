// +build plan9

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
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Dir)
	// 9P has string uid/gid, not numeric.
	return vm.NewString(si.Gid)
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
	si := fi.Sys().(*syscall.Dir)
	return date.New(vm, time.Unix(int64(si.Atime), 0))
}

// lastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the modification time of the file.
func lastInfoChangeDate(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	return FileLastDataChangeDate(vm, target, locals, msg)
}

// userID is a File method.
//
// userId returns the user ID owning the file.
func userID(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Dir)
	// 9P has string uid/gid, not numeric.
	return vm.NewString(si.Uid)
}
