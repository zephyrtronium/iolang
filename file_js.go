// +build js,wasm

package iolang

import (
	"os"
	"syscall"
	"time"
)

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
