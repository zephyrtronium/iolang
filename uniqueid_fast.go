// +build !nounsafe

package iolang

import "unsafe"

// Using unsafe to retrieve the object's address is about nine times faster
// than using reflect, which gives an overall boost to GetSlot of about 15-20%.

// UniqueID returns the object's address.
func (o *Object) UniqueID() uintptr {
	return uintptr(unsafe.Pointer(o))
}
