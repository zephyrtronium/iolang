// +build static_addons linux,!cgo darwin,!cgo !linux,!darwin

// Keep build constraints updated as the requirements to use package plugin
// change.

package main

import (
	"github.com/zephyrtronium/iolang"

	iorange "github.com/zephyrtronium/iolang/addons/range"
)

const numStaticAddons = 1

func setupStaticAddons(vm *iolang.VM) {
	ch := make(chan *iolang.Object, numStaticAddons)
	go func() { ch <- <-vm.LoadAddon(iorange.OpenAddon(vm)) }()
	for i := 0; i < numStaticAddons; i++ {
		<-ch
	}
}
