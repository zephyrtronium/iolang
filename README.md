# iolang

This is an attempt at a pure Go implementation of [Io](http://iolanguage.org/) with a focus on embedding into Go programs. Much of the hard work has been done, but hard work still remains.

The current way to embed an iolang interpreter in a Go program involves using the `NewVM()` function to get a `*VM` and using `SetSlot()` to make available any extras, then feeding the VM things to evaluate with its `DoString()` and `DoReader()` methods. The VM also has methods like `NewNumber()`, `NewString()`, `ObjectWith()`, &c. to create primitives. The API is currently incomplete and will change.

The `io` directory contains an interactive interpreter as an example of embedding iolang.
