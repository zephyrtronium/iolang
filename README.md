# iolang

This is a pure Go implementation of [Io](http://iolanguage.org/). Much of the hard work has been done, but hard work still remains.

To embed an iolang interpreter into a Go program, one would use the `NewVM()` function to get a `*VM`, use `SetSlot()` to make available any extras, then feed the VM code to evaluate with its `DoString()` and `DoReader()` methods. The VM also has methods like `NewNumber()`, `NewString()`, `ObjectWith()`, &c. to create primitives. The API is currently incomplete and will change.

The `io` directory contains a REPL as an example of embedding iolang.

## Io Primer

`"Hello, world!" println`

Io is a programming language primarily inspired by Self, Smalltalk, and Lisp. Execution is implemented by message passing: the phrase `Object slot`, when evaluated, creates a message with the text `slot` and sends it to `Object`, which is itself a message passed to the default `Lobby` object. Messages can also be provided with arguments, which are simply additional messages to provide more information with the message, in parentheses. `Object slot(arg1, arg2)` provides the `arg1` and `arg2` messages as arguments to the call. The only syntactic elements other than messages are numeric and string literals and comments.

When an object receives a message, it checks for a slot on the object with the same name as the message text. If the object has such a slot, then it is activated and produces a value. Slots can be added to objects using the `:=` or `::=` assignment operators; `Object x := 1` creates a slot on `Object` named `x` that will produce the value 1 when activated.

Certain objects have special behavior when activated. Methods are encapsulated messages that, when activated, send that message to the receiver of the message which activated the method. For example, with `println` being an Object method, `x println` executes `println`'s message in the context of `x`, meaning that the default location for slot lookups becomes `x` instead of `Lobby`. Methods also have their own locals, so new slots created within them exist only inside the method body, and they can take arguments, setting slots on the locals object for each argument.

If an object does not own a slot with the same name as a message passed to it, the object will instead check for that slot in its prototypes. Objects can have any number of protos, and the search proceeds in depth-first order without duplicates. If the slot is found in one of the protos, then it is activated, but still with the original object as the receiver. This implements an inheritance-like concept, but with "superclasses" being themselves objects. (If there isn't any proto with the slot, then the message is sent to the object's `forward` slot, and if that slot doesn't exist, either, then an exception is raised.)

Producing "subclasses" is done using the `clone` method. A clone of an object, say `x`, is another object which has (initially) empty slots and `x` as its only prototype. If we say `y := x clone`, then `y` will respond to the same messages as `x` via delegation; new slots can be created which will be activated in place of the proto's, providing polymorphism. The typical pattern to implement a new "type" is to say `NewType := Object clone do(x := y; z := w)`, where `do` is an `Object` method which evaluates code in the context of its receiver. Then, "instances" of the new "class" can be created using `instance := NewType clone`. Notice that `clone` is the method used both to create a new type and an instance of that type - this is prototype-based programming.

An important aspect of Io lacking syntax beyond messages is that control flow is implemented as `Object` methods. `if` is a method taking one to three arguments: `if(cond, message when cond is true, message when cond is false)` evaluates its first argument, then the second if it is true or the third if it is false. Because message arguments are themselves messages, the other argument is not evaluated. When any of the arguments are not supplied, the evaluation result is returned instead, which enables alternate forms of branching: `if(cond) then(message when cond is true) elseif(cond2, different message) else(last thing to try)`. There are also loops, including `for` to loop over a range with a counter, `while` to perform a loop as long as a condition is true, `loop` to perform a loop forever, and others. `continue`, `break`, and `return` generally do what you expect. Each of these methods is an expression, and the last value produced during evaluation of each is returned.

For a more in-depth introduction to Io, check out [the official guide](iolanguage.org/guide/guide.html) and [reference](iolanguage.org/reference/index.html). There are more code examples at [the original implementation's GitHub repository](https://github.com/IoLanguage/io) as well.

## TODO

- Implement primitive (Core) types:
	+ Compiler
- Concurrency; coroutines, futures, promises, actors, &c.
	+ Core Coroutine
		* Call coroutine
	+ Core Scheduler
- Figure out whether Calls really need slotContext, because it seems like it's always the same as Call sender.
- Finish implementing CFunctions for existing primitive types:
	+ Object
	+ Sequence
	+ Exception
	+ File - figure out how/whether to implement popen and reopen.
	+ Number
	+ Date - fromString requires a robust implementation.
- Write initialization code/Io methods for all types.
	+ Create Error type.
	+ Lots to do for most Core types.
- Write tests, both in Go and in Io.
- Importer, and implement Addons, ideally supporting Go's `-buildmode=plugin`.
- Possibly turn Stop into a real object.
- Document differences between this implementation and the original.
