package iolang

import (
	"strings"
	"sync/atomic"
	"testing"
)

// TestPerform tests that objects can receive and possibly forward messages to
// activate slots and produce appropriate results.
func TestPerform(t *testing.T) {
	pt := &performTester{obj: testVM.NewObject(nil)}
	res := testVM.ObjectWith(nil, []*Object{testVM.BaseObject}, pt, performTesterTag{})
	anc := testVM.NewObject(Slots{"t": res})
	target := anc.Clone()
	target.SetSlot("forward", testVM.NewCFunction(performTestForward, nil))
	tm := testVM.IdentMessage("t")
	cases := map[string]struct {
		o       *Object
		msg     *Message
		succeed bool
		v       *Object
	}{
		"Local":       {anc, tm, true, res},
		"Ancestor":    {target, tm, true, res},
		"Forward":     {target, testVM.IdentMessage("T"), true, res},
		"Fail":        {anc, testVM.IdentMessage("u"), false, nil},
		"ForwardFail": {target, testVM.IdentMessage("u"), false, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, stop := testVM.Perform(c.o, c.o, c.msg)
			var n int32
			if c.succeed {
				if stop != NoStop {
					t.Errorf("wrong control flow: want %v (NoStop), got %v (%v)", c.v, r, stop)
				}
				n = 1
			} else {
				if stop != ExceptionStop {
					t.Errorf("wrong control flow: want <anything> (ExceptionStop), got %v (%v)", r, stop)
				}
			}
			if m := atomic.LoadInt32(&pt.act); m != n {
				t.Errorf("wrong activation count: want %d, have %d", n, m)
			}
			atomic.StoreInt32(&pt.act, 0)
		})
	}
}

type performTester struct {
	act int32
	obj *Object
}

type performTesterTag struct{}

func (performTesterTag) Activate(vm *VM, self, target, locals, context *Object, msg *Message) *Object {
	v := self.Value.(*performTester)
	atomic.AddInt32(&v.act, 1)
	return v.obj.Activate(vm, target, locals, context, msg)
}

func (performTesterTag) CloneValue(value interface{}) interface{} {
	return &performTester{obj: value.(*performTester).obj.Clone()}
}

func (performTesterTag) String() string {
	return "performTester"
}

func performTestForward(vm *VM, target, locals *Object, msg *Message) *Object {
	nn := strings.ToLower(msg.Name())
	if v, proto := target.GetSlot(nn); proto != nil {
		return v.Activate(vm, target, locals, proto, vm.IdentMessage(nn))
	}
	return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), msg.Name())
}

func BenchmarkPerform(b *testing.B) {
	o := testVM.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	p := testVM.BaseObject.Clone()
	nm := testVM.IdentMessage("type")
	cm := testVM.IdentMessage("thisContext")
	cases := map[string]*Object{
		"Local":    testVM.BaseObject,
		"Proto":    p,
		"Ancestor": o,
	}
	// o has the deepest search depth, so it will reserve the most space in
	// vm.protoSet and vm.protoStack. Performing once here ensures that results
	// are consistent within the actual benchmark.
	testVM.Perform(o, o, nm)
	for name, o := range cases {
		b.Run(name, func(b *testing.B) {
			b.Run("Type", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					BenchDummy, _ = testVM.Perform(o, o, nm)
				}
			})
			b.Run("ThisContext", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					BenchDummy, _ = testVM.Perform(o, o, cm)
				}
			})
		})
	}
}

// TestPerformNilResult tests that VM.Perform always converts nil to VM.Nil.
func TestPerformNilResult(t *testing.T) {
	cf := testVM.NewCFunction(nilResult, nil)
	o := testVM.NewObject(Slots{
		"f":       cf,
		"forward": cf,
	})
	cases := map[string]struct {
		o   *Object
		msg *Message
	}{
		"HaveSlot": {o, testVM.IdentMessage("f")},
		"Forward":  {o, testVM.IdentMessage("g")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, stop := testVM.Perform(c.o, c.o, c.msg)
			if r != testVM.Nil {
				t.Errorf("result not VM.Nil: want %T@%p, got %T@%p", testVM.Nil, testVM.Nil, r, r)
			}
			if stop != NoStop {
				t.Errorf("wrong control flow: want NoStop, got %v", stop)
			}
		})
	}
}

func nilResult(vm *VM, target, locals *Object, msg *Message) *Object {
	return nil
}
