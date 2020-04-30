package file_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/file" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// Date is a dependency.
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Date", "File"})
}
