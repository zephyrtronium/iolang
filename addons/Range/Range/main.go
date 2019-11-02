package main

import (
	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/addons/Range"
)

// IoAddon returns an object to load the addon.
func IoAddon() iolang.Addon {
	return Range.IoAddon()
}
