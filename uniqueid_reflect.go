// +build nounsafe

package iolang

import "reflect"

// The default implementation of UniqueID uses unsafe.Pointer. If you can't use
// packages importing unsafe, you can build with -tags=nounsafe to select this
// implementation instead at a performance penalty in passing messages of about
// 15% to 20%.

// UniqueID returns the object's address.
func (o *Object) UniqueID() uintptr {
	return reflect.ValueOf(o).Pointer()
}
