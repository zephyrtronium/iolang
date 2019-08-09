package iolang

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// A Duration is an object representing a length of time.
type Duration struct {
	Object
	Value time.Duration
}

// NewDuration creates a new Duration object with the given duration.
func (vm *VM) NewDuration(d time.Duration) *Duration {
	return &Duration{
		Object: Object{Protos: vm.CoreProto("Duration")},
		Value:  d,
	}
}

// Activate returns the duration.
func (d *Duration) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return d, NoStop
}

// Clone creates a clone of the duration with the same value.
func (d *Duration) Clone() Interface {
	return &Duration{
		Object: Object{Protos: []Interface{d}},
		Value:  d.Value,
	}
}

func (vm *VM) initDuration() {
	var kind *Duration
	slots := Slots{
		"+=":         vm.NewCFunction(DurationPlusEq, kind),
		"-=":         vm.NewCFunction(DurationMinusEq, kind),
		"asNumber":   vm.NewCFunction(DurationAsNumber, kind),
		"asString":   vm.NewCFunction(DurationAsString, kind),
		"days":       vm.NewCFunction(DurationDays, kind),
		"fromNumber": vm.NewCFunction(DurationFromNumber, kind),
		"hours":      vm.NewCFunction(DurationHours, kind),
		"minutes":    vm.NewCFunction(DurationMinutes, kind),
		"seconds":    vm.NewCFunction(DurationSeconds, kind),
		"setDays":    vm.NewCFunction(DurationSetDays, kind),
		"setHours":   vm.NewCFunction(DurationSetHours, kind),
		"setMinutes": vm.NewCFunction(DurationSetMinutes, kind),
		"setSeconds": vm.NewCFunction(DurationSetSeconds, kind),
		"setYears":   vm.NewCFunction(DurationSetYears, kind),
		"type":       vm.NewString("Duration"),
		"years":      vm.NewCFunction(DurationYears, kind),
	}
	slots["totalSeconds"] = slots["asNumber"]
	vm.SetSlot(vm.Core, "Duration", &Duration{Object: *vm.ObjectWith(slots)})
}

// DurationAsNumber is a Duration method.
//
// asNumber returns the duration as the number of seconds it represents.
func DurationAsNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(d.Value.Seconds()), NoStop
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
func DurationAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	// There's no way to escape % characters, and years and days are kinda
	// nonsense, but I guess it's easy to program.
	format := "%Y years %d days %H:%M:%S"
	if msg.ArgCount() > 0 {
		s, err, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		format = s.String()
	}
	const (
		year = 365 * 24 * time.Hour
		day  = 24 * time.Hour
	)
	rep := strings.NewReplacer(
		"%Y", fmt.Sprintf("%d", d.Value/year),
		"%y", fmt.Sprintf("%04d", d.Value/year),
		"%d", fmt.Sprintf("%02d", d.Value%year/day),
		"%H", fmt.Sprintf("%02d", d.Value%day/time.Hour),
		"%M", fmt.Sprintf("%02d", d.Value%time.Hour/time.Minute),
		"%S", fmt.Sprintf("%.6f", float64(d.Value%time.Minute)/float64(time.Second)))
	return vm.NewString(rep.Replace(format)), NoStop
}

// DurationDays is a Duration method.
//
// days returns the number of days represented by the duration, with a day
// defined as 60*60*24 seconds, not including multiples of 365 days.
func DurationDays(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(float64(d.Value / (60 * 60 * 24 * time.Second) % 365)), NoStop
}

// DurationFromNumber is a Duration method.
//
// fromNumber sets the duration to the given number of seconds.
func DurationFromNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	d.Value = time.Duration(n.Value * float64(time.Second))
	return target, NoStop
}

// DurationHours is a Duration method.
//
// hours returns the number of whole hours the duration represents, modulo 24.
func DurationHours(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(float64(int64(d.Value.Hours()) % 24)), NoStop
}

// DurationMinusEq is a Duration method.
//
// -= decreases this duration by the argument duration.
func DurationMinusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	dd, ok := r.(*Duration)
	if !ok {
		return vm.RaiseException("argument 0 to -= must be Duration, not " + vm.TypeName(r))
	}
	d.Value -= dd.Value
	return target, NoStop
}

// DurationMinutes is a Duration method.
//
// minutes returns the number of whole minutes the duration represents, modulo
// 60.
func DurationMinutes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(float64(int64(d.Value.Minutes()) % 60)), NoStop
}

// DurationPlusEq is a Duration method.
//
// += increases this duration by the argument duration.
func DurationPlusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, NoStop
	}
	dd, ok := r.(*Duration)
	if !ok {
		return vm.RaiseException("argument 0 to += must be Duration, not " + vm.TypeName(r))
	}
	d.Value += dd.Value
	return target, NoStop
}

// DurationSeconds is a Duration method.
//
// seconds returns the fractional number of seconds the duration represents,
// modulo 60.
func DurationSeconds(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(math.Mod(d.Value.Seconds(), 60)), NoStop
}

// DurationSetDays is a Duration method.
//
// setDays sets the number of days the duration represents, with a day defined
// as 60*60*24 seconds. Overflow into years is allowed.
func DurationSetDays(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	cd := d.Value / (24 * time.Hour) % 365
	delta := n.Value - float64(cd)
	d.Value += time.Duration(delta * 24 * float64(time.Hour))
	return target, NoStop
}

// DurationSetHours is a Duration method.
//
// setHours sets the number of hours the duration represents. Overflow into
// days is allowed.
func DurationSetHours(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	ch := d.Value / time.Hour % 24
	delta := n.Value - float64(ch)
	d.Value += time.Duration(delta * float64(time.Hour))
	return target, NoStop
}

// DurationSetMinutes is a Duration method.
//
// setMinutes sets the number of minutes the duration represents. Overflow into
// hours is allowed.
func DurationSetMinutes(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	cm := d.Value / time.Minute % 60
	delta := n.Value - float64(cm)
	d.Value += time.Duration(delta * float64(time.Minute))
	return target, NoStop
}

// DurationSetSeconds is a Duration method.
//
// setSeconds sets the number of seconds the duration represents. Overflow and
// underflow are handled correctly.
func DurationSetSeconds(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	cs := d.Value / time.Second % 60
	delta := n.Value - float64(cs)
	d.Value += time.Duration(delta * float64(time.Second))
	return target, NoStop
}

// DurationSetYears is a Duration method.
//
// setYears sets the number of years the duration represents, with a year
// defined as 60*60*24*365 seconds. Overflow, underflow, and fractional values
// are handled correctly. However, because durations are represented as integer
// nanoseconds internally, this conversion is never exact.
func DurationSetYears(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	cy := d.Value / (24 * 365 * time.Hour)
	delta := n.Value - float64(cy)
	d.Value += time.Duration(delta * float64(24*365*time.Hour)) // not exact!!
	return target, NoStop
}

// DurationYears is a Duration method.
//
// years returns the number of whole years represented by the duration, with a
// year defined as 60*60*24*365 seconds.
func DurationYears(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Duration)
	return vm.NewNumber(float64(d.Value / (24 * 365 * time.Hour))), NoStop
}
