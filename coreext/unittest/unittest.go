//go:generate go run ../../cmd/gencore unittest_init.go unittest ./io
//go:generate gofmt -s -w unittest_init.go

package unittest

import (
	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"

	// importing for side effects
	_ "github.com/zephyrtronium/iolang/coreext/directory"
	_ "github.com/zephyrtronium/iolang/coreext/file"
)

func init() {
	internal.Register(initUnitTest)
}

func initUnitTest(vm *iolang.VM) {
	internal.Ioz(vm, coreIo, coreFiles)
}
