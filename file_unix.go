// +build !windows
// +build !plan9
// +build !wasm
// +build !darwin

package iolang

import (
	"fmt"
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Errorf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewNumber(float64(s.Gid))
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Sprintf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewDate(time.Unix(s.Atim.Unix()))
}

// FileLastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the date at which the file's metadata was last
// changed.
func FileLastInfoChangeDate(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Sprintf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewDate(time.Unix(s.Ctim.Unix()))
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Errorf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewNumber(float64(s.Uid))
}
