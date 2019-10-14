package iolang

import (
	"os"
	"path/filepath"
)

// DirectoryTag is the Tag for Directory objects.
const DirectoryTag = BasicTag("Directory")

// NewDirectory creates a new Directory with the given path.
func (vm *VM) NewDirectory(path string) *Object {
	return &Object{
		Protos: vm.CoreProto("Directory"),
		Value:  path,
		Tag:    DirectoryTag,
	}
}

// DirectoryArgAt evaluates the nth argument and returns it as a string. If a
// stop occurs during evaluation, the path will be empty, and the stop status
// and result will be returned. If the evaluated result is not a Directory, the
// result will be empty, and an exception will be returned with an
// ExceptionStop.
func (m *Message) DirectoryArgAt(vm *VM, locals *Object, n int) (string, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		d, ok := v.Value.(string)
		v.Unlock()
		if ok {
			return d, nil, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Directory, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return "", v, s
}

func (vm *VM) initDirectory() {
	slots := Slots{
		"at":                         vm.NewCFunction(DirectoryAt, DirectoryTag),
		"create":                     vm.NewCFunction(DirectoryCreate, DirectoryTag),
		"createSubdirectory":         vm.NewCFunction(DirectoryCreateSubdirectory, DirectoryTag),
		"currentWorkingDirectory":    vm.NewCFunction(DirectoryCurrentWorkingDirectory, nil),
		"exists":                     vm.NewCFunction(DirectoryExists, DirectoryTag),
		"items":                      vm.NewCFunction(DirectoryItems, DirectoryTag),
		"name":                       vm.NewCFunction(DirectoryName, DirectoryTag),
		"path":                       vm.NewCFunction(DirectoryPath, DirectoryTag),
		"setCurrentWorkingDirectory": vm.NewCFunction(DirectorySetCurrentWorkingDirectory, nil),
		"setPath":                    vm.NewCFunction(DirectorySetPath, nil),
		"type":                       vm.NewString("Directory"),
	}
	vm.Core.SetSlot("Directory", &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
		Value:  "",
		Tag:    DirectoryTag,
	})
}

// DirectoryAt is a Directory method.
//
// at returns a File or Directory object at the given path (always) relative to
// the directory, or nil if there is no such file.
func DirectoryAt(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	p := filepath.Join(d, filepath.FromSlash(s))
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
func DirectoryCreate(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	_, err := os.Stat(d)
	if err != nil && !os.IsNotExist(err) {
		return vm.IoError(err)
	}
	err = os.Mkdir(d, 0755)
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
func DirectoryCreateSubdirectory(vm *VM, target, locals *Object, msg *Message) *Object {
	nm, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	p := filepath.Join(d, filepath.FromSlash(nm))
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
func DirectoryCurrentWorkingDirectory(vm *VM, target, locals *Object, msg *Message) *Object {
	d, err := os.Getwd()
	if err != nil {
		return vm.NewString(".")
	}
	return vm.NewString(d)
}

// DirectoryExists is a Directory method.
//
// exists returns true if the directory exists and is a directory.
func DirectoryExists(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	fi, err := os.Stat(d)
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
func DirectoryItems(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	f, err := os.Open(d)
	if err != nil {
		return vm.IoError(err)
	}
	fis, err := f.Readdir(0)
	f.Close()
	if err != nil {
		return vm.IoError(err)
	}
	l := make([]*Object, len(fis))
	for i, fi := range fis {
		p := filepath.Join(d, fi.Name())
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
func DirectoryName(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	return vm.NewString(filepath.Base(d))
}

// DirectoryPath is a Directory method.
//
// path returns the directory's path.
func DirectoryPath(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	return vm.NewString(filepath.ToSlash(d))
}

// DirectorySetCurrentWorkingDirectory is a Directory method.
//
// setCurrentWorkingDirectory sets the program's current working directory.
func DirectorySetCurrentWorkingDirectory(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	if err := os.Chdir(s); err != nil {
		return vm.False
	}
	return vm.True
}

// DirectorySetPath is a Directory method.
//
// setPath sets the path of the Directory object.
func DirectorySetPath(vm *VM, target, locals *Object, msg *Message) *Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = filepath.FromSlash(s)
	target.Unlock()
	return target
}
