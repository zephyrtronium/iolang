package main

import (
	"bufio"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"github.com/zephyrtronium/iolang"
	"os"
)

func main() {
	vm := iolang.NewVM(os.Args[1:]...)
	// iolang.Debugvm = vm
	iolang.SetSlot(vm.Lobby, "ps1", vm.NewString("io> "))
	iolang.SetSlot(vm.Lobby, "ps2", vm.NewString("... "))
	iolang.SetSlot(vm.Lobby, "isRunning", vm.True)
	vm.MustDoString(`Lobby setSlot("exit", method(Lobby setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	// spew.Config.MaxDepth = 2
	for isRunning, _ := iolang.GetSlot(vm.Lobby, "isRunning"); vm.AsBool(isRunning); isRunning, _ = iolang.GetSlot(vm.Lobby, "isRunning") {
		ps1, _ := iolang.GetSlot(vm.Lobby, "ps1")
		fmt.Print(ps1.(*iolang.Sequence).String())
		ok := stdin.Scan()
		x, _ := iolang.CheckStop(vm.DoString(stdin.Text(), "Command Line"), iolang.ExceptionStop)
		// spew.Dump(x)
		fmt.Println(vm.AsString(x))
		if !ok {
			break
		}
	}
	fmt.Println(stdin.Err())
}
