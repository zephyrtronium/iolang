//go:generate go run ../../cmd/gencore duration_init.go duration ./io
//go:generate gofmt -s -w duration_init.go

package duration

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/internal"
)

// DurationTag is the Tag for Duration objects.
const DurationTag = iolang.BasicTag("Duration")

// New creates a new Duration object with the given duration.
func New(vm *iolang.VM, d time.Duration) *iolang.Object {
	return vm.ObjectWith(nil, vm.CoreProto("Duration"), d, DurationTag)
}

// ArgAt evaluates the nth argument and returns it as a time.Duration.
// If a stop occurs during evaluation, the duration will be zero, and the stop
// status and result will be returned. If the evaluated result is not a
// Duration, the result will be zero, and an exception will be returned with an
// ExceptionStop.
func ArgAt(vm *iolang.VM, m *iolang.Message, locals *iolang.Object, n int) (time.Duration, *iolang.Object, iolang.Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == iolang.NoStop {
		v.Lock()
		d, ok := v.Value.(time.Duration)
		v.Unlock()
		if ok {
			return d, nil, iolang.NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Duration, not %s", n, m.Text, vm.TypeName(v))
		s = iolang.ExceptionStop
	}
	return 0, v, s
}

func init() {
	internal.Register(initDuration)
}

func initDuration(vm *iolang.VM) {
	slots := iolang.Slots{
		"+=":         vm.NewCFunction(plusEq, DurationTag),
		"-=":         vm.NewCFunction(minusEq, DurationTag),
		"asNumber":   vm.NewCFunction(asNumber, DurationTag),
		"asString":   vm.NewCFunction(asString, DurationTag),
		"days":       vm.NewCFunction(days, DurationTag),
		"fromNumber": vm.NewCFunction(fromNumber, DurationTag),
		"hours":      vm.NewCFunction(hours, DurationTag),
		"minutes":    vm.NewCFunction(minutes, DurationTag),
		"seconds":    vm.NewCFunction(seconds, DurationTag),
		"setDays":    vm.NewCFunction(setDays, DurationTag),
		"setHours":   vm.NewCFunction(setHours, DurationTag),
		"setMinutes": vm.NewCFunction(setMinutes, DurationTag),
		"setSeconds": vm.NewCFunction(setSeconds, DurationTag),
		"setYears":   vm.NewCFunction(setYears, DurationTag),
		"type":       vm.NewString("Duration"),
		"years":      vm.NewCFunction(years, DurationTag),
	}
	slots["totalSeconds"] = slots["asNumber"]
	internal.CoreInstall(vm, "Duration", slots, time.Duration(0), DurationTag)
	internal.Ioz(vm, coreIo, coreFiles)
}

// asNumber is a Duration method.
//
// asNumber returns the duration as the number of seconds it represents.
func asNumber(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(d.Seconds())
}

// asString is a Duration method.
//
// asString formats the duration. The format may use the following directives:
//
// 	%Y - Years, with a year defined as 60*60*24*365 seconds.
// 	%y - Four digit years.
// 	%d - Days, with a day defined as 60*60*24 seconds.
// 	%H - Hours.
// 	%M - Minutes.
// 	%S - Seconds, with six-digit fraction.
//
// The default format is "%Y years %d days %H:%M:%S". Note that the definitions
// of years and days never account for leap years or leap seconds, so it is
// probably better to avoid them.
func asString(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	// There's no way to escape % characters, and years and days are kinda
	// nonsense, but I guess it's easy to program.
	format := "%Y years %d days %H:%M:%S"
	if msg.ArgCount() > 0 {
		s, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != iolang.NoStop {
			return vm.Stop(exc, stop)
		}
		format = s
	}
	const (
		year = 365 * 24 * time.Hour
		day  = 24 * time.Hour
	)
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	rep := strings.NewReplacer(
		"%Y", fmt.Sprintf("%d", d/year),
		"%y", fmt.Sprintf("%04d", d/year),
		"%d", fmt.Sprintf("%02d", d%year/day),
		"%H", fmt.Sprintf("%02d", d%day/time.Hour),
		"%M", fmt.Sprintf("%02d", d%time.Hour/time.Minute),
		"%S", fmt.Sprintf("%.6f", float64(d%time.Minute)/float64(time.Second)))
	return vm.NewString(rep.Replace(format))
}

// days is a Duration method.
//
// days returns the number of days represented by the duration, with a day
// defined as 60*60*24 seconds, not including multiples of 365 days.
func days(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(d / (60 * 60 * 24 * time.Second) % 365))
}

// fromNumber is a Duration method.
//
// fromNumber sets the duration to the given number of seconds.
func fromNumber(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = time.Duration(n * float64(time.Second))
	target.Unlock()
	return target
}

// hours is a Duration method.
//
// hours returns the number of whole hours the duration represents, modulo 24.
func hours(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(int64(d.Hours()) % 24))
}

// minusEq is a Duration method.
//
// -= decreases this duration by the argument duration.
func minusEq(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dd, exc, stop := ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	target.Value = d - dd
	target.Unlock()
	return target
}

// minutes is a Duration method.
//
// minutes returns the number of whole minutes the duration represents, modulo
// 60.
func minutes(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(int64(d.Minutes()) % 60))
}

// plusEq is a Duration method.
//
// += increases this duration by the argument duration.
func plusEq(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dd, exc, stop := ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	target.Value = d + dd
	target.Unlock()
	return target
}

// seconds is a Duration method.
//
// seconds returns the fractional number of seconds the duration represents,
// modulo 60.
func seconds(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(math.Mod(d.Seconds(), 60))
}

// setDays is a Duration method.
//
// setDays sets the number of days the duration represents, with a day defined
// as 60*60*24 seconds. Overflow into years is allowed.
func setDays(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	cd := d / (24 * time.Hour) % 365
	delta := n - float64(cd)
	target.Value = d + time.Duration(delta*24*float64(time.Hour))
	target.Unlock()
	return target
}

// setHours is a Duration method.
//
// setHours sets the number of hours the duration represents. Overflow into
// days is allowed.
func setHours(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	ch := d / time.Hour % 24
	delta := n - float64(ch)
	target.Value = d + time.Duration(delta*float64(time.Hour))
	target.Unlock()
	return target
}

// setMinutes is a Duration method.
//
// setMinutes sets the number of minutes the duration represents. Overflow into
// hours is allowed.
func setMinutes(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	cm := d / time.Minute % 60
	delta := n - float64(cm)
	target.Value = d + time.Duration(delta*float64(time.Minute))
	target.Unlock()
	return target
}

// setSeconds is a Duration method.
//
// setSeconds sets the number of seconds the duration represents. Overflow and
// underflow are handled correctly.
func setSeconds(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	cs := d / time.Second % 60
	delta := n - float64(cs)
	target.Value = d + time.Duration(delta*float64(time.Second))
	target.Unlock()
	return target
}

// setYears is a Duration method.
//
// setYears sets the number of years the duration represents, with a year
// defined as 60*60*24*365 seconds. Overflow, underflow, and fractional values
// are handled correctly. However, because durations are represented as integer
// nanoseconds internally, this conversion is never exact.
func setYears(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	cy := d / (24 * 365 * time.Hour)
	delta := n - float64(cy)
	target.Value = d + time.Duration(delta*float64(24*365*time.Hour)) // not exact!!
	target.Unlock()
	return target
}

// years is a Duration method.
//
// years returns the number of whole years represented by the duration, with a
// year defined as 60*60*24*365 seconds.
func years(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(d / (24 * 365 * time.Hour)))
}
