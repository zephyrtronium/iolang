package iolang

import (
	"strings"
	"sync/atomic"
	"testing"
)

// A SourceTestCase is a test case containing source code and a predicate to
// check the result.
type SourceTestCase struct {
	Source string
	Pass   func(result Interface) bool
}

// TestFunc returns a test function for the test case.
func (c SourceTestCase) TestFunc(name string) func(*testing.T) {
	return func(t *testing.T) {
		msg, err := testVM.Parse(strings.NewReader(c.Source), name)
		if err != nil {
			t.Fatalf("could not parse %q: %v", c.Source, err)
		}
		if err := testVM.OpShuffle(msg); err != nil {
			t.Fatalf("could not opshuffle %q: %v", c.Source, err)
		}
		if r := testVM.DoMessage(msg, testVM.Lobby); !c.Pass(r) {
			t.Errorf("%q produced wrong result; got %T@%p", c.Source, r, r)
		}
	}
}

// PassEqual returns a Pass function for a SourceTestCase that predicates on
// equality.
func PassEqual(want Interface) func(Interface) bool {
	return func(result Interface) bool {
		return want == result
	}
}

// CheckSlots is a testing helper to check whether an object has exactly the
// slots we expect.
func CheckSlots(t *testing.T, obj Interface, slots []string) {
	t.Helper()
	o := obj.SP()
	o.L.Lock()
	defer o.L.Unlock()
	checked := make(map[string]bool, len(slots))
	for _, name := range slots {
		checked[name] = true
		t.Run("Have_"+name, func(t *testing.T) {
			slot, ok := o.Slots[name]
			if !ok {
				t.Fatal("no slot", name)
			}
			if slot == nil {
				t.Fatal("slot", name, "is nil")
			}
		})
	}
	for name := range o.Slots {
		t.Run("Want_"+name, func(t *testing.T) {
			if !checked[name] {
				t.Fatal("unexpected slot", name)
			}
		})
	}
}

// CheckObjectIsProto is a testing helper to check that an object has exactly
// one proto, which is Core Object. obj must come from testVM.
func CheckObjectIsProto(t *testing.T, obj Interface) {
	t.Helper()
	o := obj.SP()
	o.L.Lock()
	defer o.L.Unlock()
	switch len(o.Protos) {
	case 0:
		t.Fatal("no protos")
	case 1: // do nothing
	default:
		t.Error("incorrect number of protos: expected 1, have", len(o.Protos))
	}
	if p := o.Protos[0]; p != testVM.BaseObject {
		t.Errorf("wrong proto: expected %T@%p, have %T@%p", testVM.BaseObject, testVM.BaseObject, p, p)
	}
}

// singleLookupObject is a testing object that panics if its SP method is called
// more than once.
type singleLookupObject struct {
	o *Object
	c uint32
}

func (o *singleLookupObject) SP() *Object {
	if !atomic.CompareAndSwapUint32(&o.c, 0, 1) {
		panic("multiple inspections of a single-lookup object")
	}
	return o.o
}

func (o *singleLookupObject) Activate(vm *VM, target, locals, context Interface, msg *Message) Interface {
	return o.o.Activate(vm, target, locals, context, msg)
}

func (o *singleLookupObject) Clone() Interface {
	return &singleLookupObject{o: o.o.Clone().SP()}
}

func (o *singleLookupObject) isIoObject() {}

// TestGetSlot tests that GetSlot can find local and ancestor slots, and that no
// object is checked more than once.
func TestGetSlot(t *testing.T) {
	sl := &singleLookupObject{o: testVM.Lobby}
	cases := map[string]struct {
		o, v, p Interface
		slot    string
	}{
		"Local":        {testVM.Lobby, testVM.Lobby, testVM.Lobby, "Lobby"},
		"Ancestor":     {testVM.Lobby, testVM.BaseObject, testVM.Core, "Object"},
		"Never":        {testVM.Lobby, nil, nil, "fail to find"},
		"OnceLocal":    {sl, testVM.Lobby, sl, "Lobby"},
		"OnceAncestor": {&singleLookupObject{o: testVM.Lobby}, testVM.BaseObject, testVM.Core, "Object"},
		"OnceNever":    {&singleLookupObject{o: testVM.Lobby}, nil, nil, "fail to find"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, p := GetSlot(c.o, c.slot)
			if v != c.v {
				t.Errorf("slot %s found wrong object: have %T@%p, want %T@%p", c.slot, v, v, c.v, c.v)
			}
			if p != c.p {
				t.Errorf("slot %s found on wrong proto: have %T@%p, want %T@%p", c.slot, p, p, c.p, c.p)
			}
		})
	}
}

