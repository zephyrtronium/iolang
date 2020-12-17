package internal_test

import (
	"testing"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/testutils"
)

func BenchmarkMethodActivate(b *testing.B) {
	vm := testutils.VM()
	m := vm.MustDoString(`method(nil)`)
	if m.Tag() != iolang.BlockTag {
		b.Fatalf("method is %v, not Block", m.Tag())
	}
	msg := vm.IdentMessage("f")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Activate(vm, vm.Lobby, vm.Lobby, vm.Lobby, msg)
	}
}
