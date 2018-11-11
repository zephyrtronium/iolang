package iolang

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// FileLastAccessDate is a File method.
//
// lastAccessDate returns the date at which the file was last accessed.
func FileLastAccessDate(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys()
	s, ok := si.(*syscall.Win32FileAttributeData)
	if !ok {
		panic(fmt.Sprintf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewDate(time.Unix(0, s.LastAccessTime.Nanoseconds()))
}

// FileLastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the date at which the file was created.
func FileLastInfoChangeDate(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	si := fi.Sys()
	s, ok := si.(*syscall.Win32FileAttributeData)
	if !ok {
		panic(fmt.Sprintf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewDate(time.Unix(0, s.CreationTime.Nanoseconds()))
}
