package iolang

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// A File is an object allowing interfacing with the operating system's files.
type File struct {
	Object
	File *os.File
	// Path is in the OS's convention internally, but Io-facing methods convert
	// it to slash-separated.
	Path string
	Mode string
	EOF  bool // no equivalent to feof() in Go
}

// NewFile creates a File object with the given file. The mode string should
// be one of "read", "update", or "append", depending on the flags used when
// opening the file.
func (vm *VM) NewFile(file *os.File, mode string) *File {
	f := File{
		Object: *vm.CoreInstance("File"),
		File:   file,
		Mode:   mode,
	}
	if file != nil {
		f.Path = file.Name()
	}
	return &f
}

// NewFileAt creates a File object unopened at the given path. The mode will be
// set to "read". The path should use the OS's separator convention.
func (vm *VM) NewFileAt(path string) *File {
	return &File{
		Object: *vm.CoreInstance("File"),
		Path:   path,
		Mode:   "read",
	}
}

// Activate returns the file.
func (f *File) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return f, NoStop
}

// Clone creates a clone of this file with no associated file.
func (f *File) Clone() Interface {
	return &File{
		Object: Object{Slots: Slots{}, Protos: []Interface{f}},
		Mode:   "update",
	}
}

// ReadLine reads one line from the file such that the file cursor will be
// positioned after the first encountered newline. The line without the newline
// is returned, along with any error, which may include io.EOF.
func (f *File) ReadLine() (line []byte, err error) {
	// Basically, we get to reimplement fgets because we can't predict how
	// bufio would interact with other methods, there's no other fgets
	// equivalent in the standard library, and Io uses fgets. :)
	var fi os.FileInfo
	fi, err = f.File.Stat()
	n, tn := 0, 0
	if err == nil && fi.Mode().IsRegular() {
		// If we can seek, we can read portions of the file at a time and seek
		// back once we find what we're looking for.
		b := make([]byte, 4096)
		for {
			tn, err = f.File.Read(b)
			if tn == 0 {
				break
			}
			n += tn
			k := bytes.IndexByte(b[:n], '\n')
			if k != -1 {
				j := 0
				if k > 0 && b[k-1] == '\r' {
					j = 1
				}
				line = append(line, b[:k-j]...)
				_, err = f.File.Seek(int64(k-n+1), io.SeekCurrent)
				break
			} else {
				line = append(line, b[:n]...)
			}
			if err != nil {
				break
			}
		}
	} else {
		// We (probably) can't seek, so instead we get to read one byte at a
		// time until we find a newline.
		b := []byte{0}
		err = nil // could be non-nil from Stat, but we don't care
		for {
			tn, err = f.File.Read(b)
			if tn == 0 {
				break
			}
			n += tn
			if b[0] == '\n' {
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				break
			}
			line = append(line, b...)
		}
	}
	if err == io.EOF {
		if n > 0 {
			err = nil
		} else {
			f.EOF = true
		}
	}
	return
}

