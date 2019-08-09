package iolang

import (
	"fmt"
	"gitlab.com/variadico/lctime"
	"math"
	"time"
)

// Date represents an instant in time as an Io object.
type Date struct {
	Object
	Date time.Time
}

// NewDate creates a new Date object with the given time.
func (vm *VM) NewDate(date time.Time) *Date {
	return &Date{
		Object: Object{Protos: vm.CoreProto("Date")},
		Date:   date,
	}
}

// Activate returns the date.
func (d *Date) Activate(vm *VM, target, locals, context Interface, msg *Message) (Interface, Stop) {
	return d, NoStop
}

// Clone creates a clone of the date.
func (d *Date) Clone() Interface {
	return &Date{
		Object: Object{Protos: []Interface{d}},
		Date:   d.Date,
	}
}

func (vm *VM) initDate() {
	var kind *Date
	slots := Slots{
		"+=":                vm.NewCFunction(DatePlusEq, kind),
		"-":                 vm.NewCFunction(DateMinus, kind),
		"-=":                vm.NewCFunction(DateMinusEq, kind),
		"asNumber":          vm.NewCFunction(DateAsNumber, kind),
		"asString":          vm.NewCFunction(DateAsString, kind),
		"clock":             vm.NewCFunction(DateClock, nil),
		"convertToLocal":    vm.NewCFunction(DateConvertToLocal, kind),
		"convertToLocation": vm.NewCFunction(DateConvertToLocation, kind),
		"convertToUTC":      vm.NewCFunction(DateConvertToUTC, kind),
		"copy":              vm.NewCFunction(DateCopy, kind),
		"cpuSecondsToRun":   vm.NewCFunction(DateCpuSecondsToRun, nil),
		"day":               vm.NewCFunction(DateDay, kind),
		"fromNumber":        vm.NewCFunction(DateFromNumber, kind),
		"gmtOffset":         vm.NewCFunction(DateGmtOffset, kind),
		"gmtOffsetSeconds":  vm.NewCFunction(DateGmtOffsetSeconds, kind),
		"hour":              vm.NewCFunction(DateHour, kind),
		"isDST":             vm.NewCFunction(DateIsDST, kind),
		"isPast":            vm.NewCFunction(DateIsPast, kind),
		"isValidTime":       vm.NewCFunction(DateIsValidTime, nil),
		"location":          vm.NewCFunction(DateLocation, nil),
		"minute":            vm.NewCFunction(DateMinute, kind),
		"month":             vm.NewCFunction(DateMonth, kind),
		"now":               vm.NewCFunction(DateNow, kind),
		"second":            vm.NewCFunction(DateSecond, kind),
		"secondsSince":      vm.NewCFunction(DateSecondsSince, kind),
		"secondsSinceNow":   vm.NewCFunction(DateSecondsSinceNow, kind),
		"setDay":            vm.NewCFunction(DateSetDay, kind),
		"setGmtOffset":      vm.NewCFunction(DateSetGmtOffset, kind),
		"setHour":           vm.NewCFunction(DateSetHour, kind),
		"setMinute":         vm.NewCFunction(DateSetMinute, kind),
		"setMonth":          vm.NewCFunction(DateSetMonth, kind),
		"setSecond":         vm.NewCFunction(DateSetSecond, kind),
		"setToUTC":          vm.NewCFunction(DateSetToUTC, kind),
		"setYear":           vm.NewCFunction(DateSetYear, kind),
		"type":              vm.NewString("Date"),
		"year":              vm.NewCFunction(DateYear, kind),
	}
	// isDST and isDaylightSavingsTime are distinct in Io, but they seem to
	// serve the same purpose, with the former inspecting the struct timezone
	// and the latter creating a new time instance off the timestamp to check.
	// Since we don't have a forward-facing DST concept in Go, there isn't any
	// obvious reason to have them be distinct in this implementation.
	slots["isDaylightSavingsTime"] = slots["isDST"]
	vm.SetSlot(vm.Core, "Date", &Date{Object: *vm.ObjectWith(slots), Date: time.Now()})
}

