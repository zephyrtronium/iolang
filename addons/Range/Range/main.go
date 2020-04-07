package main

import (
	"github.com/zephyrtronium/iolang/coreext/addon"
	"github.com/zephyrtronium/iolang/addons/Range"
)

// IoAddon returns an object to load the addon.
func IoAddon() addon.Interface {
	return Range.IoAddon()
}
