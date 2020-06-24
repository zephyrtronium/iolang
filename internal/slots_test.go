package internal_test

import (
	"sync"
	"testing"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/testutils"
)

// TestProtos tests that an object returns the same list of protos as it is
// created with.
func TestProtos(t *testing.T) {
	vm := testutils.VM()
	cases := map[string][]*iolang.Object{
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
							if i >= len(c) {
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

// TestForeachProto tests that ForeachProto visits each of an object's protos
// exactly once.
func TestForeachProto(t *testing.T) {
	vm := testutils.VM()
	cases := map[string]struct {
		p []*iolang.Object
		e []float64
	}{
		"none": {nil, nil},
		"one":  {[]*iolang.Object{vm.NewNumber(0)}, []float64{0}},
		"ten": {
			[]*iolang.Object{vm.NewNumber(0), vm.NewNumber(1), vm.NewNumber(2), vm.NewNumber(3), vm.NewNumber(4), vm.NewNumber(5), vm.NewNumber(6), vm.NewNumber(7), vm.NewNumber(8), vm.NewNumber(9)},
			[]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			obj := vm.ObjectWith(nil, c.p, nil, nil)
			r := []float64{}
			obj.ForeachProto(func(p *iolang.Object) bool {
				if p.Tag() != iolang.NumberTag {
					t.Errorf("proto is %v, not Number", p.Tag())
					return false
				}
				r = append(r, p.Value.(float64))
				return true
			})
			for i, x := range r {
				if i >= len(c.e) {
					t.Errorf("too many protos; have %v at %d", x, i)
					continue // report every unexpected protos
				}
				if x != c.e[i] {
					t.Errorf("wrong proto value: expected %v, got %v", c.e[i], x)
				}
			}
			if len(r) < len(c.e) {
				t.Errorf("not enough protos; expected %v, have %v (missing %v)", c.e, r, c.e[len(r):])
			}
		})
	}
	t.Run("concurrent", func(t *testing.T) {
		for name, c := range cases {
			c := c // redeclare loop variable
			t.Run(name, func(t *testing.T) {
				wg := sync.WaitGroup{}
				obj := vm.ObjectWith(nil, c.p, nil, nil)
				const n = 128
				wg.Add(n)
				for k := 0; k < n; k++ {
					go func() {
						defer wg.Done()
						r := []float64{}
						obj.ForeachProto(func(p *iolang.Object) bool {
							if p.Tag() != iolang.NumberTag {
								t.Errorf("proto is %v, not Number", p.Tag())
								return false
							}
							r = append(r, p.Value.(float64))
							return true
						})
						for i, x := range r {
							if i >= len(c.e) {
								t.Errorf("too many protos; have %v at %d", x, i)
								continue // report every unexpected proto
							}
							if x != c.e[i] {
								t.Errorf("wrong proto value: expected %v, got %v", c.e[i], x)
							}
						}
						if len(r) < len(c.e) {
							t.Errorf("not enough protos; expected %v, have %v (missing %v)", c.e, r, c.e[len(r):])
						}
					}()
				}
				wg.Wait()
			})
		}
	})
	t.Run("cease", func(t *testing.T) {
		// This test is trivial for zero or one protos, but ranging is easy.
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				n := 0
				obj := vm.ObjectWith(nil, c.p, nil, nil)
				obj.ForeachProto(func(p *iolang.Object) bool {
					n++
					if n != 1 {
						t.Errorf("iterator ran %d times; expected 0 or 1", n)
					}
					return false
				})
			})
		}
	})
}