// DateAsNumber is a Date method.
//
// asNumber converts the date into seconds since 1970-01-01 00:00:00 UTC.
func DateAsNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	s := d.Date.UnixNano()
	return vm.NewNumber(float64(s) / 1e9), NoStop
}

// DateAsString is a Date method.
//
// asString converts the date to a string representation using ANSI C datetime
// formatting. See https://godoc.org/github.com/variadico/lctime for the full
// list of supported directives.
func DateAsString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	format := "%Y-%m-%d %H:%M:%S %Z"
	if len(msg.Args) > 0 {
		s, err, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return err, stop
		}
		format = s.String()
	}
	return vm.NewString(lctime.Strftime(format, d.Date)), NoStop
}

// DateClock is a Date method.
//
// clock returns the number of seconds since Io initialization as a Number.
func DateClock(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	dur := time.Since(vm.StartTime)
	return vm.NewNumber(dur.Seconds()), NoStop
}

// DateConvertToLocal is a Date method.
//
// convertToLocal converts the date to the local timezone.
func DateConvertToLocal(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	d.Date = d.Date.Local()
	return target, NoStop
}

// DateConvertToLocation is a Date method.
//
// convertToLocation converts the time to have the given IANA Time Zone
// database location, e.g. "America/New_York". See
// https://golang.org/pkg/time/#LoadLocation for more information.
func DateConvertToLocation(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// I'm providing this as an alternative to Io's Date convertToZone, because
	// that would be a lot of effort to support and less consistent.
	d := target.(*Date)
	s, aerr, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return aerr, stop
	}
	loc, err := time.LoadLocation(s.String())
	if err != nil {
		return vm.IoError(err)
	}
	d.Date = d.Date.In(loc)
	return target, NoStop
}

// DateConvertToUTC is a Date method.
//
// convertToUTC converts the date to UTC.
func DateConvertToUTC(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	d.Date = d.Date.UTC()
	return target, NoStop
}

// DateCopy is a Date method.
//
// copy sets the receiver to the same date as the argument.
func DateCopy(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	a, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return a, stop
	}
	dd, ok := a.(*Date)
	if !ok {
		return vm.RaiseException("argument 0 to copy must be Date, not " + vm.TypeName(a))
	}
	d.Date = dd.Date
	return target, NoStop
}

// DateCpuSecondsToRun is a Date method.
//
// cpuSecondsToRun returns the duration taken to evaluate its argument.
func DateCpuSecondsToRun(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	m := msg.ArgAt(0)
	t := time.Now()
	r, stop := m.Eval(vm, locals)
	if stop == ExceptionStop {
		return r, stop
	}
	dur := time.Since(t)
	return vm.NewNumber(float64(dur) / 1e9), NoStop
}

// DateDay is a Date method.
//
// day returns the day of the month of the date.
func DateDay(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Day())), NoStop
}

// DateFromNumber is a Date method.
//
// fromNumber sets the date to the date corresponding to the given number of
// seconds since the Unix epoch.
func DateFromNumber(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	d.Date = time.Unix(0, int64(n.Value*1e9))
	return target, NoStop
}

/* TODO: this. Would like to be locale-aware since our strftime is, but not
** mandatory because Io isn't.
// DateFromString is a Date method.
//
// fromString creates a date from the given string representation
func DateFromString(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {

}
*/

// DateGmtOffset is a Date method.
//
// gmtOffset returns the date's timezone offset to UTC as a string.
func DateGmtOffset(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	_, s := d.Date.Zone()
	// Go's convention is seconds east of UTC, but Io's (C's?) is minutes west.
	return vm.NewString(fmt.Sprintf("%+03d%02d", s/-3600, s/60%60)), NoStop
}

