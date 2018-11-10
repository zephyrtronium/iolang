package iolang

import (
	"os"
	"path/filepath"
	"runtime"
)

func (vm *VM) initSystem() {
	slots := Slots{
		"activeCpus":             vm.NewCFunction(SystemActiveCpus),
		"exit":                   vm.NewCFunction(SystemExit),
		"getEnvironmentVariable": vm.NewCFunction(SystemGetEnvironmentVariable),
		"iovmName":               vm.NewString("github.com/zephyrtronium/iolang"),
		// TODO: iospecVersion
		// launchScript should be set by the VM creator as appropriate.
		"launchScript": vm.Nil,
		"platform":     vm.NewString(runtime.GOOS),
		// TODO: platformVersion
		"setEnvironmentVariable": vm.NewCFunction(SystemSetEnvironmentVariable),
		"setLobby":               vm.NewCFunction(SystemSetLobby),
		// TODO: sleep
		// TODO: system
		"thisProcessPid": vm.NewCFunction(SystemThisProcessPid),
		"type":           vm.NewString("System"),
		// TODO: version
	}
	args := make([]Interface, len(os.Args)-1)
	for i, v := range os.Args[1:] {
		args[i] = vm.NewString(v)
	}
	slots["args"] = vm.NewList(args...)
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
	SetSlot(vm.Core, "System", vm.ObjectWith(slots))
}

// SystemActiveCpus is a System method.
//
// activeCpus returns the number of CPUs available for coroutines.
func SystemActiveCpus(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(runtime.GOMAXPROCS(0)))
}

// SystemExit is a System method.
//
// exit exits the process with an exit code which defaults to 0.
func SystemExit(vm *VM, target, locals Interface, msg *Message) Interface {
	code := 0
	if len(msg.Args) > 0 {
		n, err := msg.NumberArgAt(vm, locals, 0)
		if err != nil {
			return vm.IoError(err)
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
func SystemGetEnvironmentVariable(vm *VM, target, locals Interface, msg *Message) Interface {
	name, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	s, ok := os.LookupEnv(name.String())
	if ok {
		return vm.NewString(s)
	}
	return vm.Nil
}

// SystemSetEnvironmentVariable is a System method.
//
// setEnvironmentVariable sets the value of an environment variable.
func SystemSetEnvironmentVariable(vm *VM, target, locals Interface, msg *Message) Interface {
	name, err := msg.StringArgAt(vm, locals, 0)
	if err != nil {
		return vm.IoError(err)
	}
	val, err := msg.StringArgAt(vm, locals, 1)
	if err != nil {
		return vm.IoError(err)
	}
	err = os.Setenv(name.String(), val.String())
	if err != nil {
		return vm.IoError(err)
	}
	return target
}

// SystemSetLobby is a System method.
//
// setLobby changes the Lobby. This had garbage collection implications in the
// original Io, but is mostly irrelevant in this implementation due to use of
// Go's GC.
func SystemSetLobby(vm *VM, target, locals Interface, msg *Message) Interface {
	o, ok := CheckStop(msg.EvalArgAt(vm, locals, 0), LoopStops)
	if !ok {
		return o
	}
	vm.Lobby = o.SP()
	return target
}

// SystemThisProcessPid is a System method.
//
// thisProcessPid returns the pid of this process.
func SystemThisProcessPid(vm *VM, target, locals Interface, msg *Message) Interface {
	return vm.NewNumber(float64(os.Getpid()))
}
