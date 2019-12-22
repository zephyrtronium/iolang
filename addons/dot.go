// Package addons provides easier access to common iolang types and constants.
//
// This package is intended to be imported using the "import ." construct. It
// uses type aliases and copies some constants to make iolang addons much less
// verbose.
//
// This is the place where evil dwells.
package addons

import "github.com/zephyrtronium/iolang"

type (
	Object = iolang.Object
	Slots  = iolang.Slots

	Stop = iolang.Stop

	VM        = iolang.VM
	Message   = iolang.Message
	Exception = iolang.Exception
)

const (
	NoStop        = iolang.NoStop
	ContinueStop  = iolang.ContinueStop
	BreakStop     = iolang.BreakStop
	ReturnStop    = iolang.ReturnStop
	ExceptionStop = iolang.ExceptionStop
	ExitStop      = iolang.ExitStop
)