func (vm *VM) initFile() {
	var kind *File
	slots := Slots{
		"asBuffer":           vm.NewCFunction(FileAsBuffer, kind),
		"at":                 vm.NewCFunction(FileAt, kind),
		"atPut":              vm.NewCFunction(FileAtPut, kind),
		"close":              vm.NewCFunction(FileClose, kind),
		"contents":           vm.NewCFunction(FileContents, kind),
		"descriptor":         vm.NewCFunction(FileDescriptor, kind),
		"exists":             vm.NewCFunction(FileExists, kind),
		"flush":              vm.NewCFunction(FileFlush, kind),
		"foreach":            vm.NewCFunction(FileForeach, kind),
		"foreachLine":        vm.NewCFunction(FileForeachLine, kind),
		"isAtEnd":            vm.NewCFunction(FileIsAtEnd, kind),
		"isDirectory":        vm.NewCFunction(FileIsDirectory, kind),
		"isLink":             vm.NewCFunction(FileIsLink, kind),
		"isOpen":             vm.NewCFunction(FileIsOpen, kind),
		"isPipe":             vm.NewCFunction(FileIsPipe, kind),
		"isRegularFile":      vm.NewCFunction(FileIsRegularFile, kind),
		"isSocket":           vm.NewCFunction(FileIsSocket, kind),
		"isUserExecutable":   vm.NewCFunction(FileIsUserExecutable, kind),
		"lastDataChangeDate": vm.NewCFunction(FileLastDataChangeDate, kind),
		"mode":               vm.NewCFunction(FileMode, kind),
		"moveTo":             vm.NewCFunction(FileMoveTo, kind),
		"name":               vm.NewCFunction(FileName, kind),
		"open":               vm.NewCFunction(FileOpen, kind),
		"openForAppending":   vm.NewCFunction(FileOpenForAppending, kind),
		"openForReading":     vm.NewCFunction(FileOpenForReading, kind),
		"openForUpdating":    vm.NewCFunction(FileOpenForUpdating, kind),
		"path":               vm.NewCFunction(FilePath, kind),
		"position":           vm.NewCFunction(FilePosition, kind),
		"positionAtEnd":      vm.NewCFunction(FilePositionAtEnd, kind),
		"protectionMode":     vm.NewCFunction(FileProtectionMode, kind),
		"readBufferOfLength": vm.NewCFunction(FileReadBufferOfLength, kind),
		"readLine":           vm.NewCFunction(FileReadLine, kind),
		"readLines":          vm.NewCFunction(FileReadLines, kind),
		"readStringOfLength": vm.NewCFunction(FileReadStringOfLength, kind),
		"readToEnd":          vm.NewCFunction(FileReadToEnd, kind),
		"rewind":             vm.NewCFunction(FileRewind, kind),
		"setPath":            vm.NewCFunction(FileSetPath, kind),
		"setPosition":        vm.NewCFunction(FileSetPosition, kind),
		"size":               vm.NewCFunction(FileSize, kind),
		"temporaryFile":      vm.NewCFunction(FileTemporaryFile, nil),
		"truncateToSize":     vm.NewCFunction(FileTruncateToSize, kind),
		"type":               vm.NewString("File"),
		"write":              vm.NewCFunction(FileWrite, kind),

		// Methods with platform-dependent implementations:
		"lastAccessDate":     vm.NewCFunction(FileLastAccessDate, kind),
		"lastInfoChangeDate": vm.NewCFunction(FileLastInfoChangeDate, kind),
	}
	slots["descriptorId"] = slots["descriptor"]
	vm.SetSlot(vm.Core, "File", &File{Object: *vm.ObjectWith(slots)})

	stdin := vm.NewFile(os.Stdin, "read")
	stdout := vm.NewFile(os.Stdout, "")
	stderr := vm.NewFile(os.Stderr, "")
	vm.SetSlot(stdout, "mode", vm.Nil)
	vm.SetSlot(stderr, "mode", vm.Nil)
	slots["standardInput"] = stdin
	slots["standardOutput"] = stdout
	slots["standardError"] = stderr
}

// FileAsBuffer is a File method.
//
// asBuffer reads the contents of the file into a Sequence.
func FileAsBuffer(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	b, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewSequence(b, true, "utf8"), NoStop
}

// FileAt is a File method.
//
// at returns as a Number the byte in the file at a given position.
func FileAt(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	b := []byte{0}
	switch _, err := f.File.ReadAt(b, int64(n.Value)); err {
	case nil:
		return vm.NewNumber(float64(b[0])), NoStop
	case io.EOF:
		return vm.Nil, NoStop
	default:
		return vm.IoError(err)
	}
}

// FileAtPut is a File method.
//
// atPut writes a single byte to the file at a given position.
func FileAtPut(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	c, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	if _, err := f.File.WriteAt([]byte{byte(c.Value)}, int64(n.Value)); err != nil {
		return vm.IoError(err)
	}
	return target, NoStop
}

// FileClose is a File method.
//
// close closes the file.
func FileClose(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	switch f.File {
	case nil, os.Stdin, os.Stdout, os.Stderr:
		// Do nothing.
	default:
		// TODO: Io does extra stuff for pipes, esp. on non-Windows.
		if err := f.File.Close(); err != nil {
			return vm.IoError(err)
		}
		f.File = nil
	}
	return target, NoStop
}

// FileContents is a File method.
//
// contents reads the contents of the file into a buffer object. Same as
// asBuffer, but can also read standardInput.
func FileContents(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	if f.File == os.Stdin {
		b, err := ioutil.ReadAll(f.File)
		if err != nil {
			return vm.IoError(err)
		}
		return vm.NewSequence(b, true, "utf8"), NoStop
	}
	b, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewSequence(b, true, "utf8"), NoStop
}

