package iolang

import (
	"testing"
)

// TestNumberCache tests that certain numbers always have identical objects.
func TestNumberCache(t *testing.T) {
	vm := TestingVM()
	for i := numberCacheMin; i <= numberCacheMax; i++ {
		if vm.NewNumber(float64(i)) != vm.NewNumber(float64(i)) {
			t.Error(i, "not cached")
		}
	}
}