// TestGetLocalSlot tests that GetLocalSlot can find local but not ancestor
// slots.
func TestGetLocalSlot(t *testing.T) {
	cases := map[string]struct {
		o, v Interface
		ok   bool
		slot string
	}{
		"Local":    {testVM.Lobby, testVM.Lobby, true, "Lobby"},
		"Ancestor": {testVM.Lobby, nil, false, "Object"},
		"Never":    {testVM.Lobby, nil, false, "fail to find"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, ok := GetLocalSlot(c.o, c.slot)
			if ok != c.ok {
				t.Errorf("slot %s has wrong presence: have %v, want %v", c.slot, ok, c.ok)
			}
			if v != c.v {
				t.Errorf("slot %s found wrong object: have %T@%p, want %T@%p", c.slot, v, v, c.v, c.v)
			}
		})
	}
}

// TestObjectGoActivate tests that an Object set to be activatable activates its
// activate slot when activated.
func TestObjectGoActivate(t *testing.T) {
	o := testVM.ObjectWith(Slots{})
	SetSlot(testVM.Lobby, "TestObjectActivate", o)
	cases := map[string]SourceTestCase{
		"InactiveNoActivate": {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(false)`, PassEqual(o)},
		"InactiveActivate":   {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(false)`, PassEqual(o)},
		"ActiveNoActivate":   {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(true)`, PassEqual(o)},
		"ActiveActivate":     {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(true)`, PassEqual(testVM.Lobby)},
	}
	for name, c := range cases {
		t.Run(name, c.TestFunc("TestObjectActivate/"+name))
	}
	RemoveSlot(testVM.Lobby, "TestObjectActivate")
}

// TestObjectSlots tests that a new VM Object has the slots we expect.
func TestObjectSlots(t *testing.T) {
	slots := []string{
		"",
		"!=",
		"-",
		"..",
		"<",
		"<=",
		"==",
		">",
		">=",
		"?",
		// "@",
		// "@@",
		// "actorProcessQueue",
		// "actorRun",
		"addTrait",
		"ancestorWithSlot",
		"ancestors",
		"and",
		"appendProto",
		"apropos",
		"asBoolean",
		"asGoRepr",
		"asSimpleString",
		"asString",
		"asyncSend",
		"block",
		"break",
		"clone",
		"cloneWithoutInit",
		"compare",
		"contextWithSlot",
		"continue",
		// "coroDo",
		// "coroDoLater",
		// "coroFor",
		// "coroWith",
		// "currentCoro",
		// "deprecatedWarning",
		"do",
		"doFile",
		"doMessage",
		"doRelativeFile",
		"doString",
		"evalArg",
		"evalArgAndReturnNil",
		"evalArgAndReturnSelf",
		"for",
		"foreachSlot",
		"futureSend",
		"getLocalSlot",
		"getSlot",
		// "handleActorException",
		"hasLocalSlot",
		"hasProto",
		"hasSlot",
		"if",
		"ifError",
		"ifNil",
		"ifNilEval",
		"ifNonNil",
		"ifNonNilEval",
		"in",
		// "init",
		// "inlineMethod",
		"isActivatable",
		"isError",
		"isIdenticalTo",
		"isKindOf",
		"isLaunchScript",
		"isNil",
		"isTrue",
		// "launchFile",
		"lazySlot",
		"lexicalDo",
		"list",
		"loop",
		// "memorySize",
		"message",
		"method",
		"newSlot",
		"not",
		"or",
		"pause",
		"perform",
		"performWithArgList",
		"prependProto",
		"print",
		"println",
		"proto",
		"protos",
		"raiseIfError",
		"relativeDoFile",
		"removeAllProtos",
		"removeAllSlots",
		"removeProto",
		"removeSlot",
		"resend",
		"return",
		"returnIfError",
		"returnIfNonNil",
		"setIsActivatable",
		"setProto",
		"setProtos",
		"setSlot",
		"setSlotWithType",
		"shallowCopy",
		"slotDescriptionMap",
		"slotNames",
		"slotSummary",
		"slotValues",
		"super",
		"switch",
		"thisContext",
		"thisLocalContext",
		"thisMessage",
		"try",
		"type",
		"uniqueHexId",
		"uniqueId",
		"updateSlot",
		"wait",
		"while",
		// "write",
		// "writeln",
		"yield",
	}
	CheckSlots(t, testVM.BaseObject, slots)
}
