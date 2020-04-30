package path_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/path" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Path"})
}
