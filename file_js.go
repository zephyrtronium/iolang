// +build js,wasm

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
	// No such thing on Wasm, or at least not readily available.
	return vm.NewNumber(-1)
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
	si := fi.Sys().(*syscall.Stat_t)
	return vm.NewDate(time.Unix(si.Atime, si.AtimeNsec))
}

// FileLastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the modification time of the file.
func FileLastInfoChangeDate(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys().(*syscall.Stat_t)
	return vm.NewDate(time.Unix(si.Ctime, si.CtimeNsec))
}

// FileUserID is a File method.
//
// userId returns the user ID owning the file.
func FileUserID(vm *VM, target, locals *Object, msg *Message) *Object {
	// No such thing on Wasm, or at least not readily available.
	return vm.NewNumber(-1)
}