// DateGmtOffsetSeconds is a Date method.
//
// gmtOffsetSeconds returns the date's timezone offset to UTC in seconds.
func DateGmtOffsetSeconds(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	_, s := d.Date.Zone()
	return vm.NewNumber(-float64(s)), NoStop
}

// DateHour is a Date method.
//
// hour returns the hour component of the date.
func DateHour(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Hour())), NoStop
}

// DateIsDST is a Date method.
//
// isDST returns whether the date is a daylight savings time.
func DateIsDST(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	// Go doesn't have anything like this explicitly, so what we can do instead
	// is create a new time six months before and see whether it has a larger
	// UTC difference. No idea whether this will actually work, though. :)
	d := target.(*Date).Date
	y := d.Year()
	m := d.Month()
	if m < time.July {
		m += 6
		y--
	} else {
		m -= 6
	}
	dd := time.Date(y, m, d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	_, s1 := d.Zone()
	_, s2 := dd.Zone()
	return vm.IoBool(s1 > s2), NoStop
}

// DateIsPast is a Date method.
//
// isPast returns true if the date is in the past.
func DateIsPast(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.IoBool(d.Date.Before(time.Now())), NoStop
}

// DateIsValidTime is a Date method.
//
// isValidTime returns whether the given hour, minute, and second combination has
// valid values for each component.
func DateIsValidTime(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	n1, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	n2, err, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return err, stop
	}
	n3, err, stop := msg.NumberArgAt(vm, locals, 2)
	if stop != NoStop {
		return err, stop
	}
	h, m, s := n1.Value, n2.Value, n3.Value
	if h < 0 {
		h += 24
	}
	if m < 0 {
		m += 60
	}
	if s < 0 {
		s += 60
	}
	return vm.IoBool(h >= 0 && h < 24 && m >= 0 && m < 60 && s >= 0 && s < 60), NoStop
}

// DateLocation is a Date method.
//
// location returns the system's time location, either "Local" or "UTC".
func DateLocation(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	return vm.NewString(time.Local.String()), NoStop
}

// DateMinus is a Date method.
//
// - produces a Date that is before the receiver by the given Duration, or
// produces the Duration between the receiver and the given Date.
func DateMinus(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	switch dd := r.(type) {
	case *Date:
		return vm.NewDuration(d.Date.Sub(dd.Date)), NoStop
	case *Duration:
		return vm.NewDate(d.Date.Add(-dd.Value)), NoStop
	}
	return vm.RaiseException("argument 0 to - must be Date or Duration, not " + vm.TypeName(r))
}

// DateMinusEq is a Date method.
//
// -= sets the receiver to the date that is before the receiver by the given
// duration.
func DateMinusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	dur, ok := r.(*Duration)
	if !ok {
		return vm.RaiseException("argument 0 to -= must be Duration, not " + vm.TypeName(r))
	}
	d.Date = d.Date.Add(-dur.Value)
	return target, NoStop
}

// DateMinute is a Date method.
//
// minute returns the minute portion of the date.
func DateMinute(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Minute())), NoStop
}

// DateMonth is a Date method.
//
// month returns the month portion of the date.
func DateMonth(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Month())), NoStop
}

// DateNow is a Date method.
//
// now sets the date to the current local time.
func DateNow(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	d.Date = time.Now()
	return target, NoStop
}

// DatePlusEq is a Date method.
//
// += sets the receiver to the date that is after the receiver by the given
// duration.
func DatePlusEq(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return r, stop
	}
	dur, ok := r.(*Duration)
	if !ok {
		return vm.RaiseException("argument 0 to += must be Duration, not " + vm.TypeName(r))
	}
	d.Date = d.Date.Add(dur.Value)
	return target, NoStop
}

// DateSecond is a Date method.
//
// second returns the fractional number of seconds within the minute of the
// date.
func DateSecond(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Second()) + float64(d.Date.Nanosecond())/1e9), NoStop
}

