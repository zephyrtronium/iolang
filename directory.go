package iolang

import (
	"os"
	"path/filepath"
)

// A Directory is an object allowing interfacing with the operating system's
// directories.
type Directory struct {
	Object
	Path string
}

// NewDirectory creates a new Directory with the given path.
func (vm *VM) NewDirectory(path string) *Directory {
	return &Directory{
		Object: *vm.CoreInstance("Directory"),
		Path:   path,
	}
}

// Activate returns the directory.
func (d *Directory) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return d
}

// Clone creates a clone of the directory.
func (d *Directory) Clone() Interface {
	return &Directory{
		Object: Object{Slots: Slots{}, Protos: []Interface{d}},
		Path:   d.Path,
	}
}

func (vm *VM) initDirectory() {
	var exemplar *Directory
	slots := Slots{
		"at":                         vm.NewTypedCFunction(DirectoryAt, exemplar),
		"create":                     vm.NewTypedCFunction(DirectoryCreate, exemplar),
		"createSubdirectory":         vm.NewTypedCFunction(DirectoryCreateSubdirectory, exemplar),
		"currentWorkingDirectory":    vm.NewCFunction(DirectoryCurrentWorkingDirectory),
		"exists":                     vm.NewTypedCFunction(DirectoryExists, exemplar),
		"items":                      vm.NewTypedCFunction(DirectoryItems, exemplar),
		"name":                       vm.NewTypedCFunction(DirectoryName, exemplar),
		"path":                       vm.NewTypedCFunction(DirectoryPath, exemplar),
		"setCurrentWorkingDirectory": vm.NewCFunction(DirectorySetCurrentWorkingDirectory),
		"setPath":                    vm.NewCFunction(DirectorySetPath),
		"type":                       vm.NewString("Directory"),
	}
	SetSlot(vm.Core, "Directory", &Directory{Object: *vm.ObjectWith(slots)})
}

// DirectoryAt is a Directory method.
//
// at returns a File or Directory object at the given path (always) relative to
// the directory, or nil if there is no such file.
func DirectoryAt(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	p := filepath.Join(d.Path, filepath.FromSlash(s.String()))
	fi, err := os.Stat(p)
	if os.IsNotExist(err) {
		return vm.Nil
	}
	if err != nil {
		return vm.IoError(err)
	}
	if !fi.IsDir() {
		return vm.NewFileAt(p)
	}
	return vm.NewDirectory(p)
}

// DirectoryCreate is a Directory method.
//
// create creates the directory if it does not exist. Returns nil on failure.
func DirectoryCreate(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	_, err := os.Stat(d.Path)
	if err != nil && !os.IsNotExist(err) {
		return vm.IoError(err)
	}
	err = os.Mkdir(d.Path, 0755)
	if err != nil {
		// This means we return nil if the path exists and is not a directory,
		// which seems wrong, but oh well.
		return vm.Nil
	}
	return target
}

// DirectoryCreateSubdirectory is a Directory method.
//
// createSubdirectory creates a subdirectory with the given name and returns a
// Directory object for it.
func DirectoryCreateSubdirectory(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	nm, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	p := filepath.Join(d.Path, filepath.FromSlash(nm.String()))
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(p, 0755); err != nil {
				return vm.IoError(err)
			}
			return vm.NewDirectory(p)
		}
		return vm.IoError(err)
	}
	if fi.IsDir() {
		return vm.NewDirectory(p)
	}
	return vm.RaiseExceptionf("%s already exists", p)
}

// DirectoryCurrentWorkingDirectory is a Directory method.
//
// currentWorkingDirectory returns the path of the current working directory
// with the operating system's path style.
func DirectoryCurrentWorkingDirectory(vm *VM, target, locals Interface, msg *Message) Interface {
	d, err := os.Getwd()
	if err != nil {
		return vm.NewString(".")
	}
	return vm.NewString(d)
}

// DirectoryExists is a Directory method.
//
// exists returns true if the directory exists and is a directory.
func DirectoryExists(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	fi, err := os.Stat(d.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return vm.False
		}
		return vm.IoError(err)
	}
	return vm.IoBool(fi.IsDir())
}

// DirectoryItems is a Directory method.
//
// items returns a list of the files and directories within this directory.
func DirectoryItems(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	f, err := os.Open(d.Path)
	if err != nil {
		return vm.IoError(err)
	}
	fis, err := f.Readdir(0)
	f.Close()
	if err != nil {
		return vm.IoError(err)
	}
	l := make([]Interface, len(fis))
	for i, fi := range fis {
		p := filepath.Join(d.Path, fi.Name())
		if fi.IsDir() {
			l[i] = vm.NewDirectory(p)
		} else {
			l[i] = vm.NewFileAt(p)
		}
	}
	return vm.NewList(l...)
}

// DirectoryName is a Directory method.
//
// name returns the name of the file or directory at the directory's path,
// similar to Unix basename.
func DirectoryName(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	return vm.NewString(filepath.Base(d.Path))
}

// DirectoryPath is a Directory method.
//
// path returns the directory's path.
func DirectoryPath(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	return vm.NewString(filepath.ToSlash(d.Path))
}

// DirectorySetCurrentWorkingDirectory is a Directory method.
//
// setCurrentWorkingDirectory sets the program's current working directory.
func DirectorySetCurrentWorkingDirectory(vm *VM, target, locals Interface, msg *Message) Interface {
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	if err := os.Chdir(s.String()); err != nil {
		return vm.False
	}
	return vm.True
}

// DirectorySetPath is a Directory method.
//
// setPath sets the path of the Directory object.
func DirectorySetPath(vm *VM, target, locals Interface, msg *Message) Interface {
	d := target.(*Directory)
	s, stop := msg.StringArgAt(vm, locals, 0)
	if stop != nil {
		return stop
	}
	d.Path = filepath.FromSlash(s.String())
	return target
}
