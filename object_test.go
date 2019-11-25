package iolang

import (
	"testing"
)

// TestGetSlot tests that GetSlot can find local and ancestor slots, and that no
// object is checked more than once.
func TestGetSlot(t *testing.T) {
	vm := TestVM()
	cases := map[string]struct {
		o, v, p *Object
		slot    string
	}{
		"Local":    {vm.Lobby, vm.Lobby, vm.Lobby, "Lobby"},
		"Ancestor": {vm.Lobby, vm.BaseObject, vm.Core, "Object"},
		"Never":    {vm.Lobby, nil, nil, "fail to find"},
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
	vm := TestVM()
	o := vm.BaseObject.Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone().Clone()
	cases := map[string]struct {
		o    *Object
		slot string
	}{
		"Local":    {vm.Lobby, "Lobby"},
		"Proto":    {vm.BaseObject, "Lobby"},
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
	vm := TestVM()
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
	vm := TestVM()
	o := vm.NewObject(Slots{})
	vm.Lobby.SetSlot("TestObjectActivate", o)
	cases := map[string]SourceTestCase{
		"InactiveNoActivate": {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(false)`, PassEqual(o)},
		"InactiveActivate":   {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(false)`, PassEqual(o)},
		"ActiveNoActivate":   {`getSlot("TestObjectActivate") removeSlot("activate") setIsActivatable(true)`, PassEqual(o)},
		"ActiveActivate":     {`getSlot("TestObjectActivate") do(activate := Lobby) setIsActivatable(true)`, PassEqual(vm.Lobby)},
	}
	for name, c := range cases {
		t.Run(name, c.TestFunc("TestObjectActivate/"+name))
	}
	vm.Lobby.RemoveSlot("TestObjectActivate")
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
	CheckSlots(t, TestVM().BaseObject, slots)
}

// TestObjectScript tests Object methods by executing Io scripts.
func TestObjectMethods(t *testing.T) {
	vm := TestVM()
	list012 := vm.NewList(vm.NewNumber(0), vm.NewNumber(1), vm.NewNumber(2))
	listxyz := vm.NewList(vm.NewString("x"), vm.NewString("y"), vm.NewString("z"))
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
		"appendProto": {
			"appendProto": {`Object clone do(appendProto(Lobby)) protos containsIdenticalTo(Lobby)`, PassIdentical(vm.True)},
			"continue":    {`Object clone appendProto(continue); Lobby`, PassControl(vm.Nil, ContinueStop)},
			"exception":   {`Object clone appendProto(Exception raise); Lobby`, PassFailure()},
		},
		// asGoRepr gets no tests //
		"asString": {
			"isSequence": {`Object asString`, PassTag(SequenceTag)},
		},
		"asyncSend": {
			"spawns":      {`yield; yield; yield; testValues asyncSendCoros := Scheduler coroCount; asyncSend(wait(1)); wait(testValues coroWaitTime); Scheduler coroCount - testValues asyncSendCoros`, PassEqual(vm.NewNumber(1))},
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
			"spawns":      {`yield; yield; yield; testValues futureSendCoros := Scheduler coroCount; futureSend(wait(1)); wait(testValues coroWaitTime); Scheduler coroCount - testValues futureSendCoros`, PassEqual(vm.NewNumber(1))},
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
		"coroWaitTime": vm.NewNumber(0.02),
		// obj is a generic object with some slots to simplify some tests.
		"obj": vm.NewObject(Slots{"x": vm.NewNumber(1), "y": vm.NewNumber(2), "z": vm.NewNumber(0)}),
	}
	vm.Lobby.SetSlot("testValues", vm.NewObject(config))
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for name, s := range c {
				t.Run(name, s.TestFunc("TestObjectMethods"))
			}
		})
	}
	vm.Lobby.RemoveSlot("testValues")
}
