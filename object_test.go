package iolang

import (
	"strings"
	"testing"
)

// A SourceTestCase is a test case containing source code and a predicate to
// check the result.
type SourceTestCase struct {
	Source string
	Pass   func(result *Object, control Stop) bool
}

// TestFunc returns a test function for the test case.
func (c SourceTestCase) TestFunc(name string) func(*testing.T) {
	return func(t *testing.T) {
		msg, err := testVM.ParseScanner(strings.NewReader(c.Source), name)
		if err != nil {
			t.Fatalf("could not parse %q: %v", c.Source, err)
		}
		if err := testVM.OpShuffle(testVM.MessageObject(msg)); err != nil {
			t.Fatalf("could not opshuffle %q: %v", c.Source, err)
		}
		if r, s := testVM.DoMessage(msg, testVM.Lobby); !c.Pass(r, s) {
			t.Errorf("%q produced wrong result; got %s@%p (%s)", c.Source, testVM.AsString(r), r, s)
		}
	}
}

// PassEqual returns a Pass function for a SourceTestCase that predicates on
// equality. To determine equality, this first checks for equal identities; if
// not, it checks that the result of testVM.Compare(want, result) is 0.
func PassEqual(want *Object) func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		if control != NoStop {
			return false
		}
		if want == result {
			return true
		}
		v, stop := testVM.Compare(want, result)
		if stop != NoStop {
			return false
		}
		n, ok := v.Value.(float64)
		if !ok {
			return false
		}
		return n == 0
	}
}

// PassIdentical returns a Pass function for a SourceTestCase that predicates
// on identity equality.
func PassIdentical(want *Object) func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		if control != NoStop {
			return false
		}
		return want == result
	}
}

// PassControl returns a Pass function for a SourceTestCase that predicates on
// equality with a certain control flow status. The control flow check precedes
// the value check. Equality here has the same semantics as in PassEqual.
func PassControl(want *Object, stop Stop) func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		if control != stop {
			return false
		}
		if want == result {
			return true
		}
		v, stop := testVM.Compare(want, result)
		if stop != NoStop {
			return false
		}
		n, ok := v.Value.(float64)
		if !ok {
			return false
		}
		return n == 0
	}
}

// PassTag returns a Pass function for a SourceTestCase that predicates on
// equality of the Tag of the result.
func PassTag(want Tag) func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		if control != NoStop {
			return false
		}
		return result.Tag() == want
	}
}

// PassFailure returns a Pass function for a SourceTestCase that returns true
// iff the result is a raised exception.
func PassFailure() func(*Object, Stop) bool {
	// This doesn't need to be a function returning a function, but it's nice to
	// stay consistent with the other predicate generators.
	return func(result *Object, control Stop) bool {
		return control == ExceptionStop
	}
}

// PassSuccess returns a Pass function for a SourceTestCase that returns true
// iff the control flow status is NoStop.
func PassSuccess() func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		return control == NoStop
	}
}

// PassLocalSlots returns a Pass function for a SourceTestCase that returns
// true iff the result locally has all of the slots in want and none of the
// slots in exclude.
func PassLocalSlots(want, exclude []string) func(*Object, Stop) bool {
	return func(result *Object, control Stop) bool {
		if control != NoStop {
			return false
		}
		for _, slot := range want {
			if _, ok := result.GetLocalSlot(slot); !ok {
				return false
			}
		}
		for _, slot := range exclude {
			if _, ok := result.GetLocalSlot(slot); ok {
				return false
			}
		}
		return true
	}
}

