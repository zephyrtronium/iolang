package main

import (
	"bufio"
	"fmt"
	"github.com/zephyrtronium/iolang"
	"os"
)

func main() {
	vm := iolang.NewVM(os.Args[1:]...)
	setupStaticAddons(vm)
	vm.Lobby.SetSlots(iolang.Slots{
		"ps1":       vm.NewString("io> "),
		"ps2":       vm.NewString("... "),
		"isRunning": vm.True,
	})
	vm.MustDoString(`Lobby setSlot("exit", method(Lobby setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	for isRunning, _ := vm.Lobby.GetSlot("isRunning"); vm.AsBool(isRunning); isRunning, _ = vm.Lobby.GetSlot("isRunning") {
		ps1, _ := vm.Lobby.GetSlot("ps1")
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
