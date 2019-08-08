package main

import (
	"bufio"
	"fmt"
	"github.com/zephyrtronium/iolang"
	"os"

	"runtime"
	"runtime/pprof"
)

func main() {
	vm := iolang.NewVM(os.Args[1:]...)
	setupStaticAddons(vm)
	vm.Lobby.SetSlots(iolang.Slots{
		"ps1":       vm.NewString("io> "),
		"ps2":       vm.NewString("... "),
		"isRunning": vm.True,
		"profiled":  vm.NewCFunction(profiled, nil),
	})
	vm.MustDoString(`Lobby setSlot("exit", method(Lobby setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	for isRunning, _ := vm.GetSlot(vm.Lobby, "isRunning"); vm.AsBool(isRunning); isRunning, _ = vm.GetSlot(vm.Lobby, "isRunning") {
		ps1, _ := vm.GetSlot(vm.Lobby, "ps1")
		fmt.Print(ps1.(*iolang.Sequence).String())
		ok := stdin.Scan()
		x, stop := vm.DoString(stdin.Text(), "Command Line")
		if stop == iolang.ExceptionStop {
			if ex, ok := x.(*iolang.Exception); ok {
				fmt.Println("Exception:")
				for i := len(ex.Stack) - 1; i >= 0; i-- {
					m := ex.Stack[i]
					if m.IsStart() {
						fmt.Printf("\t%s\t%s:%d\n", m.Name(), m.Label, m.Line)
					} else {
						fmt.Printf("\t%s %s\t%s:%d\n", m.Prev.Name(), m.Name(), m.Label, m.Line)
					}
				}
			} else {
				fmt.Println("Raised as exception:")
				fmt.Println("\t", vm.AsString(x))
			}
		}
		fmt.Println(vm.AsString(x))
		if !ok {
			break
		}
	}
	fmt.Println(stdin.Err())
}

func profiled(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) (iolang.Interface, iolang.Stop) {
	cpu, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return aerr, stop
	}
	mem, aerr, stop := msg.StringArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return aerr, stop
	}
	cf, err := os.Create(cpu.String())
	if err != nil {
		return vm.IoError(err)
	}
	defer cf.Close()
	mf, err := os.Create(mem.String())
	if err != nil {
		return vm.IoError(err)
	}
	defer mf.Close()
	m := msg.ArgAt(2)
	if err = pprof.StartCPUProfile(cf); err != nil {
		return vm.IoError(err)
	}
	defer pprof.StopCPUProfile()
	v, stop := m.Send(vm, target, locals)
	runtime.GC()
	if err = pprof.WriteHeapProfile(mf); err != nil {
		return vm.IoError(err)
	}
	return v, stop
}
