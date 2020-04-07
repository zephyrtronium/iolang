//go:generate go run ../../cmd/gencore file_init.go file ./io
//go:generate gofmt -s -w file_init.go

package file

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/date"
	"github.com/zephyrtronium/iolang/internal"
)

// A File is an object allowing interfacing with the operating system's files.
type File struct {
	File *os.File
	// Path is in the OS's convention internally, but Io-facing methods convert
	// it to slash-separated.
	Path string
	Mode string
	EOF  bool // no equivalent to feof() in Go
}

// tagFile is the Tag type for File objects.
type tagFile struct{}

func (tagFile) Activate(vm *iolang.VM, self, target, locals, context *iolang.Object, msg *iolang.Message) *iolang.Object {
	return self
}

func (tagFile) CloneValue(value interface{}) interface{} {
	return File{Mode: "update"}
}

func (tagFile) String() string {
	return "File"
}

// FileTag is the Tag for File objects. Activate returns self. CloneValue
// returns a new, unopened file with an empty path and mode set to "update".
var FileTag tagFile

// New creates a File object with the given file. The mode string should
// be one of "read", "update", or "append", depending on the flags used when
// opening the file.
func New(vm *iolang.VM, file *os.File, mode string) *iolang.Object {
	f := File{
		File: file,
		Mode: mode,
	}
	if file != nil {
		f.Path = file.Name()
	}
	return vm.ObjectWith(nil, vm.CoreProto("File"), f, FileTag)
}

// NewAt creates a File object unopened at the given path. The mode will be
// set to "read". The path should use the OS's separator convention.
func NewAt(vm *iolang.VM, path string) *iolang.Object {
	f := File{
		Path: path,
		Mode: "read",
	}
	return vm.ObjectWith(nil, vm.CoreProto("File"), f, FileTag)
}

// ReadLine reads one line from the file such that the file cursor will be
// positioned after the first encountered newline. The line without the newline
// is returned, along with any error, which may include io.EOF. If the file
// reaches EOF while reading, then the returned eof value will be true.
func (f File) ReadLine() (line []byte, eof bool, err error) {
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
			eof = true
		}
	}
	return
}

func init() {
	internal.Register(initFile)
}

