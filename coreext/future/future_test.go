package future_test

import (
	"testing"

	"github.com/zephyrtronium/iolang"
	_ "github.com/zephyrtronium/iolang/coreext/future" // side effects
	"github.com/zephyrtronium/iolang/testutils"
)

func TestRegister(t *testing.T) {
	// Coroutine is a dependency.
	testutils.CheckNewSlots(t, testutils.VM().Core, []string{"Coroutine", "Future"})
}

func TestObjectSlots(t *testing.T) {
	vm := testutils.VM()
	slots := []string{
		"@",
		"@@",
		"asyncSend",
		"futureSend",
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
		"asyncSend": {
			"spawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues asyncSendSync := true; asyncSend(while(testValues asyncSendSync, yield)); wait(testValues coroWaitTime); testValues asyncSendCoros := Scheduler coroCount; testValues asyncSendSync = false; testValues asyncSendCoros`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"sideEffects": {Source: `testValues asyncSendSideEffect := 0; asyncSend(Lobby testValues asyncSendSideEffect = 1); wait(testValues coroWaitTime); testValues asyncSendSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"empty":       {Source: `asyncSend`, Pass: testutils.PassFailure()},
		},
		"futureSend": {
			"result":      {Source: `futureSend(1) isNil not`, Pass: testutils.PassIdentical(vm.True)},
			"evaluates":   {Source: `futureSend(1) + 1`, Pass: testutils.PassEqual(vm.NewNumber(2))},
			"spawns":      {Source: `while(Scheduler coroCount > 0, yield); testValues futureSendSync := true; futureSend(while(testValues futureSendSync, yield)); wait(testValues coroWaitTime); testValues futureSendCoros := Scheduler coroCount; testValues futureSendSync = false; testValues futureSendCoros`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"sideEffects": {Source: `testValues futureSendSideEffect := 0; futureSend(Lobby testValues futureSendSideEffect = 1); wait(testValues coroWaitTime); testValues futureSendSideEffect`, Pass: testutils.PassEqual(vm.NewNumber(1))},
			"empty":       {Source: `futureSend`, Pass: testutils.PassFailure()},
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
