package internal_test

import (
	"testing"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/testutils"
)

// TestGetSlot tests that GetSlot can find local and ancestor slots, and that no
// object is checked more than once.
func TestGetSlot(t *testing.T) {
	vm := testutils.VM()
	cases := map[string]struct {
		o, v, p *iolang.Object
		slot    string
	}{
		"Local":    {vm.Lobby, vm.Lobby, vm.Lobby, "Lobby"},
		"Ancestor": {vm.Lobby, vm.BaseObject, vm.Core, "Object"},
		"Never":    {vm.Lobby, nil, nil, "fail to find"},
	}
	// TODO: test case where the lookup chain expands into the general case,
	// then returns to the single-proto case
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, p := vm.GetSlot(c.o, c.slot)
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
	vm := testutils.VM()
	o := vm.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	cases := map[string]struct {
		o    *iolang.Object
		slot string
	}{
		"Local":    {vm.Lobby, "Lobby"},
		"Proto":    {vm.BaseObject, "Lobby"},
		"Ancestor": {o, "Lobby"},
		"Missing":  {vm.Lobby, "Lobby fail to find"},
	}
	for name, c := range cases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchDummy, _ = vm.GetSlot(c.o, c.slot)
			}
		})
	}
}

// TestGetLocalSlot tests that GetLocalSlot can find local but not ancestor
// slots.
func TestGetLocalSlot(t *testing.T) {
	vm := testutils.VM()
	cases := map[string]struct {
		o, v *iolang.Object
		ok   bool
		slot string
	}{
		"Local":    {vm.Lobby, vm.Lobby, true, "Lobby"},
		"Ancestor": {vm.Lobby, nil, false, "Object"},
		"Never":    {vm.Lobby, nil, false, "fail to find"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, ok := vm.GetLocalSlot(c.o, c.slot)
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
	vm := testutils.VM()
	o := vm.NewObject(iolang.Slots{})
	vm.SetSlot(vm.Lobby, "TestObjectActivate", o)
	cases := map[string]testutils.SourceTestCase{
		"InactiveNoActivate": {Source: `getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(false)`, Pass: testutils.PassEqual(o)},
		"InactiveActivate":   {Source: `getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(false)`, Pass: testutils.PassEqual(o)},
		"ActiveNoActivate":   {Source: `getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(true)`, Pass: testutils.PassEqual(o)},
		"ActiveActivate":     {Source: `getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(true)`, Pass: testutils.PassEqual(vm.Lobby)},
	}
	for name, c := range cases {
		t.Run(name, c.TestFunc("TestObjectActivate/"+name))
	}
	vm.RemoveSlot(vm.Lobby, "TestObjectActivate")
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
		"block",
		"break",
		"clone",
		"cloneWithoutInit",
		"compare",
		"contextWithSlot",
		"continue",
		"deprecatedWarning",
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
		"init",
		"inlineMethod",
		"isActivatable",
		"isError",
		"isIdenticalTo",
		"isKindOf",
		"isLaunchScript",
		"isNil",
		"isTrue",
		"justSerialized",
		"launchFile",
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
		"serialized",
		"serializedSlots",
		"serializedSlotsWithNames",
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
		"stopStatus",
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
		"write",
		"writeln",
	}
	testutils.CheckSlots(t, testutils.VM().BaseObject, slots)
}

// TestObjectScript tests Object methods by executing Io scripts.
func TestObjectMethods(t *testing.T) {
	vm := testutils.VM()
	list012 := vm.NewList(vm.NewNumber(0), vm.NewNumber(1), vm.NewNumber(2))
	listxyz := vm.NewList(vm.NewString("x"), vm.NewString("y"), vm.NewString("z"))
	// If this test runs before TestLobbySlots, any new slots that tests create
	// will cause that to fail. To circumvent this, we provide an object to
	// carry test values, then remove it once all tests have run. This object
	// initially carries default test configuration values.
	config := iolang.Slots{
		// coroWaitTime is the time in seconds that coros should wait while
		// testing methods that spawn new coroutines. The new coroutines may
		// take any amount of time to execute, as the VM does not wait for them
		// to finish.
		"coroWaitTime": vm.NewNumber(0.02),
		// obj is a generic object with some slots to simplify some tests.
		"obj": vm.NewObject(iolang.Slots{"x": vm.NewNumber(1), "y": vm.NewNumber(2), "z": vm.NewNumber(0)}),
		// manyProtos is an object that has many protos.
		"manyProtos": vm.ObjectWith(nil, []*iolang.Object{vm.BaseObject, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil}, nil, nil),
		// manyProtosToRemove is an object that has many protos to test that
		// removeAllProtos succeeds.
		"manyProtosToRemove": vm.ObjectWith(nil, []*iolang.Object{vm.BaseObject, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil}, nil, nil),
	}
	vm.SetSlot(vm.Lobby, "testValues", vm.NewObject(config))
	cases := map[string]map[string]testutils.SourceTestCase{
		"evalArg": {
			"evalArg":   {Source: `evalArg(Lobby)`, Pass: testutils.PassEqual(vm.Lobby)},
			"continue":  {Source: `evalArg(continue; Lobby)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `evalArg(Exception raise; Lobby)`, Pass: testutils.PassFailure()},
		},
		"notEqual": {
			"0!=1":         {Source: `0 !=(1)`, Pass: testutils.PassIdentical(vm.True)},
			"0!=0":         {Source: `0 !=(0)`, Pass: testutils.PassIdentical(vm.False)},
			"1!=0":         {Source: `1 !=(0)`, Pass: testutils.PassIdentical(vm.True)},
			"incomparable": {Source: `Lobby !=(Core)`, Pass: testutils.PassIdentical(vm.True)},
			"identical":    {Source: `Lobby !=(Lobby)`, Pass: testutils.PassIdentical(vm.False)},
			"continue":     {Source: `Lobby !=(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `Lobby !=(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"minus": {
			"-1":   {Source: `-(1)`, Pass: testutils.PassEqual(vm.NewNumber(-1))},
			"-seq": {Source: `-("abc")`, Pass: testutils.PassFailure()},
		},
		"dotdot": {
			"1..2": {Source: `1 ..(2)`, Pass: testutils.PassEqual(vm.NewString("12"))},
		},
		"less": {
			"0<1":          {Source: `0 <(1)`, Pass: testutils.PassIdentical(vm.True)},
			"0<0":          {Source: `0 <(0)`, Pass: testutils.PassIdentical(vm.False)},
			"1<0":          {Source: `1 <(0)`, Pass: testutils.PassIdentical(vm.False)},
			"incomparable": {Source: `Lobby <(Core)`, Pass: testutils.PassSuccess()},
			"identical":    {Source: `Lobby <(Lobby)`, Pass: testutils.PassIdentical(vm.False)},
			"continue":     {Source: `0 <(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `0 <(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"lessEqual": {
			"0<=1":         {Source: `0 <=(1)`, Pass: testutils.PassIdentical(vm.True)},
			"0<=0":         {Source: `0 <=(0)`, Pass: testutils.PassIdentical(vm.True)},
			"1<=0":         {Source: `1 <=(0)`, Pass: testutils.PassIdentical(vm.False)},
			"incomparable": {Source: `Lobby <=(Core)`, Pass: testutils.PassSuccess()},
			"identical":    {Source: `Lobby <=(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"continue":     {Source: `0 <=(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `0 <=(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"equal": {
			"0==1":         {Source: `0 ==(1)`, Pass: testutils.PassIdentical(vm.False)},
			"0==0":         {Source: `0 ==(0)`, Pass: testutils.PassIdentical(vm.True)},
			"1==0":         {Source: `1 ==(0)`, Pass: testutils.PassIdentical(vm.False)},
			"incomparable": {Source: `Lobby ==(Core)`, Pass: testutils.PassIdentical(vm.False)},
			"identical":    {Source: `Lobby ==(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"continue":     {Source: `Lobby ==(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `Lobby ==(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"greater": {
			"0>1":          {Source: `0 >(1)`, Pass: testutils.PassIdentical(vm.False)},
			"0>0":          {Source: `0 >(0)`, Pass: testutils.PassIdentical(vm.False)},
			"1>0":          {Source: `1 >(0)`, Pass: testutils.PassIdentical(vm.True)},
			"incomparable": {Source: `Lobby >(Core)`, Pass: testutils.PassSuccess()},
			"identical":    {Source: `Lobby >(Lobby)`, Pass: testutils.PassIdentical(vm.False)},
			"continue":     {Source: `0 >(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `0 >(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"greaterEqual": {
			"0>=1":         {Source: `0 >=(1)`, Pass: testutils.PassIdentical(vm.False)},
			"0>=0":         {Source: `0 >=(0)`, Pass: testutils.PassIdentical(vm.True)},
			"1>=0":         {Source: `1 >=(0)`, Pass: testutils.PassIdentical(vm.True)},
			"incomparable": {Source: `Lobby >=(Core)`, Pass: testutils.PassSuccess()},
			"identical":    {Source: `Lobby >=(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"continue":     {Source: `0 >=(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `0 >=(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"question": {
			"have":      {Source: `?Lobby`, Pass: testutils.PassIdentical(vm.Lobby)},
			"not":       {Source: `?nothing`, Pass: testutils.PassIdentical(vm.Nil)},
			"effect":    {Source: `testValues questionEffect := 0; ?testValues questionEffect := 1; testValues questionEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"continue":  {Source: `?continue`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `?Exception raise`, Pass: testutils.PassFailure()},
			"parens":    {Source: `testValues questionEffect := 0; ?(testValues questionEffect := 1) ?(nothing); testValues questionEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
		},
		"addTrait": {
			"all":    {Source: `Object clone addTrait(testValues obj)`, Pass: testutils.PassEqualSlots(iolang.Slots{"x": vm.NewNumber(1), "y": vm.NewNumber(2), "z": vm.NewNumber(0)})},
			"res":    {Source: `Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("x", "w")))`, Pass: testutils.PassEqualSlots(iolang.Slots{"w": vm.NewNumber(1), "x": vm.NewNumber(4), "y": vm.NewNumber(2), "z": vm.NewNumber(0)})},
			"unres":  {Source: `Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("w", "x")))`, Pass: testutils.PassFailure()},
			"badres": {Source: `Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("x", 1)))`, Pass: testutils.PassFailure()},
			"short":  {Source: `Object clone addTrait`, Pass: testutils.PassFailure()},
		},
		"ancestorWithSlot": {
			"local":         {Source: `Number ancestorWithSlot("abs")`, Pass: testutils.PassIdentical(vm.Nil)},
			"proto":         {Source: `Lobby clone ancestorWithSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"localInProtos": {Source: `Lobby ancestorWithSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"nowhere":       {Source: `Lobby ancestorWithSlot("this slot doesn't exist")`, Pass: testutils.PassIdentical(vm.Nil)},
			"bad":           {Source: `Lobby ancestorWithSlot(0)`, Pass: testutils.PassFailure()},
			"continue":      {Source: `Lobby ancestorWithSlot(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":     {Source: `Lobby ancestorWithSlot(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"ancestors": {
			"ancestors": {Source: `testValues anc := Object clone ancestors; testValues anc containsIdenticalTo(Object) and testValues anc containsIdenticalTo(Core)`, Pass: testutils.PassIdentical(vm.True)},
		},
		"and": {
			"true":  {Source: `and(Object)`, Pass: testutils.PassIdentical(vm.True)},
			"false": {Source: `and(Object clone do(isTrue := false))`, Pass: testutils.PassIdentical(vm.False)},
		},
		// TODO: apropos needs special tests, since it prints
		"appendProto": {
			"appendProto": {Source: `Object clone do(appendProto(Lobby)) protos containsIdenticalTo(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"continue":    {Source: `Object clone appendProto(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":   {Source: `Object clone appendProto(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"asBoolean": {
			"asBoolean": {Source: `Object asBoolean`, Pass: testutils.PassIdentical(vm.True)},
		},
		// asGoRepr gets no tests //
		"asSimpleString": {
			"isSequence": {Source: `Object asSimpleString`, Pass: testutils.PassTag(iolang.SequenceTag)},
		},
		"asString": {
			"isSequence": {Source: `Object asString`, Pass: testutils.PassTag(iolang.SequenceTag)},
		},
		"block": {
			"noMessage": {Source: `block`, Pass: testutils.PassTag(iolang.BlockTag)},
			"exception": {Source: `block(Exception raise)`, Pass: testutils.PassSuccess()},
		},
		"break": {
			"break":     {Source: `break`, Pass: testutils.PassControl(vm.Nil, iolang.BreakStop)},
			"value":     {Source: `break(Lobby)`, Pass: testutils.PassControl(vm.Lobby, iolang.BreakStop)},
			"continue":  {Source: `break(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `break(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"clone": {
			"new":   {Source: `Object clone != Object`, Pass: testutils.PassIdentical(vm.True)},
			"proto": {Source: `Object clone protos containsIdenticalTo(Object)`, Pass: testutils.PassIdentical(vm.True)},
			"init":  {Source: `testValues initValue := 0; Object clone do(init := method(Lobby testValues initValue = 1)) clone; testValues initValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
		},
		"cloneWithoutInit": {
			"new":   {Source: `Object cloneWithoutInit != Object`, Pass: testutils.PassIdentical(vm.True)},
			"proto": {Source: `Object cloneWithoutInit protos containsIdenticalTo(Object)`, Pass: testutils.PassIdentical(vm.True)},
			"init":  {Source: `testValues noInitValue := 0; Object clone do(init := method(Lobby testValues noInitValue = 1)) cloneWithoutInit; testValues noInitValue`, Pass: testutils.PassEqual(vm.NewNumber(0))},
		},
		"compare": {
			"incomparable": {Source: `Object compare("string")`, Pass: testutils.PassTag(iolang.NumberTag)},
			"continue":     {Source: `Object compare(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":    {Source: `Object compare(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"contextWithSlot": {
			"local":         {Source: `Number contextWithSlot("abs")`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"proto":         {Source: `Lobby clone contextWithSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"localInProtos": {Source: `Lobby contextWithSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"nowhere":       {Source: `Lobby contextWithSlot("this slot doesn't exist")`, Pass: testutils.PassIdentical(vm.Nil)},
			"bad":           {Source: `Lobby contextWithSlot(0)`, Pass: testutils.PassFailure()},
			"continue":      {Source: `Lobby contextWithSlot(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":     {Source: `Lobby contextWithSlot(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"continue": {
			"continue":  {Source: `continue`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"value":     {Source: `continue(Lobby)`, Pass: testutils.PassControl(vm.Lobby, iolang.ContinueStop)},
			"exception": {Source: `continue(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"deprecatedWarning": {
			"context": {Source: `deprecatedWarning`, Pass: testutils.PassFailure()},
			// TODO: deprecatedWarning needs special tests, since it prints
		},
		"do": {
			"result":    {Source: `Object do(Lobby)`, Pass: testutils.PassIdentical(vm.BaseObject)},
			"context":   {Source: `testValues doValue := 0; testValues do(doValue := 1); testValues doValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"continue":  {Source: `do(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `do(Exception raise)`, Pass: testutils.PassFailure()},
		},
		// TODO: doFile needs special testing
		"doMessage": {
			"doMessage": {Source: `testValues doMessageValue := 0; testValues doMessage(message(doMessageValue = 1)); testValues doMessageValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"context":   {Source: `testValues doMessageValue := 2; doMessage(message(testValues doMessageValue = doMessageValue + 1), testValues); testValues doMessageValue`, Pass: testutils.PassEqual(vm.NewNumber(3))},
			"bad":       {Source: `testValues doMessage("doMessageValue := 4")`, Pass: testutils.PassFailure()},
		},
		// TODO: doRelativeFile needs special testing
		"doString": {
			"doString": {Source: `testValues doStringValue := 0; testValues doString("doStringValue = 1"); testValues doStringValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"label":    {Source: `testValues doStringLabel := "foo"; testValues doString("doStringLabel = thisMessage label", "bar"); testValues doStringLabel`, Pass: testutils.PassEqual(vm.NewString("bar"))},
			"bad":      {Source: `testValues doString(message(doStringValue := 4))`, Pass: testutils.PassFailure()},
		},
		"evalArgAndReturnNil": {
			"result":    {Source: `evalArgAndReturnNil(Lobby)`, Pass: testutils.PassIdentical(vm.Nil)},
			"eval":      {Source: `testValues evalNil := 0; evalArgAndReturnNil(testValues evalNil := 1); testValues evalNil`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"continue":  {Source: `evalArgAndReturnNil(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `evalArgAndReturnNil(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"evalArgAndReturnSelf": {
			"result":    {Source: `evalArgAndReturnSelf(nil)`, Pass: testutils.PassIdentical(vm.Lobby)},
			"eval":      {Source: `testValues evalSelf := 0; evalArgAndReturnSelf(testValues evalSelf := 1); testValues evalSelf`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"continue":  {Source: `evalArgAndReturnSelf(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `evalArgAndReturnSelf(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"for": {
			"result":         {Source: `testValues do(forResult := for(forCtr, 0, 2, forCtr * 2)) forResult`, Pass: testutils.PassEqual(vm.NewNumber(4))},
			"nothing":        {Source: `testValues do(forResult := for(forCtr, 2, 0, Exception raise)) forResult`, Pass: testutils.PassIdentical(vm.Nil)},
			"order":          {Source: `testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr))) forList`, Pass: testutils.PassEqual(list012)},
			"step":           {Source: `testValues do(forList := list; for(forCtr, 2, 0, -1, forList append(forCtr))) forList reverse`, Pass: testutils.PassEqual(list012)},
			"continue":       {Source: `testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr); continue; forList append(forCtr))) forList`, Pass: testutils.PassEqual(list012)},
			"continueResult": {Source: `testValues do(forResult := for(forCtr, 0, 2, continue(forCtr * 2))) forResult`, Pass: testutils.PassEqual(vm.NewNumber(4))},
			"break":          {Source: `testValues do(for(forCtr, 0, 2, break)) forCtr`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"breakResult":    {Source: `testValues do(forResult := for(forCtr, 0, 2, break(4))) forResult`, Pass: testutils.PassEqual(vm.NewNumber(4))},
			"return":         {Source: `testValues do(for(forCtr, 0, 2, return))`, Pass: testutils.PassControl(vm.Nil, iolang.ReturnStop)},
			"exception":      {Source: `testValues do(for(forCtr, 0, 2, Exception raise))`, Pass: testutils.PassFailure()},
			"name":           {Source: `testValues do(for(forCtrNew no responderino, 0, 2, nil))`, Pass: testutils.PassLocalSlots([]string{"forCtrNew"}, []string{"no", "responderino"})},
			"short":          {Source: `testValues do(for(forCtr))`, Pass: testutils.PassFailure()},
			"long":           {Source: `testValues do(for(forCtr, 0, 1, 2, 3, 4, 5))`, Pass: testutils.PassFailure()},
		},
		"foreachSlot": {
			"result":         {Source: `testValues do(foreachSlotResult := obj foreachSlot(value, 4)) foreachSlotResult`, Pass: testutils.PassEqual(vm.NewNumber(4))},
			"nothing":        {Source: `testValues do(foreachSlotResult := Object clone foreachSlot(value, Exception raise)) foreachSlotResult`, Pass: testutils.PassIdentical(vm.Nil)},
			"key":            {Source: `testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot))) forList sort`, Pass: testutils.PassEqual(listxyz)},
			"value":          {Source: `testValues do(forList := list; obj foreachSlot(slot, value, forList append(value))) forList sort`, Pass: testutils.PassEqual(list012)},
			"continue":       {Source: `testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot); continue; forList append(slot))) forList sort`, Pass: testutils.PassEqual(listxyz)},
			"continueResult": {Source: `testValues do(foreachSlotResult := obj foreachSlot(slot, value, continue(Lobby))) foreachSlotResult`, Pass: testutils.PassIdentical(vm.Lobby)},
			"break":          {Source: `testValues do(foreachSlotIters := 0; obj foreachSlot(slot, value, foreachSlotIters = foreachSlotIters + 1; break)) foreachSlotIters`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"breakResult":    {Source: `testValues do(foreachSlotResult := obj foreachSlot(slot, value, break(Lobby))) foreachSlotResult`, Pass: testutils.PassIdentical(vm.Lobby)},
			"return":         {Source: `testValues do(obj foreachSlot(slot, value, return))`, Pass: testutils.PassControl(vm.Nil, iolang.ReturnStop)},
			"exception":      {Source: `testValues do(obj foreachSlot(slot, value, Exception raise))`, Pass: testutils.PassFailure()},
			"name":           {Source: `testValues do(obj foreachSlot(slotNew no responderino, valueNew no responderino, nil))`, Pass: testutils.PassLocalSlots([]string{"slotNew", "valueNew"}, []string{"no", "responderino"})},
			"short":          {Source: `testValues do(obj foreachSlot(nil))`, Pass: testutils.PassFailure()},
			"long":           {Source: `testValues do(obj foreachSlot(slot, value, 1, 2, 3))`, Pass: testutils.PassFailure()},
		},
		"getLocalSlot": {
			"local":    {Source: `getLocalSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"ancestor": {Source: `Lobby clone getLocalSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Nil)},
			"never":    {Source: `getLocalSlot("this slot does not exist")`, Pass: testutils.PassIdentical(vm.Nil)},
			"bad":      {Source: `getLocalSlot(Lobby)`, Pass: testutils.PassFailure()},
		},
		"getSlot": {
			"local":    {Source: `getSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"ancestor": {Source: `Lobby clone getSlot("Lobby")`, Pass: testutils.PassIdentical(vm.Lobby)},
			"never":    {Source: `getSlot("this slot does not exist")`, Pass: testutils.PassIdentical(vm.Nil)},
			"bad":      {Source: `getSlot(Lobby)`, Pass: testutils.PassFailure()},
		},
		"hasLocalSlot": {
			"local":    {Source: `hasLocalSlot("Lobby")`, Pass: testutils.PassIdentical(vm.True)},
			"ancestor": {Source: `Lobby clone hasLocalSlot("Lobby")`, Pass: testutils.PassIdentical(vm.False)},
			"never":    {Source: `hasLocalSlot("this slot does not exist")`, Pass: testutils.PassIdentical(vm.False)},
			"bad":      {Source: `hasLocalSlot(Lobby)`, Pass: testutils.PassFailure()},
		},
		"hasSlot": {
			"local":    {Source: `hasSlot("Lobby")`, Pass: testutils.PassIdentical(vm.True)},
			"ancestor": {Source: `Lobby clone hasSlot("Lobby")`, Pass: testutils.PassIdentical(vm.True)},
			"never":    {Source: `hasSlot("this slot does not exist")`, Pass: testutils.PassIdentical(vm.False)},
			"bad":      {Source: `hasSlot(Lobby)`, Pass: testutils.PassFailure()},
		},
		"if": {
			"evalTrue":           {Source: `testValues ifResult := nil; if(true, testValues ifResult := true, testValues ifResult := false); testValues ifResult`, Pass: testutils.PassIdentical(vm.True)},
			"evalFalse":          {Source: `testValues ifResult := nil; if(false, testValues ifResult := true, testValues ifResult := false); testValues ifResult`, Pass: testutils.PassIdentical(vm.False)},
			"resultTrue":         {Source: `if(true, 1, 0)`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"resultFalse":        {Source: `if(false, 1, 0)`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"resultTrueDiadic":   {Source: `if(true, 1)`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"resultFalseDiadic":  {Source: `if(false, 1)`, Pass: testutils.PassIdentical(vm.False)},
			"resultTrueMonadic":  {Source: `if(true)`, Pass: testutils.PassIdentical(vm.True)},
			"resultFalseMonadic": {Source: `if(false)`, Pass: testutils.PassIdentical(vm.False)},
			"continue1":          {Source: `if(continue, nil, nil)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"continue2":          {Source: `if(true, continue, nil)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"continue3":          {Source: `if(false, nil, continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception1":         {Source: `if(Exception raise, nil, nil)`, Pass: testutils.PassFailure()},
			"exception2":         {Source: `if(true, Exception raise, nil)`, Pass: testutils.PassFailure()},
			"exception3":         {Source: `if(false, nil, Exception raise)`, Pass: testutils.PassFailure()},
		},
		"ifError": {
			"noEval": {Source: `testValues ifErrorResult := nil; ifError(testValues ifErrorResult := true); testValues ifErrorResult`, Pass: testutils.PassIdentical(vm.Nil)},
			"self":   {Source: `ifError(nil)`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"ifNil": {
			"noEval": {Source: `testValues ifNilResult := false; ifNil(testValues ifNilResult := true); testValues ifNilResult`, Pass: testutils.PassIdentical(vm.False)},
			"self":   {Source: `ifNil(nil)`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"ifNilEval": {
			"noEval": {Source: `testValues ifNilEvalResult := false; ifNilEval(testValues ifNilEvalResult := true); testValues ifNilEvalResult`, Pass: testutils.PassIdentical(vm.False)},
			"self":   {Source: `ifNilEval(nil)`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"ifNonNil": {
			"result":    {Source: `ifNonNil(nil)`, Pass: testutils.PassIdentical(vm.Lobby)},
			"eval":      {Source: `testValues ifNonNilResult := 0; ifNonNil(testValues ifNonNilResult := 1); testValues ifNonNilResult`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"continue":  {Source: `ifNonNil(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `ifNonNil(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"ifNonNilEval": {
			"evalArg":   {Source: `ifNonNilEval(Lobby)`, Pass: testutils.PassEqual(vm.Lobby)},
			"continue":  {Source: `ifNonNilEval(continue; Lobby)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `ifNonNilEval(Exception raise; Lobby)`, Pass: testutils.PassFailure()},
		},
		"in": {
			"contains": {Source: `testValues contains := method(self inResult := true); Object in(testValues); testValues inResult`, Pass: testutils.PassIdentical(vm.True)},
		},
		"init": {
			"self": {Source: `init`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"inlineMethod": {
			"type": {Source: `inlineMethod(nil)`, Pass: testutils.PassTag(iolang.MessageTag)},
			"text": {Source: `inlineMethod(nil) name`, Pass: testutils.PassEqual(vm.NewString("nil"))},
			"next": {Source: `inlineMethod(true nil) next name`, Pass: testutils.PassEqual(vm.NewString("nil"))},
			"prev": {Source: `inlineMethod(true) previous`, Pass: testutils.PassIdentical(vm.Nil)},
		},
		"isActivatable": {
			"false": {Source: `Object isActivatable`, Pass: testutils.PassIdentical(vm.False)},
		},
		"isError": {
			"false": {Source: `Object isError`, Pass: testutils.PassIdentical(vm.False)},
		},
		"isIdenticalTo": {
			"0===0":       {Source: `123456789 isIdenticalTo(123456789)`, Pass: testutils.PassIdentical(vm.False)},
			"unidentical": {Source: `Lobby isIdenticalTo(Core)`, Pass: testutils.PassIdentical(vm.False)},
			"identical":   {Source: `Lobby isIdenticalTo(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"continue":    {Source: `Lobby isIdenticalTo(continue); Lobby`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception":   {Source: `Lobby isIdenticalTo(Exception raise); Lobby`, Pass: testutils.PassFailure()},
		},
		"isKindOf": {
			"self":      {Source: `Object isKindOf(Object)`, Pass: testutils.PassIdentical(vm.True)},
			"proto":     {Source: `Object clone isKindOf(Object)`, Pass: testutils.PassIdentical(vm.True)},
			"ancestor":  {Source: `0 isKindOf(Lobby)`, Pass: testutils.PassIdentical(vm.True)},
			"not":       {Source: `Object isKindOf(Exception)`, Pass: testutils.PassIdentical(vm.False)},
			"continue":  {Source: `isKindOf(continue; Lobby)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `isKindOf(Exception raise; Lobby)`, Pass: testutils.PassFailure()},
		},
		// isLaunchScript needs special testing
		"isNil": {
			"false": {Source: `Object isNil`, Pass: testutils.PassIdentical(vm.False)},
		},
		"isTrue": {
			"true": {Source: `Object isTrue`, Pass: testutils.PassIdentical(vm.True)},
		},
		"justSerialized": {
			"same": {Source: `testValues justSerializedStream := SerializationStream clone; testValues obj justSerialized(testValues justSerializedStream); doString(testValues justSerializedStream output)`, Pass: testutils.PassEqualSlots(vm.GetAllSlots(config["obj"]))},
		},
		// launchFile needs special testing
		"lazySlot": {
			"initial1": {Source: `lazySlot(1)`, Pass: testutils.PassUnequal(vm.NewNumber(1))},
			"initial2": {Source: `testValues lazySlot("lazySlotValue", 1); testValues getSlot("lazySlotValue")`, Pass: testutils.PassUnequal(vm.NewNumber(1))},
			"eval1":    {Source: `testValues lazySlotValue := lazySlot(1); testValues lazySlotValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"eval2":    {Source: `testValues lazySlot("lazySlotValue", 1); testValues lazySlotValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"replace1": {Source: `testValues lazySlotValue := lazySlot(1); testValues lazySlotValue; testValues getSlot("lazySlotValue")`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"replace2": {Source: `testValues lazySlot("lazySlotValue", 1); testValues lazySlotValue; testValues getSlot("lazySlotValue")`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"once1":    {Source: `testValues lazySlotCount := 0; testValues lazySlotValue := lazySlot(testValues lazySlotCount = testValues lazySlotCount + 1; 1); testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotCount`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"once2":    {Source: `testValues lazySlotCount := 0; testValues lazySlot("lazySlotValue", testValues lazySlotCount = testValues lazySlotCount + 1; 1); testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotCount`, Pass: testutils.PassEqual(vm.NewNumber(1))},
		},
		"lexicalDo": {
			// These tests have to be careful not to call lexicalDo on an
			// object that already has the lexical context as a proto.
			"result":    {Source: `Lobby lexicalDo(Object)`, Pass: testutils.PassIdentical(vm.Lobby)},
			"context":   {Source: `testValues lexicalDoValue := 0; testValues lexicalDo(lexicalDoValue := 1); testValues lexicalDoValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"lexical":   {Source: `testValues lexicalDo(lexicalDoHasProto := thisContext protos containsIdenticalTo(Lobby)); testValues lexicalDoHasProto`, Pass: testutils.PassIdentical(vm.True)},
			"continue":  {Source: `Object clone lexicalDo(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `Object clone lexicalDo(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"list": {
			// Object list is List with, but still test it in both places.
			"zero":      {Source: `Object list`, Pass: testutils.PassEqual(vm.NewList())},
			"one":       {Source: `Object list(nil)`, Pass: testutils.PassEqual(vm.NewList(vm.Nil))},
			"five":      {Source: `Object list(nil, nil, nil, nil, nil)`, Pass: testutils.PassEqual(vm.NewList(vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil))},
			"continue":  {Source: `Object list(nil, nil, nil, nil, continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `Object list(nil, nil, nil, nil, Exception raise)`, Pass: testutils.PassFailure()},
		},
		"loop": {
			"loop":      {Source: `testValues loopCount := 0; loop(testValues loopCount = testValues loopCount + 1; if(testValues loopCount >= 5, break)); testValues loopCount`, Pass: testutils.PassEqual(vm.NewNumber(5))},
			"continue":  {Source: `testValues loopCount := 0; loop(testValues loopCount = testValues loopCount + 1; if(testValues loopCount < 5, continue); break); testValues loopCount`, Pass: testutils.PassEqual(vm.NewNumber(5))},
			"break":     {Source: `testValues loopCount := 0; loop(break; testValues loopCount = 1); testValues loopCount`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"return":    {Source: `testValues loopCount := 0; loop(return nil; testValues loopCount = 1); testValues loopCount`, Pass: testutils.PassControl(vm.Nil, iolang.ReturnStop)},
			"exception": {Source: `testValues loopCount := 0; loop(Exception raise; testValues loopCount = 1); testValues loopCount`, Pass: testutils.PassFailure()},
		},
		"message": {
			"nothing":   {Source: `message`, Pass: testutils.PassIdentical(vm.Nil)},
			"message":   {Source: `message(message)`, Pass: testutils.PassTag(iolang.MessageTag)},
			"continue":  {Source: `message(continue)`, Pass: testutils.PassTag(iolang.MessageTag)},
			"exception": {Source: `message(Exception raise)`, Pass: testutils.PassTag(iolang.MessageTag)},
		},
		"method": {
			"noMessage": {Source: `method`, Pass: testutils.PassTag(iolang.BlockTag)},
			"exception": {Source: `method(Exception raise)`, Pass: testutils.PassSuccess()},
		},
		"newSlot": {
			"makes":  {Source: `testValues newSlot("newSlotValue"); testValues`, Pass: testutils.PassLocalSlots([]string{"newSlotValue", "setNewSlotValue"}, nil)},
			"value":  {Source: `testValues newSlot("newSlotValue", 1); testValues newSlotValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"setter": {Source: `testValues newSlot("newSlotValue", 1); testValues setNewSlotValue(2); testValues newSlotValue`, Pass: testutils.PassEqual(vm.NewNumber(2))},
			"result": {Source: `testValues newSlot("newSlotValue", 1)`, Pass: testutils.PassEqual(vm.NewNumber(1))},
		},
		"not": {
			"nil": {Source: `Object not`, Pass: testutils.PassIdentical(vm.Nil)},
		},
		"or": {
			// It might be nice to change or to be a coalescing operator like
			// Python's or, by changing it to thisContext.
			"true": {Source: `Object or`, Pass: testutils.PassIdentical(vm.True)},
		},
		"perform": {
			"string":    {Source: `testValues performValue := 0; testValues perform("setSlot", "performValue", 1); testValues performValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"message":   {Source: `testValues performValue := 0; testValues perform(message(setSlot("performValue", 1))); testValues performValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"single":    {Source: `testValues performValue := 0; testValues perform(message(nil; setSlot("performValue", 1))); testValues performValue`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"several":   {Source: `testValues perform(message(nil), message(nil))`, Pass: testutils.PassFailure()},
			"wrong":     {Source: `testValues perform(nil)`, Pass: testutils.PassFailure()},
			"continue":  {Source: `testValues perform(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `testValues perform(Exception raise)`, Pass: testutils.PassFailure()},
		},
		"performWithArgList": {
			"perform":   {Source: `testValues performWithValue := 0; testValues performWithArgList("setSlot", list("performWithValue", 1)); testValues performWithValue`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"wrong":     {Source: `testValues performWithArgList("nil", "nil")`, Pass: testutils.PassFailure()},
			"continue":  {Source: `testValues performWithArgList(continue, list)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `testValues performWithArgList(Exception raise, list)`, Pass: testutils.PassFailure()},
		},
		"prependProto": {
			"none":      {Source: `testValues prependProtoObj := Object clone removeAllProtos; Object getSlot("prependProto") performOn(testValues prependProtoObj, thisLocalContext, message(prependProto(Lobby))); Object getSlot("protos") performOn(testValues prependProtoObj)`, Pass: testutils.PassEqual(vm.NewList(vm.Lobby))},
			"one":       {Source: `testValues prependProtoObj := Object clone; testValues prependProtoObj prependProto(Lobby); testValues prependProtoObj protos`, Pass: testutils.PassEqual(vm.NewList(vm.Lobby, vm.BaseObject))},
			"ten":       {Source: `testValues prependProtoObj := Object clone; testValues prependProtoObj prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby) prependProto(Lobby); testValues prependProtoObj protos`, Pass: testutils.PassEqual(vm.NewList(vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.Lobby, vm.BaseObject))},
			"continue":  {Source: `Object clone prependProto(continue)`, Pass: testutils.PassControl(vm.Nil, iolang.ContinueStop)},
			"exception": {Source: `Object clone prependProto(Exception raise)`, Pass: testutils.PassFailure()},
		},
		// print and println need special tests
		"proto": {
			"proto": {Source: `testValues proto`, Pass: testutils.PassIdentical(vm.BaseObject)},
		},
		"protos": {
			"none": {Source: `Object getSlot("protos") performOn(Object clone removeAllProtos)`, Pass: testutils.PassEqual(vm.NewList())},
			"one":  {Source: `Object clone protos`, Pass: testutils.PassEqual(vm.NewList(vm.BaseObject))},
			"ten":  {Source: `testValues manyProtos protos`, Pass: testutils.PassEqual(vm.NewList(vm.BaseObject, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil))},
		},
		"raiseIfError": {
			"nothing": {Source: `Object raiseIfError`, Pass: testutils.PassSuccess()},
		},
		// relativeDoFile needs special tests
		"removeAllProtos": {
			// There is no way to test from an Io script that both protos and
			// removeAllProtos work. We only test that removeAllProtos does not
			// completely fail here and save tests that it actually works for
			// another test function.
			"none": {Source: `Object getSlot("removeAllProtos") performOn(Object clone removeAllProtos)`, Pass: testutils.PassSuccess()},
			"one":  {Source: `Object clone removeAllProtos`, Pass: testutils.PassSuccess()},
			"ten":  {Source: `testValues manyProtosToRemove removeAllProtos`, Pass: testutils.PassSuccess()},
		},
		"removeAllSlots": {
			"none": {Source: `Object clone removeAllSlots`, Pass: testutils.PassSuccess()},
			"one":  {Source: `testValues slotsObj := Object clone do(x := 0); testValues slotsObj clone do(x := 1) removeAllSlots x`, Pass: testutils.PassEqual(vm.NewNumber(0))},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for name, s := range c {
				t.Run(name, s.TestFunc("TestObjectMethods"))
			}
		})
	}
	vm.RemoveSlot(vm.Lobby, "testValues")
}