func initFile(vm *iolang.VM) {
	slots := iolang.Slots{
		"at":                 vm.NewCFunction(at, FileTag),
		"atPut":              vm.NewCFunction(atPut, FileTag),
		"close":              vm.NewCFunction(fileClose, FileTag),
		"contents":           vm.NewCFunction(contents, FileTag),
		"descriptor":         vm.NewCFunction(descriptor, FileTag),
		"exists":             vm.NewCFunction(exists, FileTag),
		"flush":              vm.NewCFunction(flush, FileTag),
		"foreach":            vm.NewCFunction(foreach, FileTag),
		"foreachLine":        vm.NewCFunction(foreachLine, FileTag),
		"isAtEnd":            vm.NewCFunction(isAtEnd, FileTag),
		"isDirectory":        vm.NewCFunction(isDirectory, FileTag),
		"isLink":             vm.NewCFunction(isLink, FileTag),
		"isOpen":             vm.NewCFunction(isOpen, FileTag),
		"isPipe":             vm.NewCFunction(isPipe, FileTag),
		"isRegularFile":      vm.NewCFunction(isRegularFile, FileTag),
		"isSocket":           vm.NewCFunction(isSocket, FileTag),
		"isUserExecutable":   vm.NewCFunction(isUserExecutable, FileTag),
		"lastDataChangeDate": vm.NewCFunction(lastDataChangeDate, FileTag),
		"mode":               vm.NewCFunction(mode, FileTag),
		"moveTo":             vm.NewCFunction(moveTo, FileTag),
		"name":               vm.NewCFunction(name, FileTag),
		"open":               vm.NewCFunction(open, FileTag),
		"openForAppending":   vm.NewCFunction(openForAppending, FileTag),
		"openForReading":     vm.NewCFunction(openForReading, FileTag),
		"openForUpdating":    vm.NewCFunction(openForUpdating, FileTag),
		"path":               vm.NewCFunction(path, FileTag),
		"position":           vm.NewCFunction(position, FileTag),
		"positionAtEnd":      vm.NewCFunction(positionAtEnd, FileTag),
		"protectionMode":     vm.NewCFunction(protectionMode, FileTag),
		"readBufferOfLength": vm.NewCFunction(readBufferOfLength, FileTag),
		"readLine":           vm.NewCFunction(readLine, FileTag),
		"readLines":          vm.NewCFunction(readLines, FileTag),
		"readStringOfLength": vm.NewCFunction(readStringOfLength, FileTag),
		"readToBufferLength": vm.NewCFunction(readToBufferLength, FileTag),
		"readToEnd":          vm.NewCFunction(readToEnd, FileTag),
		"remove":             vm.NewCFunction(remove, FileTag),
		"rewind":             vm.NewCFunction(rewind, FileTag),
		"setPath":            vm.NewCFunction(setPath, FileTag),
		"setPosition":        vm.NewCFunction(setPosition, FileTag),
		"size":               vm.NewCFunction(size, FileTag),
		"temporaryFile":      vm.NewCFunction(temporaryFile, nil),
		"truncateToSize":     vm.NewCFunction(truncateToSize, FileTag),
		"type":               vm.NewString("File"),
		"write":              vm.NewCFunction(write, FileTag),

		// Methods with platform-dependent implementations:
		"groupId":            vm.NewCFunction(groupID, FileTag),
		"lastAccessDate":     vm.NewCFunction(lastAccessDate, FileTag),
		"lastInfoChangeDate": vm.NewCFunction(lastInfoChangeDate, FileTag),
		"userId":             vm.NewCFunction(userID, FileTag),
	}
	slots["asBuffer"] = slots["contents"]
	slots["descriptorId"] = slots["descriptor"]
	proto := internal.CoreInstall(vm, "File", slots, File{}, FileTag)

	stdin := New(vm, os.Stdin, "read")
	stdout := New(vm, os.Stdout, "")
	stderr := New(vm, os.Stderr, "")
	vm.SetSlot(stdout, "mode", vm.Nil)
	vm.SetSlot(stderr, "mode", vm.Nil)
	slots = iolang.Slots{
		"standardInput":  stdin,
		"standardOutput": stdout,
		"standardError":  stderr,
	}
	vm.SetSlots(proto, slots)
	internal.Ioz(vm, coreIo, coreFiles)
}

// FileAt is a File method.
//
// at returns as a Number the byte in the file at a given position.
func at(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	b := []byte{0}
	switch _, err := f.File.ReadAt(b, int64(n)); err {
	case nil:
		return vm.NewNumber(float64(b[0]))
	case io.EOF:
		return vm.Nil
	default:
		return vm.IoError(err)
	}
}

// FileAtPut is a File method.
//
// atPut writes a single byte to the file at a given position.
func atPut(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	c, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	if _, err := f.File.WriteAt([]byte{byte(c)}, int64(n)); err != nil {
		return vm.IoError(err)
	}
	return target
}

// FileClose is a File method.
//
// close closes the file.
func fileClose(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	switch f.File {
	case nil, os.Stdin, os.Stdout, os.Stderr:
		// Do nothing.
	default:
		// TODO: Io does extra stuff for pipes, esp. on non-Windows.
		if err := f.File.Close(); err != nil {
			return vm.IoError(err)
		}
		f.File = nil
		target.Lock()
		target.Value = f.File
		target.Unlock()
	}
	return target
}

// FileContents is a File method.
//
// contents reads the contents of the file into a buffer object. Same as
// asBuffer, but can also read standardInput.
func contents(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	if f.File == os.Stdin {
		b, err := ioutil.ReadAll(f.File)
		if err != nil {
			return vm.IoError(err)
		}
		return vm.NewSequence(b, true, "utf8")
	}
	b, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewSequence(b, true, "utf8")
}

// FileDescriptor is a File method.
//
// descriptor returns the underlying file descriptor as a Number.
func descriptor(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.NewNumber(float64(f.File.Fd()))
}

// FileExists is a File method.
//
// exists returns whether a file with this file's path exists.
func exists(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	_, err := os.Stat(f.Path)
	if err == nil {
		return vm.True
	}
	if os.IsNotExist(err) {
		return vm.False
	}
	return vm.IoError(err)
}

