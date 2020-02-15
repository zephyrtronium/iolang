package iolang

import (
	"testing"
)

// TestNumberCache tests that certain numbers always have identical objects.
func TestNumberCache(t *testing.T) {
	vm := TestingVM()
	// These constants are independent from but (as of writing) equal to those
	// used to construct the number cache. It is allowable for more numbers to
	// be cached, but fewer is not allowable, as there is real(ish?) code that
	// depends on caching of certain numbers.
	const (
		testNumberCacheMin = -10
		testNumberCacheMax = 256
	)
	for i := testNumberCacheMin; i <= testNumberCacheMax; i++ {
		if vm.NewNumber(float64(i)) != vm.NewNumber(float64(i)) {
			t.Error(i, "not cached")
		}
	}
}
