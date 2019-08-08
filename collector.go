package iolang

import (
	"fmt"
	"runtime"
	"time"
)

// Io has its own mark-and-sweep garbage collector that has more features than
// Go's. Because of this, the Collector interface is very different.

func (vm *VM) initCollector() {
	slots := Slots{
		"collect":   vm.NewCFunction(CollectorCollect, nil),
		"showStats": vm.NewCFunction(CollectorShowStats, nil),
		"timeUsed":  vm.NewCFunction(CollectorTimeUsed, nil),
		"type":      vm.NewString("Collector"),
	}
	vm.SetSlot(vm.Core, "Collector", vm.ObjectWith(slots))
}

// CollectorCollect is a Collector method.
//
// collect triggers a garbage collection cycle and returns the number of
// objects collected program-wide (not only in the Io VM). This is much slower
// than allowing collection to happen automatically, as the GC statistics must
// be recorded twice to retrieve the freed object count.
func CollectorCollect(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	old := stats.Frees
	runtime.GC()
	runtime.ReadMemStats(&stats)
	return vm.NewNumber(float64(stats.Frees - old)), NoStop
}

// CollectorShowStats is a Collector method.
//
// showStats prints detailed garbage collector information to standard output.
func CollectorShowStats(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	if s.NumGC > 0 {
		last := time.Unix(0, int64(s.LastGC))
		fmt.Printf("Last GC at %v (%v ago)", last, time.Since(last))
	} else {
		fmt.Print("GC has not run")
	}
	fmt.Printf(showStatsFormat,
		s.TotalAlloc, s.Mallocs,
		s.HeapAlloc, float64(s.HeapAlloc)/float64(s.TotalAlloc)*100, s.Mallocs-s.Frees,
		s.NextGC,
		s.Frees,
		s.NumGC,
		s.GCCPUFraction*100,
		s.HeapIdle,
		s.HeapInuse,
		s.StackInuse,
		s.MSpanInuse,
		s.GCSys)
	return target, NoStop
}

// CollectorTimeUsed is a Collector method.
//
// timeUsed reports the number of seconds spent in stop-the-world garbage
// collection.
func CollectorTimeUsed(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return vm.NewNumber(float64(stats.PauseTotalNs) / 1e9), NoStop
}

const showStatsFormat = `
Lifetime allocated: %d B (%d objects)
Owned allocated: %d B (%.2f%%, %d objects)
Next GC target: %d B
Freed objects: %d
Completed cycles: %d
GC CPU usage: %.6f%%
Idle heap: %d B
In-use heap spans: %d B
Stack spans: %d B
In-use mspans: %d B
GC metadata: %d B
`
