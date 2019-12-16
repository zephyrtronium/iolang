# testutils

[![GoDoc](https://godoc.org/github.com/zephyrtronium/iolang/testutils?status.svg)](https://godoc.org/github.com/zephyrtronium/iolang/testutils)

Package testutils provides utilities for testing Io code in Go:

- `TestingVM` provides synchronized access to a VM for executing code. Also,
    `ResetTestingVM` restores that VM to a fresh, clean state.
- `SourceTestCase` creates sub-test functions that execute code and check the
    result using one of various predicates (which `Pass*` functions provide).
- `CheckSlots` is a helper that checks that an object has exactly the expected
    slots.
- `CheckObjectIsProto` is a helper that checks that an object has `Core Object`
    as its only proto.

The contents of this package are copied from internal testing code. To read
example usage, see e.g. `object_test.go` in the base iolang package.
