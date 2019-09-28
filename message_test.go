package iolang

import (
	"strings"
	"sync/atomic"
	"testing"
)

// TestPerform tests that objects can receive and possibly forward messages to
// activate slots and produce appropriate results.
func TestPerform(t *testing.T) {
	res := &performTester{Object: Object{Protos: []Interface{testVM.BaseObject}}}
	anc := testVM.ObjectWith(Slots{"t": res})
	target := anc.Clone()
	testVM.SetSlot(target, "forward", testVM.NewCFunction(performTestForward, nil))
	tm := testVM.IdentMessage("t")
	cases := map[string]struct {
		o       Interface
		msg     *Message
		succeed bool
		v       Interface
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
			if m := atomic.LoadInt32(&res.activated); m != n {
				t.Errorf("wrong activation count: want %d, have %d", n, m)
			}
			atomic.StoreInt32(&res.activated, 0)
		})
	}
}

type performTester struct {
	Object
	activated int32
}

func (p *performTester) Clone() Interface {
	return &performTester{Object: Object{Protos: []Interface{p}}}
}

func (p *performTester) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	atomic.AddInt32(&p.activated, 1)
	return p.Object.Activate(vm, target, locals, context, msg)
}

func performTestForward(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	nn := strings.ToLower(msg.Name())
	if v, proto := vm.GetSlot(target, nn); proto != nil {
		return v.Activate(vm, target, locals, proto, vm.IdentMessage(nn))
	}
	return vm.RaiseExceptionf("%s does not respond to %s", vm.TypeName(target), msg.Name())
}
