package iolang

import (
	"strings"
	"sync/atomic"
	"testing"
)

// TestPerform tests that objects can receive and possibly forward messages to
// activate slots and produce appropriate results.
func TestPerform(t *testing.T) {
	vm := TestingVM()
	pt := &performTester{obj: vm.NewObject(nil)}
	res := vm.ObjectWith(nil, []*Object{vm.BaseObject}, pt, performTesterTag{})
	anc := vm.NewObject(Slots{"t": res})
	target := anc.Clone()
	vm.SetSlot(target, "forward", vm.NewCFunction(performTestForward, nil))
	tm := vm.IdentMessage("t")
	cases := map[string]struct {
		o       *Object
		msg     *Message
		succeed bool
		v       *Object
	}{
		"Local":       {anc, tm, true, res},
		"Ancestor":    {target, tm, true, res},
		"Forward":     {target, vm.IdentMessage("T"), true, res},
		"Fail":        {anc, vm.IdentMessage("u"), false, nil},
		"ForwardFail": {target, vm.IdentMessage("u"), false, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, stop := vm.Perform(c.o, c.o, c.msg)
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
	if v, proto := vm.GetSlot(target, nn); proto != nil {
		return v.Activate(vm, target, locals, proto, vm.IdentMessage(nn))
	}
	return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), msg.Name())
}

func BenchmarkPerform(b *testing.B) {
	vm := TestingVM()
	o := vm.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	p := vm.BaseObject.Clone()
	nm := vm.IdentMessage("type")
	cm := vm.IdentMessage("thisContext")
	am := vm.IdentMessage("clone")
	cases := map[string]*Object{
		"Local":    vm.BaseObject,
		"Proto":    p,
		"Ancestor": o,
	}
	// o has the deepest search depth, so it will reserve the most space in
	// vm.protoSet and vm.protoStack. Performing once here ensures that results
	// are consistent within the actual benchmark.
	vm.Perform(o, o, nm)
	for name, o := range cases {
		b.Run(name, func(b *testing.B) {
			b.Run("Type", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					BenchDummy, _ = vm.Perform(o, o, nm)
				}
			})
			b.Run("ThisContext", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					BenchDummy, _ = vm.Perform(o, o, cm)
				}
			})
			b.Run("Clone", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					BenchDummy, _ = vm.Perform(o, o, am)
				}
			})
		})
	}
}

func BenchmarkPerformParallel(b *testing.B) {
	vm := TestingVM()
	o := vm.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	p := vm.BaseObject.Clone()
	m := vm.IdentMessage("type")
	cases := map[string]*Object{
		"Local":    vm.BaseObject,
		"Proto":    p,
		"Ancestor": o,
	}
	for name, o := range cases {
		b.Run(name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				// Make a new coroutine to execute messages.
				coro := vm.VMFor(vm.Coro.Clone())
				for pb.Next() {
					BenchDummy, _ = coro.Perform(o, o, m)
				}
			})
		})
	}
}

func BenchmarkEvalParallel(b *testing.B) {
	vm := TestingVM()
	m, err := vm.Parse(strings.NewReader(`benchmarkValue := benchmarkValue + 1`), "BenchmarkPerformParallel")
	if err != nil {
		panic(err)
	}
	vm.SetSlot(vm.Lobby, "benchmarkValue", vm.NewNumber(0))
	defer vm.RemoveSlot(vm.Lobby, "benchmarkValue")
	b.RunParallel(func(pb *testing.PB) {
		coro := vm.VMFor(vm.Coro.Clone())
		for pb.Next() {
			BenchDummy, _ = m.Eval(coro, coro.Lobby)
		}
	})
}

// TestPerformNilResult tests that VM.Perform always converts nil to VM.Nil.
func TestPerformNilResult(t *testing.T) {
	vm := TestingVM()
	cf := vm.NewCFunction(nilResult, nil)
	o := vm.NewObject(Slots{
		"f":       cf,
		"forward": cf,
	})
	cases := map[string]struct {
		o   *Object
		msg *Message
	}{
		"HaveSlot": {o, vm.IdentMessage("f")},
		"Forward":  {o, vm.IdentMessage("g")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, stop := vm.Perform(c.o, c.o, c.msg)
			if r != vm.Nil {
				t.Errorf("result not VM.Nil: want %T@%p, got %T@%p", vm.Nil, vm.Nil, r, r)
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
