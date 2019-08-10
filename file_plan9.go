package iolang

import (
	"os"
	"syscall"
	"time"
)

// FileLastAccessDate is a File method.
//
// lastAccessDate returns the date at which the file was last accessed.
func FileLastAccessDate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Dir)
	return vm.NewDate(time.Unix(int64(si.Atime), 0)), NoStop
}

// FileLastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the modification time of the file.
func FileLastInfoChangeDate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return FileLastDataChangeDate(vm, target, locals, msg)
}
