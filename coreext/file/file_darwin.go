// +build darwin

package file

import (
	"fmt"
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Errorf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewNumber(float64(s.Gid))
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Sprintf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return date.New(vm, time.Unix(s.Atimespec.Sec, s.Atimespec.Nsec))
}

// lastInfoChangeDate is a File method.
//
// lastInfoChangeDate returns the date at which the file's metadata was last
// changed.
func lastInfoChangeDate(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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
	return date.New(vm, time.Unix(s.Ctimespec.Sec, s.Ctimespec.Nsec))
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
	si := fi.Sys()
	s, ok := si.(*syscall.Stat_t)
	if !ok {
		panic(fmt.Errorf("iolang: %T.Sys() returned wrong type %T", fi, si))
	}
	return vm.NewNumber(float64(s.Uid))
}
