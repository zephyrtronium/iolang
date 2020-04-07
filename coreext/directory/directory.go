//go:generate go run ../../cmd/gencore directory_init.go directory ./io
//go:generate gofmt -s -w directory_init.go

package directory

import (
	"os"
	"path/filepath"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/file"
	"github.com/zephyrtronium/iolang/internal"
)

// DirectoryTag is the Tag for Directory objects.
const DirectoryTag = iolang.BasicTag("Directory")

// New creates a new Directory with the given path.
func New(vm *iolang.VM, path string) *iolang.Object {
	return vm.ObjectWith(nil, vm.CoreProto("Directory"), path, DirectoryTag)
}

// ArgAt evaluates the nth argument and returns it as a string. If a
// stop occurs during evaluation, the path will be empty, and the stop status
// and result will be returned. If the evaluated result is not a Directory, the
// result will be empty, and an exception will be returned with an
// ExceptionStop.
func ArgAt(vm *iolang.VM, m *iolang.Message, locals *iolang.Object, n int) (string, *iolang.Object, iolang.Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == iolang.NoStop {
		v.Lock()
		d, ok := v.Value.(string)
		v.Unlock()
		if ok {
			return d, nil, iolang.NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Directory, not %s", n, m.Text, vm.TypeName(v))
		s = iolang.ExceptionStop
	}
	return "", v, s
}

func init() {
	internal.Register(initDirectory)
}

func initDirectory(vm *iolang.VM) {
	slots := iolang.Slots{
		"at":                         vm.NewCFunction(at, DirectoryTag),
		"create":                     vm.NewCFunction(create, DirectoryTag),
		"createSubdirectory":         vm.NewCFunction(createSubdirectory, DirectoryTag),
		"currentWorkingDirectory":    vm.NewCFunction(currentWorkingDirectory, nil),
		"exists":                     vm.NewCFunction(exists, DirectoryTag),
		"items":                      vm.NewCFunction(items, DirectoryTag),
		"name":                       vm.NewCFunction(name, DirectoryTag),
		"path":                       vm.NewCFunction(path, DirectoryTag),
		"setCurrentWorkingDirectory": vm.NewCFunction(setCurrentWorkingDirectory, nil),
		"setPath":                    vm.NewCFunction(setPath, nil),
		"type":                       vm.NewString("Directory"),
	}
	internal.CoreInstall(vm, "Directory", slots, "", DirectoryTag)
	internal.Ioz(vm, coreIo, coreFiles)
}

// DirectoryAt is a Directory method.
//
// at returns a File or Directory object at the given path (always) relative to
// the directory, or nil if there is no such file.
func at(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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
		return file.NewAt(vm, p)
	}
	return New(vm, p)
}

// DirectoryCreate is a Directory method.
//
// create creates the directory if it does not exist. Returns nil on failure.
func create(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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
func createSubdirectory(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	nm, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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
			return New(vm, p)
		}
		return vm.IoError(err)
	}
	if fi.IsDir() {
		return New(vm, p)
	}
	return vm.RaiseExceptionf("%s already exists", p)
}

// DirectoryCurrentWorkingDirectory is a Directory method.
//
// currentWorkingDirectory returns the path of the current working directory
// with the operating system's path style.
func currentWorkingDirectory(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	d, err := os.Getwd()
	if err != nil {
		return vm.NewString(".")
	}
	return vm.NewString(d)
}

// DirectoryExists is a Directory method.
//
// exists returns true if the directory exists and is a directory.
func exists(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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
func items(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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
	l := make([]*iolang.Object, len(fis))
	for i, fi := range fis {
		p := filepath.Join(d, fi.Name())
		if fi.IsDir() {
			l[i] = New(vm, p)
		} else {
			l[i] = file.NewAt(vm, p)
		}
	}
	return vm.NewList(l...)
}

// DirectoryName is a Directory method.
//
// name returns the name of the file or directory at the directory's path,
// similar to Unix basename.
func name(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	return vm.NewString(filepath.Base(d))
}

// DirectoryPath is a Directory method.
//
// path returns the directory's path.
func path(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(string)
	target.Unlock()
	return vm.NewString(filepath.ToSlash(d))
}

// DirectorySetCurrentWorkingDirectory is a Directory method.
//
// setCurrentWorkingDirectory sets the program's current working directory.
func setCurrentWorkingDirectory(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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
func setPath(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = filepath.FromSlash(s)
	target.Unlock()
	return target
}
