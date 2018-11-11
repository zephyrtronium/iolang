// +build !windows
// +build !plan9

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
func FileLastInfoChangeDate(vm *VM, target, locals Interface, msg *Message) Interface {
	f := target.(*File)
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
