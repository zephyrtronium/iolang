package main

import (
	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/addons/range"
)

// OpenAddon returns an object to load the addon.
func OpenAddon(vm *iolang.VM) iolang.Addon {
	return iorange.OpenAddon(vm)
}
