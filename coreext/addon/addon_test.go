package addon_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/addon" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// Directory is a dependency.
	slots := []string{
		"Addon",
		"AddonLoader",
		"Directory",
		"tildeExpandsTo",
	}
	testutils.CheckNewSlots(t, testutils.VM().Core, slots)
}
