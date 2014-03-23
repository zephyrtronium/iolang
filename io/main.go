package main

import (
	"bufio"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"github.com/zephyrtronium/iolang"
	"os"
)

func main() {
	vm := iolang.NewVM()
	// iolang.Debugvm = vm
	iolang.SetSlot(vm.Lobby, "ps1", vm.NewString("io> "))
	iolang.SetSlot(vm.Lobby, "ps2", vm.NewString("... "))
	iolang.SetSlot(vm.Lobby, "isRunning", vm.True)
	// method() doesn't exist yet, but oh well.
	vm.DoString(`Lobby setSlot("exit", method(self setSlot("isRunning", false)))`)

	stdin := bufio.NewScanner(os.Stdin)
	// spew.Config.MaxDepth = 2
	for isRunning, _ := iolang.GetSlot(vm.Lobby, "isRunning"); vm.AsBool(isRunning); isRunning, _ = iolang.GetSlot(vm.Lobby, "isRunning") {
		ps1, _ := iolang.GetSlot(vm.Lobby, "ps1")
		fmt.Print(ps1.(*iolang.String).Value)
		ok := stdin.Scan()
		x := vm.DoString(stdin.Text())
		// spew.Dump(x)
		fmt.Println(x)
		if !ok {
			break
		}
	}
	fmt.Println(stdin.Err())
}
