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
condition, then if it is true (controlled by the result's isTrue slot), it
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

// IoVersion is the interpreter version, used for the System version slot. It
// bears no relation to versions of the original implementation.
const IoVersion = "1"

// IoSpecVer is the Io language version, used for the System iospecVersion
// slot.
const IoSpecVer = "0.0.0"
