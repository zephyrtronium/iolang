package iolang

import (
	"os"
	"path/filepath"
	"runtime"
)

func (vm *VM) initSystem() {
	slots := Slots{
		"activeCpus":             vm.NewCFunction(SystemActiveCpus, nil),
		"arch":                   vm.NewString(runtime.GOARCH),
		"exit":                   vm.NewCFunction(SystemExit, nil),
		"getEnvironmentVariable": vm.NewCFunction(SystemGetEnvironmentVariable, nil),
		"iovmName":               vm.NewString("github.com/zephyrtronium/iolang"),
		"iospecVersion":          vm.NewString(IoSpecVer),
		"launchScript":           vm.Nil,
		"platform":               vm.NewString(runtime.GOOS),
		"platformVersion":        vm.NewString(platformVersion),
		"setEnvironmentVariable": vm.NewCFunction(SystemSetEnvironmentVariable, nil),
		"setLobby":               vm.NewCFunction(SystemSetLobby, nil),
		// TODO: sleep
		// TODO: system
		"thisProcessPid": vm.NewCFunction(SystemThisProcessPid, nil),
		"type":           vm.NewString("System"),
		"version":        vm.NewString(IoVersion),
	}
	// installPrefix is the directory two above the executable path, and ioPath
	// is $installPrefix/lib/io. It is notable that paths on the System object
	// use the operating system's path separators, unlike most other paths in
	// Io, which are / only.
	//
	// In the case that Io is launched via `go run`, this will be nonsense. The
	// industrious thing to do would be to search $GOPATH for something that
	// looks like this package, but it would be expensive and not guaranteed to
	// be correct. Instead, the nonsense shall remain nonsense.
	exe, err := os.Executable()
	if err == nil {
		ip := filepath.Dir(filepath.Dir(exe))
		slots["installPrefix"] = vm.NewString(ip)
		slots["ioPath"] = vm.NewString(filepath.Join(ip, "lib", "io"))
	} else {
		// os.Executable is unsupported on nacl.
		// TODO: no idea what should be reasonable here.
		slots["installPrefix"] = vm.NewString("")
		slots["ioPath"] = vm.NewString("")
	}
	// launchPath is the working directory at the time of VM initialization,
	// which is now.
	wd, err := os.Getwd()
	if err == nil {
		slots["launchPath"] = vm.NewString(wd)
	} else {
		slots["launchPath"] = vm.Nil
	}
	vm.SetSlot(vm.Core, "System", vm.ObjectWith(slots))
}

func (vm *VM) initArgs(args []string) {
	l := make([]Interface, len(args))
	for i, v := range args {
		l[i] = vm.NewString(v)
	}
	s, _ := vm.GetLocalSlot(vm.Core, "System")
	vm.SetSlot(s, "args", vm.NewList(l...))
}

// SetLaunchScript sets the System launchScript slot to the given string, as a
// convenience for VM creators who intend to execute that Io source file. The
// default System launchScript value is nil, which signifies an interactive
// session.
func (vm *VM) SetLaunchScript(path string) {
	s, ok := vm.GetLocalSlot(vm.Core, "System")
	if !ok {
		// No System. Is a "DOES NOT COMPUTE" joke sufficiently witty here?
		panic("iolang.SetLaunchScript: no Core System")
	}
	vm.SetSlot(s, "launchScript", vm.NewString(path))
}

// SystemActiveCpus is a System method.
//
// activeCpus returns the number of CPUs available for coroutines.
func SystemActiveCpus(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(runtime.GOMAXPROCS(0))), NoStop
}

// SystemExit is a System method.
//
// exit exits the process with an exit code which defaults to 0.
func SystemExit(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	code := 0
	if len(msg.Args) > 0 {
		n, err, stop := msg.NumberArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		code = int(n.Value)
	}
	os.Exit(code)
	panic("unreachable")
}

// SystemGetEnvironmentVariable is a System method.
//
// getEnvironmentVariable returns the value of the environment variable with
// the given name, or nil if it does not exist.
func SystemGetEnvironmentVariable(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	name, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	s, ok := os.LookupEnv(name.String())
	if ok {
		return vm.NewString(s), NoStop
	}
	return vm.Nil, NoStop
}

// SystemSetEnvironmentVariable is a System method.
//
// setEnvironmentVariable sets the value of an environment variable.
func SystemSetEnvironmentVariable(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	name, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	val, aerr, stop := msg.StringArgAt(vm, locals, 1)
	if stop != NoStop {
		return aerr, stop
	}
	err := os.Setenv(name.String(), val.String())
	if err != nil {
		return vm.IoError(err)
	}
	return target, NoStop
}

// SystemSetLobby is a System method.
//
// setLobby changes the Lobby. This had garbage collection implications in the
// original Io, but is mostly irrelevant in this implementation due to use of
// Go's GC.
func SystemSetLobby(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	o, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return o, stop
	}
	vm.Lobby = o
	return target, NoStop
}

// SystemThisProcessPid is a System method.
//
// thisProcessPid returns the pid of this process.
func SystemThisProcessPid(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewNumber(float64(os.Getpid())), NoStop
}