// DateSecondsSince is a Date method.
//
// secondsSince returns the number of seconds between the receiver and the
// argument, i.e. receiver - argument.
func DateSecondsSince(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	v, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return v, stop
	}
	d2, ok := v.(*Date)
	if !ok {
		return vm.RaiseException("argument 0 to secondsSince must be Date, not " + vm.TypeName(v))
	}
	dur := d.Date.Sub(d2.Date)
	return vm.NewNumber(dur.Seconds()), NoStop
}

// DateSecondsSinceNow is a Date method.
//
// secondsSinceNow returns the number of seconds between now and the receiver.
func DateSecondsSinceNow(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	dur := time.Since(d.Date)
	return vm.NewNumber(dur.Seconds()), NoStop
}

// DateSetDay is a Date method.
//
// setDay sets the day of the date.
func DateSetDay(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	d.Date = time.Date(dd.Year(), dd.Month(), int(n.Value), dd.Hour(), dd.Minute(), dd.Second(), dd.Nanosecond(), dd.Location())
	return target, NoStop
}

// DateSetGmtOffset is a Date method.
//
// setGmtOffset sets the timezone of the date to the given number of minutes
// west of UTC.
func DateSetGmtOffset(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	sw := int(n.Value * -60)
	mw := sw / 60
	var loc *time.Location
	if sw == 0 {
		loc = time.FixedZone("UTC", 0)
	} else {
		loc = time.FixedZone(fmt.Sprintf("UTC%+03d%02d", mw/-60, mw%60), sw)
	}
	d.Date = time.Date(dd.Year(), dd.Month(), dd.Day(), dd.Hour(), dd.Minute(), dd.Second(), dd.Nanosecond(), loc)
	return target, NoStop
}

// DateSetHour is a Date method.
//
// setHour sets the hour of the date.
func DateSetHour(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	d.Date = time.Date(dd.Year(), dd.Month(), dd.Day(), int(n.Value), dd.Minute(), dd.Second(), dd.Nanosecond(), dd.Location())
	return target, NoStop
}

// DateSetMinute is a Date method.
//
// setMinute sets the minute of the date.
func DateSetMinute(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	d.Date = time.Date(dd.Year(), dd.Month(), dd.Day(), dd.Hour(), int(n.Value), dd.Second(), dd.Nanosecond(), dd.Location())
	return target, NoStop
}

// DateSetMonth is a Date method.
//
// setMonth sets the month of the date.
func DateSetMonth(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	d.Date = time.Date(dd.Year(), time.Month(n.Value), dd.Day(), dd.Hour(), dd.Minute(), dd.Second(), dd.Nanosecond(), dd.Location())
	return target, NoStop
}

// DateSetSecond is a Date method.
//
// setSecond sets the (fractional) second of the date.
func DateSetSecond(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	s := int(n.Value)
	ns := int((n.Value - math.Floor(n.Value)) * 1e9)
	d.Date = time.Date(dd.Year(), dd.Month(), dd.Day(), dd.Hour(), dd.Minute(), s, ns, dd.Location())
	return target, NoStop
}

// DateSetToUTC is a Date method.
//
// setToUTC sets the location of the date to UTC.
func DateSetToUTC(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	dd := d.Date
	d.Date = time.Date(dd.Year(), dd.Month(), dd.Day(), dd.Hour(), dd.Minute(), dd.Second(), dd.Nanosecond(), time.UTC)
	return target, NoStop
}

// DateSetYear is a Date method.
//
// setYear sets the year of the date.
func DateSetYear(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return err, stop
	}
	dd := d.Date
	d.Date = time.Date(int(n.Value), dd.Month(), dd.Day(), dd.Hour(), dd.Minute(), dd.Second(), dd.Nanosecond(), dd.Location())
	return target, NoStop
}

// DateYear is a Date method.
//
// year returns the year of the date.
func DateYear(vm *VM, target, locals Interface, msg *Message) (Interface, Stop) {
	d := target.(*Date)
	return vm.NewNumber(float64(d.Date.Year())), NoStop
}
