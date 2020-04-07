package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/zephyrtronium/iolang"
	// import for side effects
	_ "github.com/zephyrtronium/iolang/coreext"

	"runtime"
	"runtime/pprof"
)

func main() {
	vm := iolang.NewVM(os.Args[1:]...)
	setupStaticAddons(vm)
	vm.SetSlots(vm.Lobby, iolang.Slots{
		"ps1":       vm.NewString("io> "),
		"ps2":       vm.NewString("... "),
		"isRunning": vm.True,
		"profiled":  vm.NewCFunction(profiled, nil),
	})
	vm.MustDoString(`Lobby setSlot("exit", method(Lobby setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	for isRunning, _ := vm.GetSlot(vm.Lobby, "isRunning"); vm.IsAlive() && vm.AsBool(isRunning); isRunning, _ = vm.GetSlot(vm.Lobby, "isRunning") {
		p := "io> "
		ps1, _ := vm.GetSlot(vm.Lobby, "ps1")
		if ps1 != nil {
			s, ok := ps1.Value.(iolang.Sequence)
			if ok {
				ps1.Lock()
				p = s.String()
				ps1.Unlock()
			}
		}
		fmt.Print(p)
		ok := stdin.Scan()
		x, stop := vm.DoString(stdin.Text(), "Command Line")
		if stop == iolang.ExceptionStop {
			if ex, ok := x.Value.(iolang.Exception); ok {
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
		if !ok || !vm.IsAlive() {
			break
		}
		fmt.Println(vm.AsString(x))
	}
	fmt.Println(stdin.Err())
}

func profiled(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	cpu, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	mem, exc, stop := msg.StringArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	cf, err := os.Create(cpu)
	if err != nil {
		return vm.IoError(err)
	}
	defer cf.Close()
	mf, err := os.Create(mem)
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
	return vm.Stop(v, stop)
}
