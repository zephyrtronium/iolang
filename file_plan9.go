// +build plan9

package iolang

import (
	"os"
	"syscall"
	"time"
)

// FileGroupID is a File method.
//
// groupId returns the group ID owning the file.
func FileGroupID(vm *VM, target, locals *Object, msg *Message) *Object {
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

// FileLastAccessDate is a File method.
//
// lastAccessDate returns the date at which the file was last accessed.
func FileLastAccessDate(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Dir)
	return vm.NewDate(time.Unix(int64(si.Atime), 0))
}

// FileLastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the modification time of the file.
func FileLastInfoChangeDate(vm *VM, target, locals *Object, msg *Message) *Object {
	return FileLastDataChangeDate(vm, target, locals, msg)
}

// FileUserID is a File method.
//
// userId returns the user ID owning the file.
func FileUserID(vm *VM, target, locals *Object, msg *Message) *Object {
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
