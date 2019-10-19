package iolang

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DurationTag is the Tag for Duration objects.
const DurationTag = BasicTag("Duration")

// NewDuration creates a new Duration object with the given duration.
func (vm *VM) NewDuration(d time.Duration) *Object {
	return &Object{
		Protos: vm.CoreProto("Duration"),
		Value:  d,
		Tag:    DurationTag,
	}
}

// DurationArgAt evaluates the nth argument and returns it as a time.Duration.
// If a stop occurs during evaluation, the duration will be zero, and the stop
// status and result will be returned. If the evaluated result is not a
// Duration, the result will be zero, and an exception will be returned with an
// ExceptionStop.
func (m *Message) DurationArgAt(vm *VM, locals *Object, n int) (time.Duration, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		d, ok := v.Value.(time.Duration)
		v.Unlock()
		if ok {
			return d, nil, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Duration, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return 0, v, s
}

func (vm *VM) initDuration() {
	slots := Slots{
		"+=":         vm.NewCFunction(DurationPlusEq, DurationTag),
		"-=":         vm.NewCFunction(DurationMinusEq, DurationTag),
		"asNumber":   vm.NewCFunction(DurationAsNumber, DurationTag),
		"asString":   vm.NewCFunction(DurationAsString, DurationTag),
		"days":       vm.NewCFunction(DurationDays, DurationTag),
		"fromNumber": vm.NewCFunction(DurationFromNumber, DurationTag),
		"hours":      vm.NewCFunction(DurationHours, DurationTag),
		"minutes":    vm.NewCFunction(DurationMinutes, DurationTag),
		"seconds":    vm.NewCFunction(DurationSeconds, DurationTag),
		"setDays":    vm.NewCFunction(DurationSetDays, DurationTag),
		"setHours":   vm.NewCFunction(DurationSetHours, DurationTag),
		"setMinutes": vm.NewCFunction(DurationSetMinutes, DurationTag),
		"setSeconds": vm.NewCFunction(DurationSetSeconds, DurationTag),
		"setYears":   vm.NewCFunction(DurationSetYears, DurationTag),
		"type":       vm.NewString("Duration"),
		"years":      vm.NewCFunction(DurationYears, DurationTag),
	}
	slots["totalSeconds"] = slots["asNumber"]
	vm.Core.SetSlot("Duration", &Object{
		Slots:  slots,
		Protos: []*Object{vm.BaseObject},
		Tag:    DurationTag,
	})
}

// DurationAsNumber is a Duration method.
//
// asNumber returns the duration as the number of seconds it represents.
func DurationAsNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(d.Seconds())
}

// DurationAsString is a Duration method.
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
func DurationAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	// There's no way to escape % characters, and years and days are kinda
	// nonsense, but I guess it's easy to program.
	format := "%Y years %d days %H:%M:%S"
	if msg.ArgCount() > 0 {
		s, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
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

// DurationDays is a Duration method.
//
// days returns the number of days represented by the duration, with a day
// defined as 60*60*24 seconds, not including multiples of 365 days.
func DurationDays(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(d / (60 * 60 * 24 * time.Second) % 365))
}

// DurationFromNumber is a Duration method.
//
// fromNumber sets the duration to the given number of seconds.
func DurationFromNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = time.Duration(n * float64(time.Second))
	target.Unlock()
	return target
}

// DurationHours is a Duration method.
//
// hours returns the number of whole hours the duration represents, modulo 24.
func DurationHours(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(int64(d.Hours()) % 24))
}

// DurationMinusEq is a Duration method.
//
// -= decreases this duration by the argument duration.
func DurationMinusEq(vm *VM, target, locals *Object, msg *Message) *Object {
	dd, exc, stop := msg.DurationArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	target.Value = d - dd
	target.Unlock()
	return target
}

// DurationMinutes is a Duration method.
//
// minutes returns the number of whole minutes the duration represents, modulo
// 60.
func DurationMinutes(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(int64(d.Minutes()) % 60))
}

// DurationPlusEq is a Duration method.
//
// += increases this duration by the argument duration.
func DurationPlusEq(vm *VM, target, locals *Object, msg *Message) *Object {
	dd, exc, stop := msg.DurationArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Duration)
	target.Value = d + dd
	target.Unlock()
	return target
}

// DurationSeconds is a Duration method.
//
// seconds returns the fractional number of seconds the duration represents,
// modulo 60.
func DurationSeconds(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(math.Mod(d.Seconds(), 60))
}

// DurationSetDays is a Duration method.
//
// setDays sets the number of days the duration represents, with a day defined
// as 60*60*24 seconds. Overflow into years is allowed.
func DurationSetDays(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
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

// DurationSetHours is a Duration method.
//
// setHours sets the number of hours the duration represents. Overflow into
// days is allowed.
func DurationSetHours(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
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

// DurationSetMinutes is a Duration method.
//
// setMinutes sets the number of minutes the duration represents. Overflow into
// hours is allowed.
func DurationSetMinutes(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
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

// DurationSetSeconds is a Duration method.
//
// setSeconds sets the number of seconds the duration represents. Overflow and
// underflow are handled correctly.
func DurationSetSeconds(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
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

// DurationSetYears is a Duration method.
//
// setYears sets the number of years the duration represents, with a year
// defined as 60*60*24*365 seconds. Overflow, underflow, and fractional values
// are handled correctly. However, because durations are represented as integer
// nanoseconds internally, this conversion is never exact.
func DurationSetYears(vm *VM, target, locals *Object, msg *Message) *Object {
	n, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
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

// DurationYears is a Duration method.
//
// years returns the number of whole years represented by the duration, with a
// year defined as 60*60*24*365 seconds.
func DurationYears(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Duration)
	target.Unlock()
	return vm.NewNumber(float64(d / (24 * 365 * time.Hour)))
}