// CheckSlots is a testing helper to check whether an object has exactly the
// slots we expect.
func CheckSlots(t *testing.T, obj *Object, slots []string) {
	t.Helper()
	obj.Lock()
	defer obj.Unlock()
	checked := make(map[string]bool, len(slots))
	for _, name := range slots {
		checked[name] = true
		t.Run("Have_"+name, func(t *testing.T) {
			slot, ok := obj.Slots[name]
			if !ok {
				t.Fatal("no slot", name)
			}
			if slot == nil {
				t.Fatal("slot", name, "is nil")
			}
		})
	}
	for name := range obj.Slots {
		t.Run("Want_"+name, func(t *testing.T) {
			if !checked[name] {
				t.Fatal("unexpected slot", name)
			}
		})
	}
}

// CheckObjectIsProto is a testing helper to check that an object has exactly
// one proto, which is Core Object. obj must come from testVM.
func CheckObjectIsProto(t *testing.T, obj *Object) {
	t.Helper()
	obj.Lock()
	defer obj.Unlock()
	switch len(obj.Protos) {
	case 0:
		t.Fatal("no protos")
	case 1: // do nothing
	default:
		t.Error("incorrect number of protos: expected 1, have", len(obj.Protos))
	}
	if p := obj.Protos[0]; p != testVM.BaseObject {
		t.Errorf("wrong proto: expected %T@%p, have %T@%p", testVM.BaseObject, testVM.BaseObject, p, p)
	}
}

// TestGetSlot tests that GetSlot can find local and ancestor slots, and that no
// object is checked more than once.
func TestGetSlot(t *testing.T) {
	cases := map[string]struct {
		o, v, p *Object
		slot    string
	}{
		"Local":    {testVM.Lobby, testVM.Lobby, testVM.Lobby, "Lobby"},
		"Ancestor": {testVM.Lobby, testVM.BaseObject, testVM.Core, "Object"},
		"Never":    {testVM.Lobby, nil, nil, "fail to find"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, p := c.o.GetSlot(c.slot)
			if v != c.v {
				t.Errorf("slot %s found wrong object: have %T@%p, want %T@%p", c.slot, v, v, c.v, c.v)
			}
			if p != c.p {
				t.Errorf("slot %s found on wrong proto: have %T@%p, want %T@%p", c.slot, p, p, c.p, c.p)
			}
		})
	}
}

// BenchmarkGetSlot benchmarks VM.GetSlot in various depths of search.
func BenchmarkGetSlot(b *testing.B) {
	o := testVM.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	cases := map[string]struct {
		o    *Object
		slot string
	}{
		"Local":    {testVM.Lobby, "Lobby"},
		"Proto":    {testVM.BaseObject, "Lobby"},
		"Ancestor": {o, "Lobby"},
	}
	for name, c := range cases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchDummy, _ = c.o.GetSlot(c.slot)
			}
		})
	}
}

