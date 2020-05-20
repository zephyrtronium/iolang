package internal

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestProtoHead tests that the internal means to synchronize accesses to an
// object's first proto prevents inconsistent behavior.
func TestProtoHead(t *testing.T) {
	vm := NewVM() // Not testutils.VM; that would cause an import cycle.
	cases := map[string]func(t *testing.T){
		"uncontested": func(t *testing.T) {
			obj := vm.ObjectWith(nil, []*Object{vm.BaseObject}, nil, nil)
			r := obj.protoHead()
			if r != vm.BaseObject {
				t.Errorf("wrong protoHead: wanted %v, got %v", vm.BaseObject, r)
			}
		},
		"contested": func(t *testing.T) {
			// logicalDeleted is a special value that indicates an object's
			// protos list is being modified.
			obj := vm.ObjectWith(nil, []*Object{logicalDeleted}, nil, nil)
			obj.protos.mu.Lock()
			ch := make(chan *Object)
			sig := new(Object)
			go func() {
				ch <- sig // Signal that this goroutine is running.
				ch <- obj.protoHead()
				close(ch) // Avoid hanging if protoHead succeeds immediately.
			}()
			<-ch // Wait for the new goroutine to start.
			select {
			case r := <-ch:
				t.Errorf("unexpected send of proto head: (logicalDeleted is @%p,) got %[2]v@%[2]p", logicalDeleted, r)
			case <-time.After(30 * time.Millisecond):
			}
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&obj.protos.p)), unsafe.Pointer(vm.BaseObject))
			obj.protos.mu.Unlock()
			switch r := <-ch; r {
			case sig:
				t.Log("protoHead may not have been contested during 30 ms sleep")
			case vm.BaseObject: // success; do nothing
			default:
				t.Errorf("wrong proto: expected %[1]v@%[1]p, (logicalDeleted is @%[3]p,) got %[2]v@%[2]v", vm.BaseObject, r, logicalDeleted)
			}
		},
	}
	for name, c := range cases {
		t.Run(name, c)
	}
}

// TestProtos tests that an object returns the same list of protos as it is
// created with.
func TestProtos(t *testing.T) {
	vm := NewVM()
	cases := map[string][]*Object{
		"none": nil,
		"one":  {vm.NewObject(nil)},
		"ten":  {vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil), vm.NewObject(nil)},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			obj := vm.ObjectWith(nil, c, nil, nil)
			p := obj.Protos()
			for i, v := range p {
				if i > len(c) {
					t.Errorf("too many protos: have %v at %d", v, i)
					continue // report every unexpected proto
				}
				if v != c[i] {
					t.Errorf("wrong proto at %d: want %v, have %v", i, c[i], v)
				}
			}
			for i := len(p); i < len(c); i++ {
				t.Errorf("too few protos: missing %v at %d", c[i], i)
			}
		})
	}
	t.Run("concurrent", func(t *testing.T) {
		for name, c := range cases {
			c := c // redeclare loop variable
			t.Run(name, func(t *testing.T) {
				wg := sync.WaitGroup{}
				obj := vm.ObjectWith(nil, c, nil, nil)
				const n = 128
				wg.Add(n)
				for k := 0; k < n; k++ {
					go func() {
						defer wg.Done()
						p := obj.Protos()
						for i, v := range p {
							if i > len(c) {
								t.Errorf("too many protos: have %v at %d", v, i)
								continue // report every unexpected proto
							}
							if v != c[i] {
								t.Errorf("wrong proto at %d: want %v, have %v", i, c[i], v)
							}
						}
						for i := len(p); i < len(c); i++ {
							t.Errorf("too few protos: missing %v at %d", c[i], i)
						}
					}()
				}
				wg.Wait()
			})
		}
	})
}
