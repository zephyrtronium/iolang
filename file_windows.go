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
	// See FileUserID below.
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
func FileLastInfoChangeDate(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
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

// FileUserID is a File method.
//
// userId returns the user ID owning the file.
func FileUserID(vm *VM, target, locals *Object, msg *Message) *Object {
	// We could encode the file path in UTF-16, use x/sys/windows.CreateFile
	// to get a handle, use GetSecurityInfo to get a SECURITY_DESCRIPTOR, use
	// LookupAccountSid to write the account name to a *uint16 backed by a
	// slice of ??? size, and finally decode from UTF-16 to get a string user
	// ID, all during runtime.LockOSThread... or we can say "Windows doesn't
	// have this," which isn't far from the truth anyway, since "user ID"
	// implies numeric to many programmers. (On Plan 9, we return a string, but
	// anyone using Plan 9 should expect things to be different.)
	return vm.NewNumber(-1)
}
