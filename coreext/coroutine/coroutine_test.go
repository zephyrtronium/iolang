package coroutine_test

import (
	"testing"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/coroutine"
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Coroutine"})
}

func TestObjectSlots(t *testing.T) {
	vm := testutils.VM()
	slots := []string{
		"coroDo",
		"coroDoLater",
		"coroFor",
		"coroWith",
		"currentCoro",
		"pause",
		"yield",
	}
	testutils.CheckNewSlots(t, vm.BaseObject, slots)
}

func TestObjectMethods(t *testing.T) {
	vm := testutils.VM()
	config := iolang.Slots{
		// coroWaitTime is the time in seconds that coros should wait while
		// testing methods that spawn new coroutines. The new coroutines may
		// take any amount of time to execute, as the VM does not wait for them
		// to finish.
		"coroWaitTime": vm.NewNumber(0.02),
	}
	vm.SetSlot(vm.Lobby, "testValues", vm.NewObject(config))
	cases := map[string]map[string]testutils.SourceTestCase{
		"coroDo": {
			"spawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues coroDoSync := true; coroDo(while(testValues coroDoSync, yield)); wait(testValues coroWaitTime); testValues coroDoCoros := Scheduler coroCount; testValues coroDoSync = false; testValues coroDoCoros`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"sideEffects": {Source: `testValues coroDoSideEffect := 0; coroDo(testValues coroDoSideEffect := 1); wait(testValues coroWaitTime); testValues coroDoSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			// TODO: test that this has the right target
		},
		"coroDoLater": {
			"spawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues coroDoLaterSync := true; coroDoLater(while(testValues coroDoLaterSync, yield)); wait(testValues coroWaitTime); testValues coroDoLaterCoros := Scheduler coroCount; testValues coroDoLaterSync = false; testValues coroDoLaterCoros`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"sideEffects": {Source: `testValues coroDoLaterSideEffect := 0; coroDoLater(testValues coroDoLaterSideEffect := 1); wait(testValues coroWaitTime); testValues coroDoLaterSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			// TODO: test that this has the right target
		},
		"coroFor": {
			"noSpawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues coroForSync := true; coroFor(while(testValues coroForSync, yield)); wait(testValues coroWaitTime); testValues coroForCoros := Scheduler coroCount; testValues coroForSync = false; testValues coroForCoros`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"noSideEffects": {Source: `testValues coroForSideEffect := 0; coroFor(testValues coroForSideEffect := 1); wait(testValues coroWaitTime); testValues coroForSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"type":          {Source: `coroFor(nil)`, Pass: testutils.PassTag(coroutine.CoroutineTag)},
			"message":       {Source: `coroFor(nil) runMessage name`, Pass: testutils.PassEqual(vm.NewString("nil"))},
			"target":        {Source: `0 coroFor(nil) runTarget`, Pass: testutils.PassIdentical(vm.Lobby)},
			"locals":        {Source: `0 coroFor(nil) runLocals`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"coroWith": {
			"noSpawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues coroWithSync := true; coroWith(while(testValues coroWithSync, yield)); wait(testValues coroWaitTime); testValues coroWithCoros := Scheduler coroCount; testValues coroWithSync = false; testValues coroWithCoros`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"noSideEffects": {Source: `testValues coroWithSideEffect := 0; coroWith(testValues coroWithSideEffect := 1); wait(testValues coroWaitTime); testValues coroWithSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"type":          {Source: `coroWith(nil)`, Pass: testutils.PassTag(coroutine.CoroutineTag)},
			"message":       {Source: `coroWith(nil) runMessage name`, Pass: testutils.PassEqual(vm.NewString("nil"))},
			"target":        {Source: `0 coroWith(nil) runTarget`, Pass: testutils.PassEqual(vm.NewNumber(0))},
			"locals":        {Source: `0 coroWith(nil) runLocals`, Pass: testutils.PassIdentical(vm.Lobby)},
		},
		"currentCoro": {
			"isCurrent": {Source: `currentCoro`, Pass: testutils.PassIdentical(vm.Coro)},
		},
		"pause": {
			"pause": {Source: `testValues pauseValue := 0; testValues pauseCoro := coroDo(testValues pauseValue = 1; Object pause; testValues pauseValue = 2); while(testValues pauseValue == 0, yield); while(Scheduler coroCount > 0, yield); testValues pauseObs := testValues pauseValue; testValues pauseCoro resume; while(testValues pauseValue < 2, yield); testValues pauseObs`, Pass: testutils.PassEqual(vm.NewNumber(1))},
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

// TestPerformSynchronous tests that multiple coroutines updating the same slot
// see each others' changes.
func TestPerformSynchronous(t *testing.T) {
	vm := testutils.VM()
	vm.SetSlot(vm.Lobby, "testValue", vm.NewNumber(0))
	defer vm.RemoveSlot(vm.Lobby, "testValue")
	r, stop := vm.DoString("100 repeat(coroDo(1000 repeat(testValue = testValue + 1))); wait(0.2); while(Scheduler coroCount > 0, yield)", "TestPerformSynchronous")
	if stop != iolang.NoStop {
		t.Errorf("%s (%v)", vm.AsString(r), stop)
	}
	x, ok := vm.GetLocalSlot(vm.Lobby, "testValue")
	if !ok {
		t.Fatal("no slot testValue on Lobby")
	}
	if x.Tag() != iolang.NumberTag {
		t.Fatalf("Lobby testValue has tag %#v, not %#v", x.Tag(), iolang.NumberTag)
	}
	if x.Value.(float64) != 100000 {
		t.Errorf("Lobby testValue has wrong value: want 100000, got %g", x.Value.(float64))
	}
}
