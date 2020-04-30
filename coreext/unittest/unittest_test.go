package unittest_test

import (
	"testing"

	_ "github.com/zephyrtronium/iolang/coreext/unittest" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// Directory and File are dependencies.
	slots := []string{
		"Directory",
		"DirectoryCollector",
		"File",
		"FileCollector",
		"RunnerMixIn",
		"TestRunner",
		"UnitTest",
	}
	testutils.CheckNewSlots(t, testutils.VM().Core, slots)
}