// FileDescriptor is a File method.
//
// descriptor returns the underlying file descriptor as a Number.
func FileDescriptor(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.NewNumber(float64(f.File.Fd())), NoStop
}

// FileExists is a File method.
//
// exists returns whether a file with this file's path exists.
func FileExists(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	_, err := os.Stat(f.Path)
	if err == nil {
		return vm.True, NoStop
	}
	if os.IsNotExist(err) {
		return vm.False, NoStop
	}
	return vm.IoError(err)
}

// FileFlush is a File method.
//
// flush synchronizes the file state between the program and the operating
// system.
func FileFlush(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	if err := f.File.Sync(); err != nil {
		return vm.IoError(err)
	}
	return target, NoStop
}

// FileForeach is a File method.
//
// foreach executes a message for each byte of the file.
func FileForeach(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 2 or 3 arguments")
	}
	f := target.(*File)
	if info, err := f.File.Stat(); err == nil && info.Mode().IsRegular() {
		// Regular file, so we can read into a buffer and then seek back if we
		// encounter a Stop.
		k, j, n := 0, 0, 0
		b := make([]byte, 4096)
		defer func() {
			f.File.Seek(int64(j-n), io.SeekCurrent)
		}()
		for {
			n, err = f.File.Read(b)
			j = 0
			for _, c := range b[:n] {
				v := vm.NewNumber(float64(c))
				if hvn {
					vm.SetSlot(locals, vn, v)
					if hkn {
						vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
					}
				}
				result, control = ev.Send(vm, v, locals)
				switch control {
				case NoStop, ContinueStop: // do nothing
				case BreakStop:
					return result, NoStop
				case ReturnStop, ExceptionStop:
					return result, control
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
				}
				k++
				j++
			}
			if err == io.EOF {
				f.EOF = true
				break
			}
			if err != nil {
				return vm.IoError(err)
			}
		}
	} else {
		// Other than a regular file. We can't necessarily seek around, so we
		// have to read one byte at a time.
		b := []byte{0}
		for k := 0; err != io.EOF; k++ {
			_, err = f.File.Read(b)
			if err != nil {
				if err == io.EOF {
					f.EOF = true
					break
				}
				return vm.IoError(err)
			}
			v := vm.NewNumber(float64(b[0]))
			if hvn {
				vm.SetSlot(locals, vn, v)
				if hkn {
					vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
				}
			}
			result, control = ev.Send(vm, v, locals)
			switch control {
			case NoStop, ContinueStop: // do nothing
			case BreakStop:
				return result, NoStop
			case ReturnStop, ExceptionStop:
				return result, control
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
			}
		}
	}
	return result, NoStop
}

// FileForeachLine is a File method.
//
// foreachLine executes a message for each line of the file.
func FileForeachLine(vm *VM, target, locals Interface, msg *Message) (result Interface, control Stop) {
	kn, vn, hkn, hvn, ev := ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseException("foreach requires 1, 2, or 3 arguments")
	}
	f := target.(*File)
	k := 0
	// f.ReadLine implements the same logic as FileForeach above.
	for {
		line, err := f.ReadLine()
		if line != nil {
			v := vm.NewSequence(line, true, "latin1")
			if hvn {
				vm.SetSlot(locals, vn, v)
				if hkn {
					vm.SetSlot(locals, kn, vm.NewNumber(float64(k)))
				}
			}
			result, control = ev.Send(vm, v, locals)
			switch control {
			case NoStop, ContinueStop: // do nothing
			case BreakStop:
				return result, NoStop
			case ReturnStop, ExceptionStop:
				return result, control
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
			}
			if err != nil {
				break
			}
			k++
		}
	}
	return result, NoStop
}

// FileIsAtEnd is a File method.
//
// isAtEnd returns true if the file is at EOF.
func FileIsAtEnd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.IoBool(f.EOF), NoStop
}

// FileIsDirectory is a File method.
//
// isDirectory returns true if the path of the file is a directory.
func FileIsDirectory(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.IsDir()), NoStop
}

// FileIsLink is a File method.
//
// isLink returns true if the path of the file is a symbolic link.
func FileIsLink(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Lstat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeSymlink != 0), NoStop
}

// FileIsOpen is a File method.
//
// isOpen returns true if the file is open.
func FileIsOpen(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.IoBool(f.File != nil), NoStop
}

