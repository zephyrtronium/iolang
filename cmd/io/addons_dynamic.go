// +build !static_addons
// +build linux,cgo darwin,cgo

// Keep build constraints updated as the requirements to use package plugin
// change.

package main

import "github.com/zephyrtronium/iolang"

func setupStaticAddons(vm *iolang.VM) {}