// FileFlush is a File method.
//
// flush synchronizes the file state between the program and the operating
// system.
func flush(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	if err := f.File.Sync(); err != nil {
		return vm.IoError(err)
	}
	return target
}

// FileForeach is a File method.
//
// foreach executes a message for each byte of the file.
func foreach(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) (result *iolang.Object) {
	kn, vn, hkn, hvn, ev := iolang.ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseExceptionf("foreach requires 2 or 3 arguments")
	}
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	var control iolang.Stop
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
				case iolang.NoStop, iolang.ContinueStop: // do nothing
				case iolang.BreakStop:
					return result
				case iolang.ReturnStop, iolang.ExceptionStop, iolang.ExitStop:
					return vm.Stop(result, control)
				default:
					panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
				}
				k++
				j++
			}
			if err == io.EOF {
				f.EOF = true
				target.Lock()
				target.Value = f
				target.Unlock()
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
					target.Lock()
					target.Value = f
					target.Unlock()
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
			case iolang.NoStop, iolang.ContinueStop: // do nothing
			case iolang.BreakStop:
				return result
			case iolang.ReturnStop, iolang.ExceptionStop, iolang.ExitStop:
				return vm.Stop(result, control)
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
			}
		}
	}
	return result
}

// FileForeachLine is a File method.
//
// foreachLine executes a message for each line of the file.
func foreachLine(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) (result *iolang.Object) {
	kn, vn, hkn, hvn, ev := iolang.ForeachArgs(msg)
	if ev == nil {
		return vm.RaiseExceptionf("foreach requires 1, 2, or 3 arguments")
	}
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	k := 0
	var control iolang.Stop
	for {
		// f.ReadLine implements the same logic as FileForeach above.
		line, eof, err := f.ReadLine()
		if eof {
			f.EOF = true
			target.Lock()
			target.Value = f
			target.Unlock()
		}
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
			case iolang.NoStop, iolang.ContinueStop: // do nothing
			case iolang.BreakStop:
				return result
			case iolang.ReturnStop, iolang.ExceptionStop, iolang.ExitStop:
				return vm.Stop(result, control)
			default:
				panic(fmt.Sprintf("iolang: invalid Stop: %v", control))
			}
			if err != nil {
				break
			}
			k++
		}
	}
	return result
}

// FileIsAtEnd is a File method.
//
// isAtEnd returns true if the file is at EOF.
func isAtEnd(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.IoBool(f.EOF)
}

// FileIsDirectory is a File method.
//
// isDirectory returns true if the path of the file is a directory.
func isDirectory(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.IsDir())
}

// FileIsLink is a File method.
//
// isLink returns true if the path of the file is a symbolic link.
func isLink(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Lstat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeSymlink != 0)
}

// FileIsOpen is a File method.
//
// isOpen returns true if the file is open.
func isOpen(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.IoBool(f.File != nil)
}

// FileIsPipe is a File method.
//
// isPipe returns true if the path of the file is a named pipe.
func isPipe(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeNamedPipe != 0)
}

// FileIsRegularFile is a File method.
//
// isRegularFile returns true if the path of the file is a regular file.
func isRegularFile(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode().IsRegular())
}

// FileIsSocket is a File method.
//
// isSocket returns true if the path of the file is a socket.
func isSocket(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode()&os.ModeSocket != 0)
}

// FileIsUserExecutable is a File method.
//
// isUserExecutable returns true if the path of the file is executable by its
// owner. Always false on Windows.
func isUserExecutable(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.IoBool(fi.Mode().Perm()&0100 != 0)
}

// FileLastDataChangeDate is a File method.
//
// lastDataChangeDate returns the date at which the file's contents were last
// modified.
func lastDataChangeDate(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return date.New(vm, fi.ModTime())
}

// FileMode is a File method.
//
// mode returns a string describing the file's mode; one of "read", "update",
// or "append".
func mode(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.NewString(f.Mode)
}

// FileMoveTo is a File method.
//
// moveTo moves the file at the file's path to the given path.
func moveTo(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	to := filepath.FromSlash(s)
	if err := os.Rename(f.Path, to); err != nil {
		return vm.IoError(err)
	}
	return target
}

