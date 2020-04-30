package directory_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/directory" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// File is a dependency.
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Directory", "File"})
}
