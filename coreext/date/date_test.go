package date_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/date" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// Duration is a dependency.
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Date", "Duration"})
}