// TestGetLocalSlot tests that GetLocalSlot can find local but not ancestor
// slots.
func TestGetLocalSlot(t *testing.T) {
	cases := map[string]struct {
		o, v *Object
		ok   bool
		slot string
	}{
		"Local":    {testVM.Lobby, testVM.Lobby, true, "Lobby"},
		"Ancestor": {testVM.Lobby, nil, false, "Object"},
		"Never":    {testVM.Lobby, nil, false, "fail to find"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, ok := c.o.GetLocalSlot(c.slot)
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
	o := testVM.NewObject(Slots{})
	testVM.Lobby.SetSlot("TestObjectActivate", o)
	cases := map[string]SourceTestCase{
		"InactiveNoActivate": {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(false)`, PassEqual(o)},
		"InactiveActivate":   {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(false)`, PassEqual(o)},
		"ActiveNoActivate":   {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(true)`, PassEqual(o)},
		"ActiveActivate":     {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(true)`, PassEqual(testVM.Lobby)},
	}
	for name, c := range cases {
		t.Run(name, c.TestFunc("TestObjectActivate/"+name))
	}
	testVM.Lobby.RemoveSlot("TestObjectActivate")
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
		"@",
		"@@",
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
		"coroDo",
		"coroDoLater",
		"coroFor",
		"coroWith",
		"currentCoro",
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

// TestObjectScript tests Object methods by executing Io scripts.
func TestObjectMethods(t *testing.T) {
	list012 := testVM.NewList(testVM.NewNumber(0), testVM.NewNumber(1), testVM.NewNumber(2))
	listxyz := testVM.NewList(testVM.NewString("x"), testVM.NewString("y"), testVM.NewString("z"))
	cases := map[string]map[string]SourceTestCase{
		"evalArg": {
			"evalArg":   {`evalArg(Lobby)`, PassEqual(testVM.Lobby)},
			"continue":  {`evalArg(continue; Lobby)`, PassControl(testVM.Nil, ContinueStop)},
			"exception": {`evalArg(Exception raise; Lobby)`, PassFailure()},
		},
		"notEqual": {
			"0!=1":         {`0 !=(1)`, PassIdentical(testVM.True)},
			"0!=0":         {`0 !=(0)`, PassIdentical(testVM.False)},
			"1!=0":         {`1 !=(0)`, PassIdentical(testVM.True)},
			"incomparable": {`Lobby !=(Core)`, PassIdentical(testVM.True)},
			"identical":    {`Lobby !=(Lobby)`, PassIdentical(testVM.False)},
			"continue":     {`Lobby !=(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`Lobby !=(Exception raise); Lobby`, PassFailure()},
		},
		"less": {
			"0<1":          {`0 <(1)`, PassIdentical(testVM.True)},
			"0<0":          {`0 <(0)`, PassIdentical(testVM.False)},
			"1<0":          {`1 <(0)`, PassIdentical(testVM.False)},
			"incomparable": {`Lobby <(Core)`, PassSuccess()},
			"identical":    {`Lobby <(Lobby)`, PassIdentical(testVM.False)},
			"continue":     {`0 <(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`0 <(Exception raise); Lobby`, PassFailure()},
		},
		"lessEqual": {
			"0<=1":         {`0 <=(1)`, PassIdentical(testVM.True)},
			"0<=0":         {`0 <=(0)`, PassIdentical(testVM.True)},
			"1<=0":         {`1 <=(0)`, PassIdentical(testVM.False)},
			"incomparable": {`Lobby <=(Core)`, PassSuccess()},
			"identical":    {`Lobby <=(Lobby)`, PassIdentical(testVM.True)},
			"continue":     {`0 <=(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`0 <=(Exception raise); Lobby`, PassFailure()},
		},
		"equal": {
			"0==1":         {`0 ==(1)`, PassIdentical(testVM.False)},
			"0==0":         {`0 ==(0)`, PassIdentical(testVM.True)},
			"1==0":         {`1 ==(0)`, PassIdentical(testVM.False)},
			"incomparable": {`Lobby ==(Core)`, PassIdentical(testVM.False)},
			"identical":    {`Lobby ==(Lobby)`, PassIdentical(testVM.True)},
			"continue":     {`Lobby ==(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`Lobby ==(Exception raise); Lobby`, PassFailure()},
		},
		"greater": {
			"0>1":          {`0 >(1)`, PassIdentical(testVM.False)},
			"0>0":          {`0 >(0)`, PassIdentical(testVM.False)},
			"1>0":          {`1 >(0)`, PassIdentical(testVM.True)},
			"incomparable": {`Lobby >(Core)`, PassSuccess()},
			"identical":    {`Lobby >(Lobby)`, PassIdentical(testVM.False)},
			"continue":     {`0 >(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`0 >(Exception raise); Lobby`, PassFailure()},
		},
		"greaterEqual": {
			"0>=1":         {`0 >=(1)`, PassIdentical(testVM.False)},
			"0>=0":         {`0 >=(0)`, PassIdentical(testVM.True)},
			"1>=0":         {`1 >=(0)`, PassIdentical(testVM.True)},
			"incomparable": {`Lobby >=(Core)`, PassSuccess()},
			"identical":    {`Lobby >=(Lobby)`, PassIdentical(testVM.True)},
			"continue":     {`0 >=(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`0 >=(Exception raise); Lobby`, PassFailure()},
		},
		"ancestorWithSlot": {
			"local":         {`Number ancestorWithSlot("abs")`, PassIdentical(testVM.Nil)},
			"localInProtos": {`Lobby ancestorWithSlot("Lobby")`, PassIdentical(testVM.Lobby)},
			"nowhere":       {`Lobby ancestorWithSlot("this slot doesn't exist")`, PassIdentical(testVM.Nil)},
			"bad":           {`Lobby ancestorWithSlot(0)`, PassFailure()},
			"continue":      {`Lobby ancestorWithSlot(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":     {`Lobby ancestorWithSlot(Exception raise); Lobby`, PassFailure()},
		},
		"appendProto": {
			"appendProto": {`Object clone do(appendProto(Lobby)) protos containsIdenticalTo(Lobby)`, PassIdentical(testVM.True)},
			"continue":    {`Object clone appendProto(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":   {`Object clone appendProto(Exception raise); Lobby`, PassFailure()},
		},
		// asGoRepr gets no tests //
		"asString": {
			"isSequence": {`Object asString`, PassTag(SequenceTag)},
		},
		"asyncSend": {
			"spawns":      {`yield; yield; yield; testValues asyncSendCoros := Scheduler coroCount; asyncSend(wait(1)); wait(testValues coroWaitTime); Scheduler coroCount - testValues asyncSendCoros`, PassEqual(testVM.NewNumber(1))},
			"sideEffects": {`testValues asyncSendSideEffect := 0; asyncSend(Lobby testValues asyncSendSideEffect = 1); wait(testValues coroWaitTime); testValues asyncSendSideEffect`, PassEqual(testVM.NewNumber(1))},
		},
		"block": {
			"noMessage": {`block`, PassTag(BlockTag)},
			"exception": {`block(Exception raise)`, PassSuccess()},
		},
		"break": {
			"break":     {`break`, PassControl(testVM.Nil, BreakStop)},
			"value":     {`break(Lobby)`, PassControl(testVM.Lobby, BreakStop)},
			"continue":  {`break(continue)`, PassControl(testVM.Nil, ContinueStop)},
			"exception": {`break(Exception raise)`, PassFailure()},
		},
		"clone": {
			"new":   {`Object clone != Object`, PassIdentical(testVM.True)},
			"proto": {`Object clone protos containsIdenticalTo(Object)`, PassIdentical(testVM.True)},
			"init":  {`testValues initValue := 0; Object clone do(init := method(Lobby testValues initValue = 1)) clone; testValues initValue`, PassEqual(testVM.NewNumber(1))},
		},
		"cloneWithoutInit": {
			"new":   {`Object cloneWithoutInit != Object`, PassIdentical(testVM.True)},
			"proto": {`Object cloneWithoutInit protos containsIdenticalTo(Object)`, PassIdentical(testVM.True)},
			"init":  {`testValues noInitValue := 0; Object clone do(init := method(Lobby testValues noInitValue = 1)) cloneWithoutInit; testValues noInitValue`, PassEqual(testVM.NewNumber(0))},
		},
		"compare": {
			"incomparable": {`Object compare("string")`, PassTag(NumberTag)},
			"continue":     {`Object compare(continue)`, PassControl(testVM.Nil, ContinueStop)},
			"exception":    {`Object compare(Exception raise)`, PassFailure()},
		},
		"contextWithSlot": {
			"local":         {`Number contextWithSlot("abs")`, PassEqual(testVM.NewNumber(0))},
			"localInProtos": {`Lobby contextWithSlot("Lobby")`, PassIdentical(testVM.Lobby)},
			"nowhere":       {`Lobby contextWithSlot("this slot doesn't exist")`, PassIdentical(testVM.Nil)},
			"bad":           {`Lobby contextWithSlot(0)`, PassFailure()},
			"continue":      {`Lobby contextWithSlot(continue); Lobby`, PassControl(testVM.Nil, ContinueStop)},
			"exception":     {`Lobby contextWithSlot(Exception raise); Lobby`, PassFailure()},
		},
		"continue": {
			"continue":  {`continue`, PassControl(testVM.Nil, ContinueStop)},
			"value":     {`continue(Lobby)`, PassControl(testVM.Lobby, ContinueStop)},
			"exception": {`continue(Exception raise)`, PassFailure()},
		},
		"do": {
			"result":    {`Object do(Lobby)`, PassIdentical(testVM.BaseObject)},
			"context":   {`testValues doValue := 0; testValues do(doValue := 1); testValues doValue`, PassEqual(testVM.NewNumber(1))},
			"continue":  {`do(continue)`, PassControl(testVM.Nil, ContinueStop)},
			"exception": {`do(Exception raise)`, PassFailure()},
		},
		// TODO: doFile needs special testing
		"doMessage": {
			"doMessage": {`testValues doMessageValue := 0; testValues doMessage(message(doMessageValue = 1)); testValues doMessageValue`, PassEqual(testVM.NewNumber(1))},
			"context":   {`testValues doMessageValue := 2; doMessage(message(testValues doMessageValue = doMessageValue + 1), testValues); testValues doMessageValue`, PassEqual(testVM.NewNumber(3))},
			"bad":       {`testValues doMessage("doMessageValue := 4")`, PassFailure()},
		},
		"doString": {
			"doString": {`testValues doStringValue := 0; testValues doString("doStringValue = 1"); testValues doStringValue`, PassEqual(testVM.NewNumber(1))},
			"label":    {`testValues doStringLabel := "foo"; testValues doString("doStringLabel = thisMessage label", "bar"); testValues doStringLabel`, PassEqual(testVM.NewString("bar"))},
			"bad":      {`testValues doString(message(doStringValue := 4))`, PassFailure()},
		},
		"evalArgAndReturnNil": {
			"result":    {`evalArgAndReturnNil(Lobby)`, PassIdentical(testVM.Nil)},
			"eval":      {`testValues evalNil := 0; evalArgAndReturnNil(testValues evalNil := 1); testValues evalNil`, PassEqual(testVM.NewNumber(1))},
			"continue":  {`evalArgAndReturnNil(continue)`, PassControl(testVM.Nil, ContinueStop)},
			"exception": {`evalArgAndReturnNil(Exception raise)`, PassFailure()},
		},
		"evalArgAndReturnSelf": {
			"result":    {`evalArgAndReturnSelf(nil)`, PassIdentical(testVM.Lobby)},
			"eval":      {`testValues evalSelf := 0; evalArgAndReturnSelf(testValues evalSelf := 1); testValues evalSelf`, PassEqual(testVM.NewNumber(1))},
			"continue":  {`evalArgAndReturnSelf(continue)`, PassControl(testVM.Nil, ContinueStop)},
			"exception": {`evalArgAndReturnSelf(Exception raise)`, PassFailure()},
		},
		"for": {
			"result":         {`testValues do(forResult := for(forCtr, 0, 2, forCtr * 2)) forResult`, PassEqual(testVM.NewNumber(4))},
			"nothing":        {`testValues do(forResult := for(forCtr, 2, 0, Exception raise)) forResult`, PassIdentical(testVM.Nil)},
			"order":          {`testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr))) forList`, PassEqual(list012)},
			"step":           {`testValues do(forList := list; for(forCtr, 2, 0, -1, forList append(forCtr))) forList reverse`, PassEqual(list012)},
			"continue":       {`testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr); continue; forList append(forCtr))) forList`, PassEqual(list012)},
			"continueResult": {`testValues do(forResult := for(forCtr, 0, 2, continue(forCtr * 2))) forResult`, PassEqual(testVM.NewNumber(4))},
			"break":          {`testValues do(for(forCtr, 0, 2, break)) forCtr`, PassEqual(testVM.NewNumber(0))},
			"breakResult":    {`testValues do(forResult := for(forCtr, 0, 2, break(4))) forResult`, PassEqual(testVM.NewNumber(4))},
			"return":         {`testValues do(for(forCtr, 0, 2, return))`, PassControl(testVM.Nil, ReturnStop)},
			"exception":      {`testValues do(for(forCtr, 0, 2, Exception raise))`, PassFailure()},
			"name":           {`testValues do(for(forCtrNew no responderino, 0, 2, nil))`, PassLocalSlots([]string{"forCtrNew"}, []string{"no", "responderino"})},
			"short":          {`testValues do(for(forCtr))`, PassFailure()},
			"long":           {`testValues do(for(forCtr, 0, 1, 2, 3, 4, 5))`, PassFailure()},
		},
		"foreachSlot": {
			"result":         {`testValues do(foreachSlotResult := obj foreachSlot(value, 4)) foreachSlotResult`, PassEqual(testVM.NewNumber(4))},
			"nothing":        {`testValues do(foreachSlotResult := Object clone foreachSlot(value, Exception raise)) foreachSlotResult`, PassIdentical(testVM.Nil)},
			"key":            {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot))) forList sort`, PassEqual(listxyz)},
			"value":          {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(value))) forList sort`, PassEqual(list012)},
			"continue":       {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot); continue; forList append(slot))) forList sort`, PassEqual(listxyz)},
			"continueResult": {`testValues do(foreachSlotResult := obj foreachSlot(slot, value, continue(Lobby))) foreachSlotResult`, PassIdentical(testVM.Lobby)},
			"break":          {`testValues do(foreachSlotIters := 0; obj foreachSlot(slot, value, foreachSlotIters = foreachSlotIters + 1; break)) foreachSlotIters`, PassEqual(testVM.NewNumber(1))},
			"breakResult":    {`testValues do(foreachSlotResult := obj foreachSlot(slot, value, break(Lobby))) foreachSlotResult`, PassIdentical(testVM.Lobby)},
			"return":         {`testValues do(obj foreachSlot(slot, value, return))`, PassControl(testVM.Nil, ReturnStop)},
			"exception":      {`testValues do(obj foreachSlot(slot, value, Exception raise))`, PassFailure()},
			"name":           {`testValues do(obj foreachSlot(slotNew no responderino, valueNew no responderino, nil))`, PassLocalSlots([]string{"slotNew", "valueNew"}, []string{"no", "responderino"})},
			"short":          {`testValues do(obj foreachSlot(nil))`, PassFailure()},
			"long":           {`testValues do(obj foreachSlot(slot, value, 1, 2, 3))`, PassFailure()},
		},
	}
	// If this test runs before TestLobbySlots, any new slots that tests create
	// will cause that to fail. To circumvent this, we provide an object to
	// carry test values, then remove it once all tests have run. This object
	// initially carries default test configuration values.
	config := Slots{
		// coroWaitTime is the time in seconds that coros should wait while
		// testing methods that spawn new coroutines. The new coroutines may
		// take any amount of time to execute, as the VM does not wait for them
		// to finish.
		"coroWaitTime": testVM.NewNumber(0.02),
		// obj is a generic object with some slots to simplify some tests.
		"obj": testVM.NewObject(Slots{"x": testVM.NewNumber(1), "y": testVM.NewNumber(2), "z": testVM.NewNumber(0)}),
	}
	testVM.Lobby.SetSlot("testValues", testVM.NewObject(config))
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for name, s := range c {
				t.Run(name, s.TestFunc("TestObjectMethods"))
			}
		})
	}
	testVM.Lobby.RemoveSlot("testValues")
}
