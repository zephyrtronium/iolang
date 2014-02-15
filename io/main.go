package main

import (
	"bufio"
	"fmt"
	"github.com/zephyrtronium/iolang"
	"os"
)

func main() {
	vm := iolang.NewVM()
	iolang.SetSlot(vm.Lobby, "ps1", vm.NewString("io> "))
	iolang.SetSlot(vm.Lobby, "ps2", vm.NewString("... "))
	iolang.SetSlot(vm.Lobby, "isRunning", vm.True)
	vm.DoString("exit := Lobby method(self isRunning = false)")

	stdin := bufio.NewScanner(os.Stdin)
	for isRunning, _ := iolang.GetSlot(vm.Lobby, "isRunning"); vm.AsBool(isRunning); isRunning, _ = iolang.GetSlot(vm.Lobby, "isRunning") {
		ps1, _ := iolang.GetSlot(vm.Lobby, "ps1")
		fmt.Print(ps1.(*iolang.String).Value)
		ok := stdin.Scan()
		x := vm.DoString(stdin.Text())
		fmt.Println(x)
		if !ok {
			break
		}
	}
	fmt.Println(stdin.Err())
}
