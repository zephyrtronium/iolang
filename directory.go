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
func (d *Directory) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return d, NoStop
}

// Clone creates a clone of the directory.
func (d *Directory) Clone() Interface {
	return &Directory{
		Object: Object{Slots: Slots{}, Protos: []Interface{d}},
		Path:   d.Path,
	}
}

func (vm *VM) initDirectory() {
	var kind *Directory
	slots := Slots{
		"at":                         vm.NewCFunction(DirectoryAt, kind),
		"create":                     vm.NewCFunction(DirectoryCreate, kind),
		"createSubdirectory":         vm.NewCFunction(DirectoryCreateSubdirectory, kind),
		"currentWorkingDirectory":    vm.NewCFunction(DirectoryCurrentWorkingDirectory, nil),
		"exists":                     vm.NewCFunction(DirectoryExists, kind),
		"items":                      vm.NewCFunction(DirectoryItems, kind),
		"name":                       vm.NewCFunction(DirectoryName, kind),
		"path":                       vm.NewCFunction(DirectoryPath, kind),
		"setCurrentWorkingDirectory": vm.NewCFunction(DirectorySetCurrentWorkingDirectory, nil),
		"setPath":                    vm.NewCFunction(DirectorySetPath, nil),
		"type":                       vm.NewString("Directory"),
	}
	vm.Core.SetSlot("Directory", &Directory{Object: *vm.ObjectWith(slots)})
}

// DirectoryAt is a Directory method.
//
// at returns a File or Directory object at the given path (always) relative to
// the directory, or nil if there is no such file.
func DirectoryAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	p := filepath.Join(d.Path, filepath.FromSlash(s.String()))
	fi, err := os.Stat(p)
	if os.IsNotExist(err) {
		return vm.Nil, NoStop
	}
	if err != nil {
		return vm.IoError(err)
	}
	if !fi.IsDir() {
		return vm.NewFileAt(p), NoStop
	}
	return vm.NewDirectory(p), NoStop
}

// DirectoryCreate is a Directory method.
//
// create creates the directory if it does not exist. Returns nil on failure.
func DirectoryCreate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	_, err := os.Stat(d.Path)
	if err != nil && !os.IsNotExist(err) {
		return vm.IoError(err)
	}
	err = os.Mkdir(d.Path, 0755)
	if err != nil {
		// This means we return nil if the path exists and is not a directory,
		// which seems wrong, but oh well.
		return vm.Nil, NoStop
	}
	return target, NoStop
}

// DirectoryCreateSubdirectory is a Directory method.
//
// createSubdirectory creates a subdirectory with the given name and returns a
// Directory object for it.
func DirectoryCreateSubdirectory(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	nm, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	p := filepath.Join(d.Path, filepath.FromSlash(nm.String()))
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(p, 0755); err != nil {
				return vm.IoError(err)
			}
			return vm.NewDirectory(p), NoStop
		}
		return vm.IoError(err)
	}
	if fi.IsDir() {
		return vm.NewDirectory(p), NoStop
	}
	return vm.RaiseExceptionf("%s already exists", p)
}

// DirectoryCurrentWorkingDirectory is a Directory method.
//
// currentWorkingDirectory returns the path of the current working directory
// with the operating system's path style.
func DirectoryCurrentWorkingDirectory(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d, err := os.Getwd()
	if err != nil {
		return vm.NewString("."), NoStop
	}
	return vm.NewString(d), NoStop
}

// DirectoryExists is a Directory method.
//
// exists returns true if the directory exists and is a directory.
func DirectoryExists(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	fi, err := os.Stat(d.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return vm.False, NoStop
		}
		return vm.IoError(err)
	}
	return vm.IoBool(fi.IsDir()), NoStop
}

// DirectoryItems is a Directory method.
//
// items returns a list of the files and directories within this directory.
func DirectoryItems(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
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
	return vm.NewList(l...), NoStop
}

// DirectoryName is a Directory method.
//
// name returns the name of the file or directory at the directory's path,
// similar to Unix basename.
func DirectoryName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	return vm.NewString(filepath.Base(d.Path)), NoStop
}

// DirectoryPath is a Directory method.
//
// path returns the directory's path.
func DirectoryPath(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	return vm.NewString(filepath.ToSlash(d.Path)), NoStop
}

// DirectorySetCurrentWorkingDirectory is a Directory method.
//
// setCurrentWorkingDirectory sets the program's current working directory.
func DirectorySetCurrentWorkingDirectory(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	if err := os.Chdir(s.String()); err != nil {
		return vm.False, NoStop
	}
	return vm.True, NoStop
}

// DirectorySetPath is a Directory method.
//
// setPath sets the path of the Directory object.
func DirectorySetPath(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Directory)
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	d.Path = filepath.FromSlash(s.String())
	return target, NoStop
}
