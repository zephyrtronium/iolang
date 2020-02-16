package iolang

import (
	"testing"
)

// TestGetSlot tests that GetSlot can find local and ancestor slots, and that no
// object is checked more than once.
func TestGetSlot(t *testing.T) {
	vm := TestingVM()
	cases := map[string]struct {
		o, v, p *Object
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
	vm := TestingVM()
	o := vm.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	cases := map[string]struct {
		o    *Object
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
	vm := TestingVM()
	cases := map[string]struct {
		o, v *Object
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
	vm := TestingVM()
	o := vm.NewObject(Slots{})
	vm.SetSlot(vm.Lobby, "TestObjectActivate", o)
	cases := map[string]SourceTestCase{
		"InactiveNoActivate": {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(false)`, PassEqual(o)},
		"InactiveActivate":   {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(false)`, PassEqual(o)},
		"ActiveNoActivate":   {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(true)`, PassEqual(o)},
		"ActiveActivate":     {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(true)`, PassEqual(vm.Lobby)},
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
		"yield",
	}
	CheckSlots(t, TestingVM().BaseObject, slots)
}

// TestObjectScript tests Object methods by executing Io scripts.
func TestObjectMethods(t *testing.T) {
	vm := TestingVM()
	list012 := vm.NewList(vm.NewNumber(0), vm.NewNumber(1), vm.NewNumber(2))
	listxyz := vm.NewList(vm.NewString("x"), vm.NewString("y"), vm.NewString("z"))
	// If this test runs before TestLobbySlots, any new slots that tests create
	// will cause that to fail. To circumvent this, we provide an object to
	// carry test values, then remove it once all tests have run. This object
	// initially carries default test configuration values.
	config := Slots{
		// coroWaitTime is the time in seconds that coros should wait while
		// testing methods that spawn new coroutines. The new coroutines may
		// take any amount of time to execute, as the VM does not wait for them
		// to finish.
		"coroWaitTime": vm.NewNumber(0.02),
		// obj is a generic object with some slots to simplify some tests.
		"obj": vm.NewObject(Slots{"x": vm.NewNumber(1), "y": vm.NewNumber(2), "z": vm.NewNumber(0)}),
	}
	vm.SetSlot(vm.Lobby, "testValues", vm.NewObject(config))
	cases := map[string]map[string]SourceTestCase{
		"evalArg": {
			"evalArg":   {`evalArg(Lobby)`, PassEqual(vm.Lobby)},
			"continue":  {`evalArg(continue; Lobby)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`evalArg(Exception raise; Lobby)`, PassFailure()},
		},
		"notEqual": {
			"0!=1":         {`0 !=(1)`, PassIdentical(vm.True)},
			"0!=0":         {`0 !=(0)`, PassIdentical(vm.False)},
			"1!=0":         {`1 !=(0)`, PassIdentical(vm.True)},
			"incomparable": {`Lobby !=(Core)`, PassIdentical(vm.True)},
			"identical":    {`Lobby !=(Lobby)`, PassIdentical(vm.False)},
			"continue":     {`Lobby !=(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`Lobby !=(Exception raise); Lobby`, PassFailure()},
		},
		"minus": {
			"-1":   {`-(1)`, PassEqual(vm.NewNumber(-1))},
			"-seq": {`-("abc")`, PassFailure()},
		},
		"dotdot": {
			"1..2": {`1 ..(2)`, PassEqual(vm.NewString("12"))},
		},
		"less": {
			"0<1":          {`0 <(1)`, PassIdentical(vm.True)},
			"0<0":          {`0 <(0)`, PassIdentical(vm.False)},
			"1<0":          {`1 <(0)`, PassIdentical(vm.False)},
			"incomparable": {`Lobby <(Core)`, PassSuccess()},
			"identical":    {`Lobby <(Lobby)`, PassIdentical(vm.False)},
			"continue":     {`0 <(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`0 <(Exception raise); Lobby`, PassFailure()},
		},
		"lessEqual": {
			"0<=1":         {`0 <=(1)`, PassIdentical(vm.True)},
			"0<=0":         {`0 <=(0)`, PassIdentical(vm.True)},
			"1<=0":         {`1 <=(0)`, PassIdentical(vm.False)},
			"incomparable": {`Lobby <=(Core)`, PassSuccess()},
			"identical":    {`Lobby <=(Lobby)`, PassIdentical(vm.True)},
			"continue":     {`0 <=(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`0 <=(Exception raise); Lobby`, PassFailure()},
		},
		"equal": {
			"0==1":         {`0 ==(1)`, PassIdentical(vm.False)},
			"0==0":         {`0 ==(0)`, PassIdentical(vm.True)},
			"1==0":         {`1 ==(0)`, PassIdentical(vm.False)},
			"incomparable": {`Lobby ==(Core)`, PassIdentical(vm.False)},
			"identical":    {`Lobby ==(Lobby)`, PassIdentical(vm.True)},
			"continue":     {`Lobby ==(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`Lobby ==(Exception raise); Lobby`, PassFailure()},
		},
		"greater": {
			"0>1":          {`0 >(1)`, PassIdentical(vm.False)},
			"0>0":          {`0 >(0)`, PassIdentical(vm.False)},
			"1>0":          {`1 >(0)`, PassIdentical(vm.True)},
			"incomparable": {`Lobby >(Core)`, PassSuccess()},
			"identical":    {`Lobby >(Lobby)`, PassIdentical(vm.False)},
			"continue":     {`0 >(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`0 >(Exception raise); Lobby`, PassFailure()},
		},
		"greaterEqual": {
			"0>=1":         {`0 >=(1)`, PassIdentical(vm.False)},
			"0>=0":         {`0 >=(0)`, PassIdentical(vm.True)},
			"1>=0":         {`1 >=(0)`, PassIdentical(vm.True)},
			"incomparable": {`Lobby >=(Core)`, PassSuccess()},
			"identical":    {`Lobby >=(Lobby)`, PassIdentical(vm.True)},
			"continue":     {`0 >=(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`0 >=(Exception raise); Lobby`, PassFailure()},
		},
		"question": {
			"have":      {`?Lobby`, PassIdentical(vm.Lobby)},
			"not":       {`?nothing`, PassIdentical(vm.Nil)},
			"effect":    {`testValues questionEffect := 0; ?testValues questionEffect := 1; testValues questionEffect`, PassEqual(vm.NewNumber(1))},
			"continue":  {`?continue`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`?Exception raise`, PassFailure()},
			"parens":    {`testValues questionEffect := 0; ?(testValues questionEffect := 1) ?(nothing); testValues questionEffect`, PassEqual(vm.NewNumber(1))},
		},
		"addTrait": {
			"all":    {`Object clone addTrait(testValues obj)`, PassEqualSlots(Slots{"x": vm.NewNumber(1), "y": vm.NewNumber(2), "z": vm.NewNumber(0)})},
			"res":    {`Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("x", "w")))`, PassEqualSlots(Slots{"w": vm.NewNumber(1), "x": vm.NewNumber(4), "y": vm.NewNumber(2), "z": vm.NewNumber(0)})},
			"unres":  {`Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("w", "x")))`, PassFailure()},
			"badres": {`Object clone do(x := 4) addTrait(testValues obj, Map clone do(atPut("x", 1)))`, PassFailure()},
			"short":  {`Object clone addTrait`, PassFailure()},
		},
		"ancestorWithSlot": {
			"local":         {`Number ancestorWithSlot("abs")`, PassIdentical(vm.Nil)},
			"proto":         {`Lobby clone ancestorWithSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"localInProtos": {`Lobby ancestorWithSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"nowhere":       {`Lobby ancestorWithSlot("this slot doesn't exist")`, PassIdentical(vm.Nil)},
			"bad":           {`Lobby ancestorWithSlot(0)`, PassFailure()},
			"continue":      {`Lobby ancestorWithSlot(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":     {`Lobby ancestorWithSlot(Exception raise); Lobby`, PassFailure()},
		},
		"ancestors": {
			"ancestors": {`testValues anc := Object clone ancestors; testValues anc containsIdenticalTo(Object) and testValues anc containsIdenticalTo(Core)`, PassIdentical(vm.True)},
		},
		"and": {
			"true":  {`and(Object)`, PassIdentical(testVM.True)},
			"false": {`and(Object clone do(isTrue := false))`, PassIdentical(testVM.False)},
		},
		// TODO: apropos needs special tests, since it prints
		"appendProto": {
			"appendProto": {`Object clone do(appendProto(Lobby)) protos containsIdenticalTo(Lobby)`, PassIdentical(vm.True)},
			"continue":    {`Object clone appendProto(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":   {`Object clone appendProto(Exception raise); Lobby`, PassFailure()},
		},
		"asBoolean": {
			"asBoolean": {`Object asBoolean`, PassIdentical(vm.True)},
		},
		// asGoRepr gets no tests //
		"asSimpleString": {
			"isSequence": {`Object asSimpleString`, PassTag(SequenceTag)},
		},
		"asString": {
			"isSequence": {`Object asString`, PassTag(SequenceTag)},
		},
		"asyncSend": {
			"spawns":      {`while(Scheduler coroCount > 0, yield); testValues asyncSendSync := true; asyncSend(while(testValues asyncSendSync, yield)); wait(testValues coroWaitTime); testValues asyncSendCoros := Scheduler coroCount; testValues asyncSendSync = false; testValues asyncSendCoros`, PassEqual(vm.NewNumber(1))},
			"sideEffects": {`testValues asyncSendSideEffect := 0; asyncSend(Lobby testValues asyncSendSideEffect = 1); wait(testValues coroWaitTime); testValues asyncSendSideEffect`, PassEqual(vm.NewNumber(1))},
			"empty":       {`asyncSend`, PassFailure()},
		},
		"block": {
			"noMessage": {`block`, PassTag(BlockTag)},
			"exception": {`block(Exception raise)`, PassSuccess()},
		},
		"break": {
			"break":     {`break`, PassControl(vm.Nil, BreakStop)},
			"value":     {`break(Lobby)`, PassControl(vm.Lobby, BreakStop)},
			"continue":  {`break(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`break(Exception raise)`, PassFailure()},
		},
		"clone": {
			"new":   {`Object clone != Object`, PassIdentical(vm.True)},
			"proto": {`Object clone protos containsIdenticalTo(Object)`, PassIdentical(vm.True)},
			"init":  {`testValues initValue := 0; Object clone do(init := method(Lobby testValues initValue = 1)) clone; testValues initValue`, PassEqual(vm.NewNumber(1))},
		},
		"cloneWithoutInit": {
			"new":   {`Object cloneWithoutInit != Object`, PassIdentical(vm.True)},
			"proto": {`Object cloneWithoutInit protos containsIdenticalTo(Object)`, PassIdentical(vm.True)},
			"init":  {`testValues noInitValue := 0; Object clone do(init := method(Lobby testValues noInitValue = 1)) cloneWithoutInit; testValues noInitValue`, PassEqual(vm.NewNumber(0))},
		},
		"compare": {
			"incomparable": {`Object compare("string")`, PassTag(NumberTag)},
			"continue":     {`Object compare(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception":    {`Object compare(Exception raise)`, PassFailure()},
		},
		"contextWithSlot": {
			"local":         {`Number contextWithSlot("abs")`, PassEqual(vm.NewNumber(0))},
			"proto":         {`Lobby clone contextWithSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"localInProtos": {`Lobby contextWithSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"nowhere":       {`Lobby contextWithSlot("this slot doesn't exist")`, PassIdentical(vm.Nil)},
			"bad":           {`Lobby contextWithSlot(0)`, PassFailure()},
			"continue":      {`Lobby contextWithSlot(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":     {`Lobby contextWithSlot(Exception raise); Lobby`, PassFailure()},
		},
		"continue": {
			"continue":  {`continue`, PassControl(vm.Nil, ContinueStop)},
			"value":     {`continue(Lobby)`, PassControl(vm.Lobby, ContinueStop)},
			"exception": {`continue(Exception raise)`, PassFailure()},
		},
		"coroDo": {
			"spawns":      {`while(Scheduler coroCount > 0, yield); testValues coroDoSync := true; coroDo(while(testValues coroDoSync, yield)); wait(testValues coroWaitTime); testValues coroDoCoros := Scheduler coroCount; testValues coroDoSync = false; testValues coroDoCoros`, PassEqual(vm.NewNumber(1))},
			"sideEffects": {`testValues coroDoSideEffect := 0; coroDo(testValues coroDoSideEffect := 1); wait(testValues coroWaitTime); testValues coroDoSideEffect`, PassEqual(vm.NewNumber(1))},
			// TODO: test that this has the right target
		},
		"coroDoLater": {
			"spawns":      {`while(Scheduler coroCount > 0, yield); testValues coroDoLaterSync := true; coroDoLater(while(testValues coroDoLaterSync, yield)); wait(testValues coroWaitTime); testValues coroDoLaterCoros := Scheduler coroCount; testValues coroDoLaterSync = false; testValues coroDoLaterCoros`, PassEqual(vm.NewNumber(1))},
			"sideEffects": {`testValues coroDoLaterSideEffect := 0; coroDoLater(testValues coroDoLaterSideEffect := 1); wait(testValues coroWaitTime); testValues coroDoLaterSideEffect`, PassEqual(vm.NewNumber(1))},
			// TODO: test that this has the right target
		},
		"coroFor": {
			"noSpawns":      {`while(Scheduler coroCount > 0, yield); testValues coroForSync := true; coroFor(while(testValues coroForSync, yield)); wait(testValues coroWaitTime); testValues coroForCoros := Scheduler coroCount; testValues coroForSync = false; testValues coroForCoros`, PassEqual(vm.NewNumber(0))},
			"noSideEffects": {`testValues coroForSideEffect := 0; coroFor(testValues coroForSideEffect := 1); wait(testValues coroWaitTime); testValues coroForSideEffect`, PassEqual(vm.NewNumber(0))},
			"type":          {`coroFor(nil)`, PassTag(CoroutineTag)},
			"message":       {`coroFor(nil) runMessage name`, PassEqual(vm.NewString("nil"))},
			"target":        {`0 coroFor(nil) runTarget`, PassIdentical(vm.Lobby)},
			"locals":        {`0 coroFor(nil) runLocals`, PassIdentical(vm.Lobby)},
		},
		"coroWith": {
			"noSpawns":      {`while(Scheduler coroCount > 0, yield); testValues coroWithSync := true; coroWith(while(testValues coroWithSync, yield)); wait(testValues coroWaitTime); testValues coroWithCoros := Scheduler coroCount; testValues coroWithSync = false; testValues coroWithCoros`, PassEqual(vm.NewNumber(0))},
			"noSideEffects": {`testValues coroWithSideEffect := 0; coroWith(testValues coroWithSideEffect := 1); wait(testValues coroWaitTime); testValues coroWithSideEffect`, PassEqual(vm.NewNumber(0))},
			"type":          {`coroWith(nil)`, PassTag(CoroutineTag)},
			"message":       {`coroWith(nil) runMessage name`, PassEqual(vm.NewString("nil"))},
			"target":        {`0 coroWith(nil) runTarget`, PassEqual(vm.NewNumber(0))},
			"locals":        {`0 coroWith(nil) runLocals`, PassIdentical(vm.Lobby)},
		},
		"currentCoro": {
			"isCurrent": {`currentCoro`, PassIdentical(vm.Coro)},
		},
		"deprecatedWarning": {
			"context": {`deprecatedWarning`, PassFailure()},
			// TODO: deprecatedWarning needs special tests, since it prints
		},
		"do": {
			"result":    {`Object do(Lobby)`, PassIdentical(vm.BaseObject)},
			"context":   {`testValues doValue := 0; testValues do(doValue := 1); testValues doValue`, PassEqual(vm.NewNumber(1))},
			"continue":  {`do(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`do(Exception raise)`, PassFailure()},
		},
		// TODO: doFile needs special testing
		"doMessage": {
			"doMessage": {`testValues doMessageValue := 0; testValues doMessage(message(doMessageValue = 1)); testValues doMessageValue`, PassEqual(vm.NewNumber(1))},
			"context":   {`testValues doMessageValue := 2; doMessage(message(testValues doMessageValue = doMessageValue + 1), testValues); testValues doMessageValue`, PassEqual(vm.NewNumber(3))},
			"bad":       {`testValues doMessage("doMessageValue := 4")`, PassFailure()},
		},
		// TODO: doRelativeFile needs special testing
		"doString": {
			"doString": {`testValues doStringValue := 0; testValues doString("doStringValue = 1"); testValues doStringValue`, PassEqual(vm.NewNumber(1))},
			"label":    {`testValues doStringLabel := "foo"; testValues doString("doStringLabel = thisMessage label", "bar"); testValues doStringLabel`, PassEqual(vm.NewString("bar"))},
			"bad":      {`testValues doString(message(doStringValue := 4))`, PassFailure()},
		},
		"evalArgAndReturnNil": {
			"result":    {`evalArgAndReturnNil(Lobby)`, PassIdentical(vm.Nil)},
			"eval":      {`testValues evalNil := 0; evalArgAndReturnNil(testValues evalNil := 1); testValues evalNil`, PassEqual(vm.NewNumber(1))},
			"continue":  {`evalArgAndReturnNil(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`evalArgAndReturnNil(Exception raise)`, PassFailure()},
		},
		"evalArgAndReturnSelf": {
			"result":    {`evalArgAndReturnSelf(nil)`, PassIdentical(vm.Lobby)},
			"eval":      {`testValues evalSelf := 0; evalArgAndReturnSelf(testValues evalSelf := 1); testValues evalSelf`, PassEqual(vm.NewNumber(1))},
			"continue":  {`evalArgAndReturnSelf(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`evalArgAndReturnSelf(Exception raise)`, PassFailure()},
		},
		"for": {
			"result":         {`testValues do(forResult := for(forCtr, 0, 2, forCtr * 2)) forResult`, PassEqual(vm.NewNumber(4))},
			"nothing":        {`testValues do(forResult := for(forCtr, 2, 0, Exception raise)) forResult`, PassIdentical(vm.Nil)},
			"order":          {`testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr))) forList`, PassEqual(list012)},
			"step":           {`testValues do(forList := list; for(forCtr, 2, 0, -1, forList append(forCtr))) forList reverse`, PassEqual(list012)},
			"continue":       {`testValues do(forList := list; for(forCtr, 0, 2, forList append(forCtr); continue; forList append(forCtr))) forList`, PassEqual(list012)},
			"continueResult": {`testValues do(forResult := for(forCtr, 0, 2, continue(forCtr * 2))) forResult`, PassEqual(vm.NewNumber(4))},
			"break":          {`testValues do(for(forCtr, 0, 2, break)) forCtr`, PassEqual(vm.NewNumber(0))},
			"breakResult":    {`testValues do(forResult := for(forCtr, 0, 2, break(4))) forResult`, PassEqual(vm.NewNumber(4))},
			"return":         {`testValues do(for(forCtr, 0, 2, return))`, PassControl(vm.Nil, ReturnStop)},
			"exception":      {`testValues do(for(forCtr, 0, 2, Exception raise))`, PassFailure()},
			"name":           {`testValues do(for(forCtrNew no responderino, 0, 2, nil))`, PassLocalSlots([]string{"forCtrNew"}, []string{"no", "responderino"})},
			"short":          {`testValues do(for(forCtr))`, PassFailure()},
			"long":           {`testValues do(for(forCtr, 0, 1, 2, 3, 4, 5))`, PassFailure()},
		},
		"foreachSlot": {
			"result":         {`testValues do(foreachSlotResult := obj foreachSlot(value, 4)) foreachSlotResult`, PassEqual(vm.NewNumber(4))},
			"nothing":        {`testValues do(foreachSlotResult := Object clone foreachSlot(value, Exception raise)) foreachSlotResult`, PassIdentical(vm.Nil)},
			"key":            {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot))) forList sort`, PassEqual(listxyz)},
			"value":          {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(value))) forList sort`, PassEqual(list012)},
			"continue":       {`testValues do(forList := list; obj foreachSlot(slot, value, forList append(slot); continue; forList append(slot))) forList sort`, PassEqual(listxyz)},
			"continueResult": {`testValues do(foreachSlotResult := obj foreachSlot(slot, value, continue(Lobby))) foreachSlotResult`, PassIdentical(vm.Lobby)},
			"break":          {`testValues do(foreachSlotIters := 0; obj foreachSlot(slot, value, foreachSlotIters = foreachSlotIters + 1; break)) foreachSlotIters`, PassEqual(vm.NewNumber(1))},
			"breakResult":    {`testValues do(foreachSlotResult := obj foreachSlot(slot, value, break(Lobby))) foreachSlotResult`, PassIdentical(vm.Lobby)},
			"return":         {`testValues do(obj foreachSlot(slot, value, return))`, PassControl(vm.Nil, ReturnStop)},
			"exception":      {`testValues do(obj foreachSlot(slot, value, Exception raise))`, PassFailure()},
			"name":           {`testValues do(obj foreachSlot(slotNew no responderino, valueNew no responderino, nil))`, PassLocalSlots([]string{"slotNew", "valueNew"}, []string{"no", "responderino"})},
			"short":          {`testValues do(obj foreachSlot(nil))`, PassFailure()},
			"long":           {`testValues do(obj foreachSlot(slot, value, 1, 2, 3))`, PassFailure()},
		},
		"futureSend": {
			"result":      {`futureSend(1) isNil not`, PassIdentical(vm.True)},
			"evaluates":   {`futureSend(1) + 1`, PassEqual(vm.NewNumber(2))},
			"spawns":      {`while(Scheduler coroCount > 0, yield); testValues futureSendSync := true; futureSend(while(testValues futureSendSync, yield)); wait(testValues coroWaitTime); testValues futureSendCoros := Scheduler coroCount; testValues futureSendSync = false; testValues futureSendCoros`, PassEqual(vm.NewNumber(1))},
			"sideEffects": {`testValues futureSendSideEffect := 0; futureSend(Lobby testValues futureSendSideEffect = 1); wait(testValues coroWaitTime); testValues futureSendSideEffect`, PassEqual(vm.NewNumber(1))},
			"empty":       {`futureSend`, PassFailure()},
		},
		"getLocalSlot": {
			"local":    {`getLocalSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"ancestor": {`Lobby clone getLocalSlot("Lobby")`, PassIdentical(vm.Nil)},
			"never":    {`getLocalSlot("this slot does not exist")`, PassIdentical(vm.Nil)},
			"bad":      {`getLocalSlot(Lobby)`, PassFailure()},
		},
		"getSlot": {
			"local":    {`getSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"ancestor": {`Lobby clone getSlot("Lobby")`, PassIdentical(vm.Lobby)},
			"never":    {`getSlot("this slot does not exist")`, PassIdentical(vm.Nil)},
			"bad":      {`getSlot(Lobby)`, PassFailure()},
		},
		"hasLocalSlot": {
			"local":    {`hasLocalSlot("Lobby")`, PassIdentical(vm.True)},
			"ancestor": {`Lobby clone hasLocalSlot("Lobby")`, PassIdentical(vm.False)},
			"never":    {`hasLocalSlot("this slot does not exist")`, PassIdentical(vm.False)},
			"bad":      {`hasLocalSlot(Lobby)`, PassFailure()},
		},
		"hasSlot": {
			"local":    {`hasSlot("Lobby")`, PassIdentical(vm.True)},
			"ancestor": {`Lobby clone hasSlot("Lobby")`, PassIdentical(vm.True)},
			"never":    {`hasSlot("this slot does not exist")`, PassIdentical(vm.False)},
			"bad":      {`hasSlot(Lobby)`, PassFailure()},
		},
		"if": {
			"evalTrue":           {`testValues ifResult := nil; if(true, testValues ifResult := true, testValues ifResult := false); testValues ifResult`, PassIdentical(vm.True)},
			"evalFalse":          {`testValues ifResult := nil; if(false, testValues ifResult := true, testValues ifResult := false); testValues ifResult`, PassIdentical(vm.False)},
			"resultTrue":         {`if(true, 1, 0)`, PassEqual(vm.NewNumber(1))},
			"resultFalse":        {`if(false, 1, 0)`, PassEqual(vm.NewNumber(0))},
			"resultTrueDiadic":   {`if(true, 1)`, PassEqual(vm.NewNumber(1))},
			"resultFalseDiadic":  {`if(false, 1)`, PassIdentical(vm.False)},
			"resultTrueMonadic":  {`if(true)`, PassIdentical(vm.True)},
			"resultFalseMonadic": {`if(false)`, PassIdentical(vm.False)},
			"continue1":          {`if(continue, nil, nil)`, PassControl(vm.Nil, ContinueStop)},
			"continue2":          {`if(true, continue, nil)`, PassControl(vm.Nil, ContinueStop)},
			"continue3":          {`if(false, nil, continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception1":         {`if(Exception raise, nil, nil)`, PassFailure()},
			"exception2":         {`if(true, Exception raise, nil)`, PassFailure()},
			"exception3":         {`if(false, nil, Exception raise)`, PassFailure()},
		},
		"ifError": {
			"noEval": {`testValues ifErrorResult := nil; ifError(testValues ifErrorResult := true); testValues ifErrorResult`, PassIdentical(vm.Nil)},
			"self":   {`ifError(nil)`, PassIdentical(vm.Lobby)},
		},
		"ifNil": {
			"noEval": {`testValues ifNilResult := false; ifNil(testValues ifNilResult := true); testValues ifNilResult`, PassIdentical(vm.False)},
			"self":   {`ifNil(nil)`, PassIdentical(vm.Lobby)},
		},
		"ifNilEval": {
			"noEval": {`testValues ifNilEvalResult := false; ifNilEval(testValues ifNilEvalResult := true); testValues ifNilEvalResult`, PassIdentical(vm.False)},
			"self":   {`ifNilEval(nil)`, PassIdentical(vm.Lobby)},
		},
		"ifNonNil": {
			"result":    {`ifNonNil(nil)`, PassIdentical(vm.Lobby)},
			"eval":      {`testValues ifNonNilResult := 0; ifNonNil(testValues ifNonNilResult := 1); testValues ifNonNilResult`, PassEqual(vm.NewNumber(1))},
			"continue":  {`ifNonNil(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`ifNonNil(Exception raise)`, PassFailure()},
		},
		"ifNonNilEval": {
			"evalArg":   {`ifNonNilEval(Lobby)`, PassEqual(vm.Lobby)},
			"continue":  {`ifNonNilEval(continue; Lobby)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`ifNonNilEval(Exception raise; Lobby)`, PassFailure()},
		},
		"in": {
			"contains": {`testValues contains := method(self inResult := true); Object in(testValues); testValues inResult`, PassIdentical(vm.True)},
		},
		"init": {
			"self": {`init`, PassIdentical(vm.Lobby)},
		},
		"inlineMethod": {
			"type": {`inlineMethod(nil)`, PassTag(MessageTag)},
			"text": {`inlineMethod(nil) name`, PassEqual(vm.NewString("nil"))},
			"next": {`inlineMethod(true nil) next name`, PassEqual(vm.NewString("nil"))},
			"prev": {`inlineMethod(true) previous`, PassIdentical(vm.Nil)},
		},
		"isActivatable": {
			"false": {`Object isActivatable`, PassIdentical(vm.False)},
		},
		"isError": {
			"false": {`Object isError`, PassIdentical(vm.False)},
		},
		"isIdenticalTo": {
			"0===0":       {`123456789 isIdenticalTo(123456789)`, PassIdentical(vm.False)},
			"unidentical": {`Lobby isIdenticalTo(Core)`, PassIdentical(vm.False)},
			"identical":   {`Lobby isIdenticalTo(Lobby)`, PassIdentical(vm.True)},
			"continue":    {`Lobby isIdenticalTo(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":   {`Lobby isIdenticalTo(Exception raise); Lobby`, PassFailure()},
		},
		"isKindOf": {
			"self":      {`Object isKindOf(Object)`, PassIdentical(vm.True)},
			"proto":     {`Object clone isKindOf(Object)`, PassIdentical(vm.True)},
			"ancestor":  {`0 isKindOf(Lobby)`, PassIdentical(vm.True)},
			"not":       {`Object isKindOf(Exception)`, PassIdentical(vm.False)},
			"continue":  {`isKindOf(continue; Lobby)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`isKindOf(Exception raise; Lobby)`, PassFailure()},
		},
		// isLaunchScript needs special testing
		"isNil": {
			"false": {`Object isNil`, PassIdentical(vm.False)},
		},
		"isTrue": {
			"true": {`Object isTrue`, PassIdentical(vm.True)},
		},
		"justSerialized": {
			"same": {`testValues justSerializedStream := SerializationStream clone; testValues obj justSerialized(testValues justSerializedStream); doString(testValues justSerializedStream output)`, PassEqualSlots(vm.GetAllSlots(config["obj"]))},
		},
		// launchFile needs special testing
		"lazySlot": {
			"initial1": {`lazySlot(1)`, PassUnequal(vm.NewNumber(1))},
			"initial2": {`testValues lazySlot("lazySlotValue", 1); testValues getSlot("lazySlotValue")`, PassUnequal(vm.NewNumber(1))},
			"eval1":    {`testValues lazySlotValue := lazySlot(1); testValues lazySlotValue`, PassEqual(vm.NewNumber(1))},
			"eval2":    {`testValues lazySlot("lazySlotValue", 1); testValues lazySlotValue`, PassEqual(vm.NewNumber(1))},
			"replace1": {`testValues lazySlotValue := lazySlot(1); testValues lazySlotValue; testValues getSlot("lazySlotValue")`, PassEqual(vm.NewNumber(1))},
			"replace2": {`testValues lazySlot("lazySlotValue", 1); testValues lazySlotValue; testValues getSlot("lazySlotValue")`, PassEqual(vm.NewNumber(1))},
			"once1":    {`testValues lazySlotCount := 0; testValues lazySlotValue := lazySlot(testValues lazySlotCount = testValues lazySlotCount + 1; 1); testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotCount`, PassEqual(vm.NewNumber(1))},
			"once2":    {`testValues lazySlotCount := 0; testValues lazySlot("lazySlotValue", testValues lazySlotCount = testValues lazySlotCount + 1; 1); testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotValue; testValues lazySlotCount`, PassEqual(vm.NewNumber(1))},
		},
		"lexicalDo": {
			// These tests have to be careful not to call lexicalDo on an
			// object that already has the lexical context as a proto.
			"result":    {`Lobby lexicalDo(Object)`, PassIdentical(vm.Lobby)},
			"context":   {`testValues lexicalDoValue := 0; testValues lexicalDo(lexicalDoValue := 1); testValues lexicalDoValue`, PassEqual(vm.NewNumber(1))},
			"lexical":   {`testValues lexicalDo(lexicalDoHasProto := thisContext protos containsIdenticalTo(Lobby)); testValues lexicalDoHasProto`, PassIdentical(vm.True)},
			"continue":  {`Object clone lexicalDo(continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`Object clone lexicalDo(Exception raise)`, PassFailure()},
		},
		"list": {
			// Object list is List with, but still test it in both places.
			"zero":      {`Object list`, PassEqual(vm.NewList())},
			"one":       {`Object list(nil)`, PassEqual(vm.NewList(vm.Nil))},
			"five":      {`Object list(nil, nil, nil, nil, nil)`, PassEqual(vm.NewList(vm.Nil, vm.Nil, vm.Nil, vm.Nil, vm.Nil))},
			"continue":  {`Object list(nil, nil, nil, nil, continue)`, PassControl(vm.Nil, ContinueStop)},
			"exception": {`Object list(nil, nil, nil, nil, Exception raise)`, PassFailure()},
		},
		"loop": {
			"loop":      {`testValues loopCount := 0; loop(testValues loopCount = testValues loopCount + 1; if(testValues loopCount >= 5, break)); testValues loopCount`, PassEqual(vm.NewNumber(5))},
			"continue":  {`testValues loopCount := 0; loop(testValues loopCount = testValues loopCount + 1; if(testValues loopCount < 5, continue); break); testValues loopCount`, PassEqual(vm.NewNumber(5))},
			"break":     {`testValues loopCount := 0; loop(break; testValues loopCount = 1); testValues loopCount`, PassEqual(vm.NewNumber(0))},
			"return":    {`testValues loopCount := 0; loop(return nil; testValues loopCount = 1); testValues loopCount`, PassControl(vm.Nil, ReturnStop)},
			"exception": {`testValues loopCount := 0; loop(Exception raise; testValues loopCount = 1); testValues loopCount`, PassFailure()},
		},
		"message": {
			"nothing":   {`message`, PassIdentical(vm.Nil)},
			"message":   {`message(message)`, PassTag(MessageTag)},
			"continue":  {`message(continue)`, PassTag(MessageTag)},
			"exception": {`message(Exception raise)`, PassTag(MessageTag)},
		},
		"method": {
			"noMessage": {`method`, PassTag(BlockTag)},
			"exception": {`method(Exception raise)`, PassSuccess()},
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