// FileName is a File method.
//
// FileName returns the name of the file or directory at the file's path,
// similar to UNIX basename.
func name(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.NewString(filepath.Base(f.Path))
}

// FileOpen is a File method.
//
// open opens the file.
func open(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
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
	target.Lock()
	target.Value = f
	target.Unlock()
	return target
}

// FileOpenForAppending is a File method.
//
// openForAppending opens the file for appending.
func openForAppending(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	f.Mode = "append"
	target.Value = f
	target.Unlock()
	return open(vm, target, locals, msg)
}

// FileOpenForReading is a File method.
//
// openForReading opens the file for reading.
func openForReading(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	f.Mode = "read"
	target.Value = f
	target.Unlock()
	return open(vm, target, locals, msg)
}

// FileOpenForUpdating is a File method.
//
// openForUpdating opens the file for updating.
func openForUpdating(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	f.Mode = "update"
	target.Value = f
	target.Unlock()
	return open(vm, target, locals, msg)
}

// FilePath is a File method.
//
// path returns the file's absolute path.
func path(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	return vm.NewString(filepath.ToSlash(f.Path))
}

// FilePosition is a File method.
//
// position returns the current position of the file cursor.
func position(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	p, err := f.File.Seek(0, io.SeekCurrent)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(p))
}

// FilePositionAtEnd is a File method.
//
// positionAtEnd moves the file cursor to the end of the file.
func positionAtEnd(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	_, err := f.File.Seek(0, io.SeekEnd)
	if err != nil {
		return vm.IoError(err)
	}
	f.EOF = false
	target.Lock()
	target.Value = f
	target.Unlock()
	return target
}

// FileProtectionMode is a File method.
//
// protectionMode returns the stat mode of the path of the file as a Number.
func protectionMode(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(fi.Mode()))
}

// FileReadBufferOfLength is a File method.
//
// readBufferOfLength reads the specified number of bytes into a Sequence.
func readBufferOfLength(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	count, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	if count < 0 {
		return vm.RaiseExceptionf("can't read negative bytes")
	}
	b := make([]byte, int(count))
	n, err := f.File.Read(b)
	if n > 0 {
		return vm.NewSequence(b, true, "utf8")
	}
	if err != nil {
		if err != io.EOF {
			return vm.IoError(err)
		}
		f.EOF = true
		target.Lock()
		target.Value = f
		target.Unlock()
	}
	return vm.Nil
}

// FileReadLine is a File method.
//
// readLine reads a line from the file.
func readLine(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	b, eof, err := f.ReadLine()
	if eof {
		f.EOF = true
		target.Lock()
		target.Value = f
		target.Unlock()
	}
	if err != nil {
		if err == io.EOF && len(b) == 0 {
			return vm.Nil
		}
		return vm.IoError(err)
	}
	return vm.NewSequence(b, true, "utf8")
}

// FileReadLines is a File method.
//
// readLines returns a List containing all lines in the file.
func readLines(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	l := []*iolang.Object{}
	for {
		b, eof, err := f.ReadLine()
		if eof {
			f.EOF = true
			target.Lock()
			target.Value = f
			target.Unlock()
		}
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
		return vm.NewList(l...)
	}
	return vm.Nil
}

// FileReadStringOfLength is a File method.
//
// readStringOfLength reads a string up to the given length from the file.
func readStringOfLength(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	count, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	if count < 0 {
		return vm.RaiseExceptionf("can't read negative bytes")
	}
	b := make([]byte, int(count))
	n, err := f.File.Read(b)
	if n > 0 {
		return vm.NewSequence(b, false, "utf8")
	}
	if err != nil {
		if err != io.EOF {
			return vm.IoError(err)
		}
		f.EOF = true
		target.Lock()
		target.Value = f
		target.Unlock()
	}
	return vm.Nil
}