// FileIsPipe is a File method.
//
// isPipe returns true if the path of the file is a named pipe.
func FileIsPipe(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeNamedPipe != 0), NoStop
}

// FileIsRegularFile is a File method.
//
// isRegularFile returns true if the path of the file is a regular file.
func FileIsRegularFile(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode().IsRegular()), NoStop
}

// FileIsSocket is a File method.
//
// isSocket returns true if the path of the file is a socket.
func FileIsSocket(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeSocket != 0), NoStop
}

// FileIsUserExecutable is a File method.
//
// isUserExecutable returns true if the path of the file is executable by its
// owner. Always false on Windows.
func FileIsUserExecutable(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode().Perm()&0100 != 0), NoStop
}

// FileLastDataChangeDate is a File method.
//
// lastDataChangeDate returns the date at which the file's contents were last
// modified.
func FileLastDataChangeDate(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewDate(fi.ModTime()), NoStop
}

// FileMode is a File method.
//
// mode returns a string describing the file's mode; one of "read", "update",
// or "append".
func FileMode(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.NewString(f.Mode), NoStop
}

// FileMoveTo is a File method.
//
// moveTo moves the file at the file's path to the given path.
func FileMoveTo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	to := filepath.FromSlash(s.String())
	if err := os.Rename(f.Path, to); err != nil {
		return vm.IoError(err)
	}
	return target, NoStop
}

// FileName is a File method.
//
// FileName returns the name of the file or directory at the file's path,
// similar to UNIX basename.
func FileName(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.NewString(filepath.Base(f.Path)), NoStop
}

// FileOpen is a File method.
//
// open opens the file.
func FileOpen(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	if f.File == nil {
		var err error
		switch f.Mode {
		case "read":
			f.File, err = os.Open(f.Path)
		case "update":
			f.File, err = os.OpenFile(f.Path, os.O_RDWR|os.O_CREATE, 0666)
		case "append":
			f.File, err = os.OpenFile(f.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		default:
			panic(fmt.Sprintf("invalid file mode %q", f.Mode))
		}
		if err != nil {
			return vm.IoError(err)
		}
	}
	return target, NoStop
}

// FileOpenForAppending is a File method.
//
// openForAppending opens the file for appending.
func FileOpenForAppending(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	f.Mode = "append"
	return FileOpen(vm, target, locals, msg)
}

// FileOpenForReading is a File method.
//
// openForReading opens the file for reading.
func FileOpenForReading(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	f.Mode = "read"
	return FileOpen(vm, target, locals, msg)
}

// FileOpenForUpdating is a File method.
//
// openForUpdating opens the file for updating.
func FileOpenForUpdating(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	f.Mode = "update"
	return FileOpen(vm, target, locals, msg)
}

// FilePath is a File method.
//
// path returns the file's absolute path.
func FilePath(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	return vm.NewString(filepath.ToSlash(f.Path)), NoStop
}

// FilePosition is a File method.
//
// position returns the current position of the file cursor.
func FilePosition(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	p, err := f.File.Seek(0, io.SeekCurrent)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(p)), NoStop
}

// FilePositionAtEnd is a File method.
//
// positionAtEnd moves the file cursor to the end of the file.
func FilePositionAtEnd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	_, err := f.File.Seek(0, io.SeekEnd)
	if err != nil {
		return vm.IoError(err)
	}
	f.EOF = false
	return target, NoStop
}

// FileProtectionMode is a File method.
//
// protectionMode returns the stat mode of the path of the file as a Number.
func FileProtectionMode(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(fi.Mode())), NoStop
}

// FileReadBufferOfLength is a File method.
//
// readBufferOfLength reads the specified number of bytes into a Sequence.
func FileReadBufferOfLength(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	count, aerr, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	if count.Value < 0 {
		return vm.RaiseException("can't read negative bytes")
	}
	b := make([]byte, int(count.Value))
	n, err := f.File.Read(b)
	if n > 0 {
		return vm.NewSequence(b, true, "utf8"), NoStop
	}
	if err != nil {
		if err != io.EOF {
			return vm.IoError(err)
		}
		f.EOF = true
	}
	return vm.Nil, NoStop
}

// FileReadLine is a File method.
//
// readLine reads a line from the file.
func FileReadLine(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	b, err := f.ReadLine()
	if err != nil {
		if err == io.EOF && len(b) == 0 {
			return vm.Nil, NoStop
		}
		return vm.IoError(err)
	}
	return vm.NewSequence(b, true, "utf8"), NoStop
}

