/*
Package iolang implements a version of the Io programming language.

Io is a dynamic, prototype-based language inspired by Smalltalk (everything is
an object), Self (prototypes - *everything* is an object), Lisp (even the
program code is made up of objects), NewtonScript (differential inheritance),
Act1 (concurrency via actors and futures), and Lua (small and embeddable
implementations). It was originally developed in C by Steve Dekorte.

Currently, this implementation is focusing primarily on becoming a fully
fledged interpreter. A read-eval-print loop exists, primarily for integration
testing. There is no file interpreter, so written Io programs are not yet
executable.

The interpreter can easily be embedded in another program. To start, use the
NewVM function to create and initialize the interpreter. The VM object has a
number of fields that then can be used to make objects available to Io code,
especially the Lobby and Addons objects. Use the VM's NewNumber, NewString, or
any other object creation methods to create the object, then use SetSlot to set
those objects' slots to them.

Io Primer

Hello World in Io:

	"Hello, world!" println

Io code executes via message passing. For example, the above snippet compiles
to two messages:

	1. "Hello, world!", a string literal
	2. println

Upon evaluation, the first message, being a literal, immediately becomes that
string value. Then, the println message is sent to it, activating the slot on
the string object named "println", which is a method that writes the object to
standard output.

Messages can also be provided with arguments, which are simply additional
messages to provide more information with the message, in parentheses. We could
add arguments to our program:

	"Hello, world!"(an argument) println(another argument)

But the program executes the same way, because neither string literals nor (the
default) println use their arguments.

When an object receives a message, it checks for a slot on the object with the
same name as the message text. If the object has such a slot, then it is
activated and produces a value. Slots can be added to objects using the :=
assignment operator; executing

	Object x := 1

creates a slot on Object named x with the value 1.

Certain objects have special behavior when activated. Methods are encapsulated
messages that, when activated, send that message to the object which received
the message that activated the method. The println method above is an example:
it is defined (roughly) as

	Object println := method(File standardOutput write(self, "\n"); self)

In the Hello World example, this method is activated with the "Hello, world!"
string as its receiver, so it executes in the context of the string, meaning
that slot lookups are performed on the string, and the "self" slot within the
method refers to the string. Methods have their own locals, so new slots
created within them exist only inside the method body. They can also be defined
to take arguments; for example, if the println method were instead defined as

	Object println := method(x, File standardOutput write(x, "\n"); self)

then we would pass the object to print as an argument instead of printing the
receiver.

If an object does not own a slot with the same name as a message passed to it,
the object will instead check for that slot in its prototypes. Objects can have
any number of protos, and the search proceeds in depth-first order without
duplicates. If the slot is found in one of the protos, then it is activated,
but still with the original object as the receiver. This implements an
inheritance-like concept, but with "superclasses" being themselves objects. (If
there isn't any proto with the slot, then the message is sent to the object's
"forward" slot, and if that slot doesn't exist, either, an exception is
raised.)

Producing "subclasses" is done using the clone method. A clone of an object
is a new object having empty slots and that object as its only prototype. If we
say "y := x clone", then y will respond to the same messages as x via
delegation; new slots can be created which will be activated in place of the
proto's, providing polymorphism. The typical pattern to implement a new "type"
is to say:

	NewType := Object clone do(
		x := y
		z := w
		someMethod := method(...)
	)

do is an Object method which evaluates code in the context of its receiver.
"Instances" of the new "class" can then be created by saying:

	instance := NewType clone

Now instance is an object which responds to NewType's x, z, and someMethod
slots, as well as all Object slots. Notice that clone is the method used both
to create a new type and an instance of that type - this is prototype-based
programming.

An important aspect of Io lacking syntax beyond messages is that control flow
is implemented as Object methods. "Object if" is a method taking one to three
arguments:

	if(condition, message when true, message when false)
	if(condition, message when true) else(message when false)
	if(condition) then(message when true) else(message when false)

All three of these forms are essentially equivalent. if evaluates the
condition, then if it is true (controlled by the result's asBoolean slot), it
evaluates the second, otherwise it evaluates the third. If the branch to
evaluate doesn't exist, if instead returns the boolean to enable the other
forms, as well as to support the elseif method. Note that because message
arguments are themselves messages, the wrong branches are never evaluated, so
their side effects don't happen.

There are also loops, including but not limited to:

	repeat(message to loop forever)
	while(condition, message to loop)
	for(counter, start, stop, message to loop)
	5 repeat(message to do 5 times)

Each loop, being a method, produces a value, which is by default the last value
encountered during evaluation of the loop. Object methods continue, break, and
return also do what their equivalents in most other programming languages do,
except that continue and break can be supplied an argument to change the loop
result.

For a more in-depth introduction to Io, check out the official guide and
reference at http://iolanguage.org/ as well as the original implementation's
GitHub repository at https://github.com/IoLanguage/io for code samples.
*/
package iolang