// FileReadToBufferLength is a File method.
//
// readToBufferLength reads the number of items given in the second argument
// and appends them to the sequence given in the first. Returns the number of
// elements actually read.
func readToBufferLength(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	seq, obj, stop := msg.SequenceArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(obj, stop)
	}
	n, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	if n < 0 {
		return vm.RaiseExceptionf("cannot read negative elements")
	}
	target.Lock()
	defer target.Unlock()
	f := target.Value.(File)
	obj.Lock()
	defer obj.Unlock()
	if err := seq.CheckMutable("File readToBufferLength"); err != nil {
		return vm.IoError(err)
	}
	kind := seq.Kind()
	is := kind.ItemSize()
	b := make([]byte, int(n)*is)
	k, err := f.File.Read(b)
	if err != nil {
		if err != io.EOF {
			return vm.IoError(err)
		}
		f.EOF = true
		target.Lock()
		target.Value = f
		target.Unlock()
	}
	b = b[:(k+is-1)/is*is]
	x := vm.SequenceFromBytes(b, kind)
	obj.Value = seq.Append(x)
	return vm.NewNumber(float64(len(b) / is))
}

// FileReadToEnd is a File method.
//
// readToEnd reads chunks of a given size (default 4096) to the end of the file
// and returns a Sequence containing the bytes read.
func readToEnd(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	sz := 4096
	if len(msg.Args) > 0 {
		n, exc, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != iolang.NoStop {
			return vm.Stop(exc, stop)
		}
		if n >= 1 {
			sz = int(n)
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
			target.Lock()
			target.Value = f
			target.Unlock()
			v = append(v, b[:n]...)
			break
		}
		if n > 0 {
			v = append(v, b[:n]...)
		} else {
			break
		}
	}
	if len(v) > 0 {
		return vm.NewSequence(v, true, "utf8")
	}
	return vm.Nil
}

// FileRemove is a File method.
//
// remove removes the file at the file's path.
func remove(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	err := os.Remove(f.Path)
	if err != nil && !os.IsNotExist(err) {
		return vm.IoError(err)
	}
	return target
}

// FileRewind is a File method.
//
// rewind returns the file cursor to the beginning of the file.
func rewind(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	_, err := f.File.Seek(0, io.SeekStart)
	if err != nil {
		return vm.IoError(err)
	}
	f.EOF = false
	target.Lock()
	target.Value = f
	target.Unlock()
	return target
}

// FileSetPath is a File method.
//
// setPath sets the file's path.
func setPath(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	f.Path = filepath.FromSlash(s)
	target.Lock()
	target.Value = f
	target.Unlock()
	return target
}

// FileSetPosition is a File method.
//
// setPosition changes the file cursor's location.
func setPosition(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	_, err := f.File.Seek(int64(n), io.SeekStart)
	if err != nil {
		return vm.IoError(err)
	}
	// We might have seeked to or past the end of the file, in which case it's
	// technically false to say we aren't at EOF, but the file might also grow
	// between the seek and the next read, in which case claiming EOF is
	// misleading. It's also easier to implement it this way.
	f.EOF = false
	target.Lock()
	target.Value = f
	target.Unlock()
	return target
}

// FileSize is a File method.
//
// size determines the file size.
func size(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	fi, err := os.Stat(f.Path)
	if err != nil {
		return vm.IoError(err)
	}
	return vm.NewNumber(float64(fi.Size()))
}

// FileTemporaryFile is a File method.
//
// temporaryFile creates a file that did not exist previously. It is not
// guaranteed to be removed at any point.
func temporaryFile(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	fp, err := ioutil.TempFile("", "iolang_temp")
	if err != nil {
		return vm.IoError(err)
	}
	return New(vm, fp, "update")
}

// FileTruncateToSize is a File method.
//
// truncateToSize truncates the file to the given size.
func truncateToSize(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	err := f.File.Truncate(int64(n))
	if err != nil {
		return vm.IoError(err)
	}
	return target
}

// FileWrite is a File method.
//
// write writes its arguments to the file.
func write(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	f := target.Value.(File)
	target.Unlock()
	for i := range msg.Args {
		s, obj, stop := msg.SequenceArgAt(vm, locals, i)
		if stop != iolang.NoStop {
			return vm.Stop(obj, stop)
		}
		obj.Lock()
		v := s.Bytes()
		obj.Unlock()
		_, err := f.File.Write(v)
		if err != nil {
			return vm.IoError(err)
		}
	}
	return target
}
