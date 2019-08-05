Package addons provides easier access to common iolang types and constants.

This package copies some types and constants to make iolang addons much less
verbose. To use it, do `import . "github.com/zephyrtronium/iolang/addons"`.
CFunction definitions can then look like this:

```
func Foo(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return x, stop
	}
	// ...
	return foo, NoStop
}
```

instead of this:

```
func Foo(vm *iolang.VM, target, locals iolang.Interface, msg *iolang.Message) (iolang.Interface, iolang.Stop) {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return x, stop
	}
	// ...
	return foo, iolang.NoStop
}
```

The following iolang types are aliased:

- Interface
- Object
- Slots
- Stop
- VM
- Message
- Exception

Additionally, the Stop constants for serial control flow are copied:

- NoStop
- ContinueStop
- BreakStop
- ReturnStop
- ExceptionStop

See the Range addon at `iolang/addons/range` for an example of how and why to
use this.