import (
	"github.com/zephyrtronium/iolang/internal"
)

// A VM processes Io programs.
type VM = internal.VM

// Object is the basic type of Io. Everything is an Object.
//
// Always use NewObject, ObjectWith, or a type-specific constructor to obtain
// new objects. Creating objects directly will result in arbitrary failures.
type Object = internal.Object

// Slots represents the set of messages to which an object responds.
type Slots = internal.Slots

// Tag is a type indicator for iolang objects. Tag values must be comparable.
// Tags for different types must not be equal, meaning they must have different
// underlying types or different values otherwise.
type Tag = internal.Tag

// BasicTag is a special Tag type for basic primitive types which do not have
// special activation and whose clones have values that are shallow copies of
// their parents.
type BasicTag = internal.BasicTag

// A Stop represents a reason for flow control.
type Stop = internal.Stop

// RemoteStop is a wrapped object and control flow status for sending to coros.
type RemoteStop = internal.RemoteStop

// A Block is a reusable, lexically scoped message. Essentially a function.
//
// NOTE: Unlike most other primitives in iolang, Block values are NOT
// synchronized. It is a race condition to modify a block that might be in use,
// such as 'call activated' or any block or method object in a scope other than
// the locals of the innermost currently executing block.
type Block = internal.Block

// Call wraps information about the activation of a Block.
type Call = internal.Call

// A CFunction is an object whose value represents a compiled function.
type CFunction = internal.CFunction

// An Exception is an Io exception.
type Exception = internal.Exception

// A Message is the fundamental syntactic element and functionality of Io.
//
// NOTE: Unlike most other primitive types in iolang, Message values are NOT
// synchronized. It is a race condition to modify a message that might be in
// use, such as 'call message' or any message object in a scope other than the
// locals of the innermost currently executing block.
type Message = internal.Message

// Scheduler helps manage a group of Io coroutines.
type Scheduler = internal.Scheduler

// A Sequence is a collection of data of one fixed-size type.
type Sequence = internal.Sequence

// An Fn is a statically compiled function which can be executed in an Io VM.
type Fn = internal.Fn

// SeqKind represents a sequence data type.
type SeqKind = internal.SeqKind

// Tag variables for core types.
var (
	BlockTag     = internal.BlockTag
	CallTag      = internal.CallTag
	CFunctionTag = internal.CFunctionTag
	ExceptionTag = internal.ExceptionTag
	ListTag      = internal.ListTag
	MapTag       = internal.MapTag
	MessageTag   = internal.MessageTag
	SequenceTag  = internal.SequenceTag
)

// Tag constants for core types.
const (
	NumberTag    = internal.NumberTag
	SchedulerTag = internal.SchedulerTag
)

// Control flow reasons.
const (
	NoStop        = internal.NoStop
	ContinueStop  = internal.ContinueStop
	BreakStop     = internal.BreakStop
	ReturnStop    = internal.ReturnStop
	ExceptionStop = internal.ExceptionStop
	ExitStop      = internal.ExitStop
)

// SeqMaxItemSize is the maximum size in bytes of a single sequence element.
const SeqMaxItemSize = internal.SeqMaxItemSize

// NewVM prepares a new VM to interpret Io code. String arguments may be passed
// to occupy the System args slot, typically os.Args[1:].
func NewVM(args ...string) *VM {
	return internal.NewVM(args...)
}

// ForeachArgs gets the arguments for a foreach method utilizing the standard
// foreach([[key,] value,] message) syntax.
func ForeachArgs(msg *Message) (kn, vn string, hkn, hvn bool, ev *Message) {
	return internal.ForeachArgs(msg)
}

// SliceArgs gets start, stop, and step values for a standard slice-like
// method invocation, which may be any of the following:
//
// 	slice(start)
// 	slice(start, stop)
// 	slice(start, stop, step)
//
// start and stop are fixed in the following sense: for each, if it is less
// than zero, then size is added to it, then, if it is still less than zero,
// it becomes -1 if the step is negative and 0 otherwise; if it is greater than
// or equal to the size, then it becomes size - 1 if step is negative and size
// otherwise.
func SliceArgs(vm *VM, locals *Object, msg *Message, size int) (start, step, stop int, exc *Object, control Stop) {
	return internal.SliceArgs(vm, locals, msg, size)
}
