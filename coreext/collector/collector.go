package collector

import (
	"fmt"
	"runtime"
	"time"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"
)

// Io has its own mark-and-sweep garbage collector that has more features than
// Go's. Because of this, the Collector interface is very different.

func init() {
	internal.Register(initCollector)
}

func initCollector(vm *iolang.VM) {
	slots := iolang.Slots{
		"collect":   vm.NewCFunction(collectorCollect, nil),
		"showStats": vm.NewCFunction(collectorShowStats, nil),
		"timeUsed":  vm.NewCFunction(collectorTimeUsed, nil),
		"type":      vm.NewString("Collector"),
	}
	internal.CoreInstall(vm, "Collector", slots, nil, nil)
}

// collectorCollect is a Collector method.
//
// collect triggers a garbage collection cycle and returns the number of
// objects collected program-wide (not only in the Io VM). This is much slower
// than allowing collection to happen automatically, as the GC statistics must
// be recorded twice to retrieve the freed object count.
func collectorCollect(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	old := stats.Frees
	runtime.GC()
	runtime.ReadMemStats(&stats)
	return vm.NewNumber(float64(stats.Frees - old))
}

// collectorShowStats is a Collector method.
//
// showStats prints detailed garbage collector information to standard output.
func collectorShowStats(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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
	return target
}

// collectorTimeUsed is a Collector method.
//
// timeUsed reports the number of seconds spent in stop-the-world garbage
// collection.
func collectorTimeUsed(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return vm.NewNumber(float64(stats.PauseTotalNs) / 1e9)
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
