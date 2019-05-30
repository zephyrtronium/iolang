package main

import (
	"bufio"
	"fmt"
	"github.com/zephyrtronium/iolang"
	"os"
)

func main() {
	vm := iolang.NewVM(os.Args[1:]...)
	iolang.SetSlot(vm.Lobby, "ps1", vm.NewString("io> "))
	iolang.SetSlot(vm.Lobby, "ps2", vm.NewString("... "))
	iolang.SetSlot(vm.Lobby, "isRunning", vm.True)
	vm.MustDoString(`Lobby setSlot("exit", method(Lobby setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	for isRunning, _ := iolang.GetSlot(vm.Lobby, "isRunning"); vm.AsBool(isRunning); isRunning, _ = iolang.GetSlot(vm.Lobby, "isRunning") {
		ps1, _ := iolang.GetSlot(vm.Lobby, "ps1")
		fmt.Print(ps1.(*iolang.Sequence).String())
		ok := stdin.Scan()
		x, eok := iolang.CheckStop(vm.DoString(stdin.Text(), "Command Line"), iolang.ReturnStop)
		if !eok {
			stop := x.(iolang.Stop)
			fmt.Println("Exception:")
			for i := len(stop.Stack) - 1; i >= 0; i-- {
				m := stop.Stack[i]
				if m.IsStart() {
					fmt.Printf("\t%s\t%s:%d\n", m.Name(), m.Label, m.Line)
				} else {
					fmt.Printf("\t%s %s\t%s:%d\n", m.Prev.Name(), m.Name(), m.Label, m.Line)
				}
			}
			x = stop.Result
		}
		fmt.Println(vm.AsString(x))
		if !ok {
			break
		}
	}
	fmt.Println(stdin.Err())
}