// FileReadLines is a File method.
//
// readLines returns a List containing all lines in the file.
func FileReadLines(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	l := []Interface{}
	for {
		b, err := f.ReadLine()
		if err != nil && err != io.EOF {
			return vm.IoError(err)
		}
		if err == nil {
			l = append(l, vm.NewSequence(b, true, "utf8"))
		} else {
			break
		}
	}
	if len(l) > 0 {
		return vm.NewList(l...), NoStop
	}
	return vm.Nil, NoStop
}

// FileReadStringOfLength is a File method.
//
// readStringOfLength reads a string up to the given length from the file.
func FileReadStringOfLength(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	count, aerr, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	if count.Value < 0 {
		return vm.RaiseException("can't read negative bytes")
	}
	b := make([]byte, int(count.Value))
	n, err := f.File.Read(b)
	if n > 0 {
		return vm.NewSequence(b, false, "utf8"), NoStop
	}
	if err != nil {
		if err != io.EOF {
			return vm.IoError(err)
		}
		f.EOF = true
	}
	return vm.Nil, NoStop
}

// FileReadToEnd is a File method.
//
// readToEnd reads chunks of a given size (default 4096) to the end of the file
// and returns a Sequence containing the bytes read.
func FileReadToEnd(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	sz := 4096
	if len(msg.Args) > 0 {
		n, err, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		if n.Value >= 1 {
			sz = int(n.Value)
		}
	}
	b := make([]byte, sz)
	v := []byte{}
	for {
		n, err := f.File.Read(b)
		if err != nil {
			if err != io.EOF {
				return vm.IoError(err)
			}
			f.EOF = true
		}
		if n > 0 {
			v = append(v, b[:n]...)
		} else {
			break
		}
	}
	if len(v) > 0 {
		return vm.NewSequence(v, true, "utf8"), NoStop
	}
	return vm.Nil, NoStop
}

// FileRemove is a File method.
//
// remove removes the file at the file's path.
func FileRemove(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	err := os.Remove(f.Path)
	if err != nil && !os.IsNotExist(err) {
		return vm.IoError(err)
	}
	return target, NoStop
}

// FileRewind is a File method.
//
// rewind returns the file cursor to the beginning of the file.
func FileRewind(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	_, err := f.File.Seek(0, io.SeekStart)
	if err != nil {
		return vm.IoError(err)
	}
	f.EOF = false
	return target, NoStop
}

// FileSetPath is a File method.
//
// setPath sets the file's path.
func FileSetPath(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	s, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	f.Path = filepath.FromSlash(s.String())
	return target, NoStop
}

// FileSetPosition is a File method.
//
// setPosition changes the file cursor's location.
func FileSetPosition(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	n, aerr, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	_, err := f.File.Seek(int64(n.Value), io.SeekStart)
	if err != nil {
		return vm.IoError(err)
	}
	// We might have seeked to or past the end of the file, in which case it's
	// technically false to say we aren't at EOF, but the file might also grow
	// between the seek and the next read, in which case claiming EOF is
	// misleading. It's also easier to implement it this way.
	f.EOF = false
	return target, NoStop
}

// FileSize is a File method.
//
// size determines the file size.
func FileSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(fi.Size())), NoStop
}

// FileTemporaryFile is a File method.
//
// temporaryFile creates a file that did not exist previously. It is not
// guaranteed to be removed at any point.
func FileTemporaryFile(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	fp, err := ioutil.TempFile("", "iolang_temp")
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewFile(fp, "update"), NoStop
}

// FileTruncateToSize is a File method.
//
// truncateToSize truncates the file to the given size.
func FileTruncateToSize(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	n, aerr, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	err := f.File.Truncate(int64(n.Value))
	if err != nil {
		return vm.IoError(err)
	}
	return target, NoStop
}

// FileWrite is a File method.
//
// write writes its arguments to the file.
func FileWrite(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	f := target.(*File)
	for i := range msg.Args {
		s, aerr, stop := msg.StringArgAt(vm, locals, i)
		if stop != NoStop {
			return aerr, stop
		}
		_, err := f.File.Write(s.Bytes())
		if err != nil {
			return vm.IoError(err)
		}
	}
	return target, NoStop
}
