Package addons provides easier access to common iolang types and constants.

This package copies some types and constants to make iolang addons much less
verbose. To use it, do `import . "github.com/zephyrtronium/iolang/addons"`.
CFunction definitions can then look like this:

```
func Foo(vm *VM, target, locals *Object, msg *Message) *Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(x, stop)
	}
	// ...
	return foo
}
```

instead of this:

```
func Foo(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	x, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(x, stop)
	}
	// ...
	return foo
}
```

The following iolang types are aliased:

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
- ExitStop

See the Range addon at `iolang/addons/range` for an example of how and why to
use this.
