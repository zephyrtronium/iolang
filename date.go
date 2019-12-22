package iolang

import (
	"fmt"
	"math"
	"time"

	"github.com/darkerbit/datesaurus"
	"gitlab.com/variadico/lctime"
)

// DateTag is the Tag for Date objects.
const DateTag = BasicTag("Date")

// NewDate creates a new Date object with the given time.
func (vm *VM) NewDate(date time.Time) *Object {
	return vm.ObjectWith(nil, vm.CoreProto("Date"), date, DateTag)
}

// DateArgAt evaluates the nth argument and returns it as a time.Time. If a
// stop occurs during evaluation, the time will be zero, and the stop status
// and result will be returned. If the evaluated result is not a Date, the
// result will be zero, and an exception will be returned with an
// ExceptionStop.
func (m *Message) DateArgAt(vm *VM, locals *Object, n int) (time.Time, *Object, Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == NoStop {
		v.Lock()
		d, ok := v.Value.(time.Time)
		v.Unlock()
		if ok {
			return d, nil, NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Date, not %s", n, m.Text, vm.TypeName(v))
		s = ExceptionStop
	}
	return time.Time{}, v, s
}

func (vm *VM) initDate() {
	slots := Slots{
		"+=":                vm.NewCFunction(DatePlusEq, DateTag),
		"-":                 vm.NewCFunction(DateMinus, DateTag),
		"-=":                vm.NewCFunction(DateMinusEq, DateTag),
		"asNumber":          vm.NewCFunction(DateAsNumber, DateTag),
		"asString":          vm.NewCFunction(DateAsString, DateTag),
		"clock":             vm.NewCFunction(DateClock, nil),
		"convertToLocal":    vm.NewCFunction(DateConvertToLocal, DateTag),
		"convertToLocation": vm.NewCFunction(DateConvertToLocation, DateTag),
		"convertToUTC":      vm.NewCFunction(DateConvertToUTC, DateTag),
		"copy":              vm.NewCFunction(DateCopy, DateTag),
		"cpuSecondsToRun":   vm.NewCFunction(DateCPUSecondsToRun, nil),
		"day":               vm.NewCFunction(DateDay, DateTag),
		"fromNumber":        vm.NewCFunction(DateFromNumber, DateTag),
		"fromString":        vm.NewCFunction(DateFromString, DateTag),
		"gmtOffset":         vm.NewCFunction(DateGmtOffset, DateTag),
		"gmtOffsetSeconds":  vm.NewCFunction(DateGmtOffsetSeconds, DateTag),
		"hour":              vm.NewCFunction(DateHour, DateTag),
		"isDST":             vm.NewCFunction(DateIsDST, DateTag),
		"isPast":            vm.NewCFunction(DateIsPast, DateTag),
		"isValidTime":       vm.NewCFunction(DateIsValidTime, nil),
		"location":          vm.NewCFunction(DateLocation, nil),
		"minute":            vm.NewCFunction(DateMinute, DateTag),
		"month":             vm.NewCFunction(DateMonth, DateTag),
		"now":               vm.NewCFunction(DateNow, DateTag),
		"second":            vm.NewCFunction(DateSecond, DateTag),
		"secondsSince":      vm.NewCFunction(DateSecondsSince, DateTag),
		"secondsSinceNow":   vm.NewCFunction(DateSecondsSinceNow, DateTag),
		"setDay":            vm.NewCFunction(DateSetDay, DateTag),
		"setGmtOffset":      vm.NewCFunction(DateSetGmtOffset, DateTag),
		"setHour":           vm.NewCFunction(DateSetHour, DateTag),
		"setMinute":         vm.NewCFunction(DateSetMinute, DateTag),
		"setMonth":          vm.NewCFunction(DateSetMonth, DateTag),
		"setSecond":         vm.NewCFunction(DateSetSecond, DateTag),
		"setToUTC":          vm.NewCFunction(DateSetToUTC, DateTag),
		"setYear":           vm.NewCFunction(DateSetYear, DateTag),
		"type":              vm.NewString("Date"),
		"year":              vm.NewCFunction(DateYear, DateTag),
	}
	// isDST and isDaylightSavingsTime are distinct in Io, but they seem to
	// serve the same purpose, with the former inspecting the struct timezone
	// and the latter creating a new time instance off the timestamp to check.
	// Since we don't have a forward-facing DST concept in Go, there isn't any
	// obvious reason to have them be distinct in this implementation.
	slots["isDaylightSavingsTime"] = slots["isDST"]
	vm.coreInstall("Date", slots, time.Now(), DateTag)
}

// DateAsNumber is a Date method.
//
// asNumber converts the date into seconds since 1970-01-01 00:00:00 UTC.
func DateAsNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	s := d.UnixNano()
	return vm.NewNumber(float64(s) / 1e9)
}

// DateAsString is a Date method.
//
// asString converts the date to a string representation using ANSI C datetime
// formatting. See https://godoc.org/github.com/variadico/lctime for the full
// list of supported directives.
func DateAsString(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	format := "%Y-%m-%d %H:%M:%S %Z"
	if len(msg.Args) > 0 {
		s, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != NoStop {
			return vm.Stop(exc, stop)
		}
		format = s
	}
	return vm.NewString(lctime.Strftime(format, d))
}

// DateClock is a Date method.
//
// clock returns the number of seconds since Io initialization as a Number.
func DateClock(vm *VM, target, locals *Object, msg *Message) *Object {
	dur := time.Since(vm.StartTime)
	return vm.NewNumber(dur.Seconds())
}

// DateConvertToLocal is a Date method.
//
// convertToLocal converts the date to the local timezone.
func DateConvertToLocal(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.Local()
	target.Unlock()
	return target
}

// DateConvertToLocation is a Date method.
//
// convertToLocation converts the time to have the given IANA Time Zone
// database location, e.g. "America/New_York". See
// https://golang.org/pkg/time/#LoadLocation for more information.
func DateConvertToLocation(vm *VM, target, locals *Object, msg *Message) *Object {
	// I'm providing this as an alternative to Io's Date convertToZone, because
	// that would be a lot of effort to support and less consistent.
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	loc, err := time.LoadLocation(s)
	if err != nil {
		return vm.IoError(err)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.In(loc)
	target.Unlock()
	return target
}

// DateConvertToUTC is a Date method.
//
// convertToUTC converts the date to UTC.
func DateConvertToUTC(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.UTC()
	target.Unlock()
	return target
}

// DateCopy is a Date method.
//
// copy sets the receiver to the same date as the argument.
func DateCopy(vm *VM, target, locals *Object, msg *Message) *Object {
	dd, exc, stop := msg.DateArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = dd
	target.Unlock()
	return target
}

// DateCPUSecondsToRun is a Date method.
//
// cpuSecondsToRun returns the duration taken to evaluate its argument.
func DateCPUSecondsToRun(vm *VM, target, locals *Object, msg *Message) *Object {
	m := msg.ArgAt(0)
	t := time.Now()
	r, stop := m.Eval(vm, locals)
	if stop == ExceptionStop {
		return vm.Stop(r, stop)
	}
	dur := time.Since(t)
	return vm.NewNumber(float64(dur) / 1e9)
}

// DateDay is a Date method.
//
// day returns the day of the month of the date.
func DateDay(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Day()))
}

// DateFromNumber is a Date method.
//
// fromNumber sets the date to the date corresponding to the given number of
// seconds since the Unix epoch.
func DateFromNumber(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	target.Value = time.Unix(0, int64(n*1e9))
	target.Unlock()
	return target
}

// DateFromString is a Date method.
//
// fromString creates a date from the given string representation.
func DateFromString(vm *VM, target, locals *Object, msg *Message) *Object {
	str, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}

	format, err, stop := msg.StringArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}

	var longForm = datesaurus.Get(format)

	v, r := time.Parse(longForm, str)
	if r != nil {
		return vm.RaiseExceptionf("argument 0 to - must be a valid date string (%s)", longForm)
	}

	target.Lock()
	target.Value = v
	target.Unlock()

	return target
}

// DateGmtOffset is a Date method.
//
// gmtOffset returns the date's timezone offset to UTC as a string.
func DateGmtOffset(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	_, s := d.Zone()
	// Go's convention is seconds east of UTC, but Io's (C's?) is minutes west.
	return vm.NewString(fmt.Sprintf("%+03d%02d", s/-3600, s/60%60))
}

// DateGmtOffsetSeconds is a Date method.
//
// gmtOffsetSeconds returns the date's timezone offset to UTC in seconds.
func DateGmtOffsetSeconds(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	_, s := d.Zone()
	return vm.NewNumber(-float64(s))
}

// DateHour is a Date method.
//
// hour returns the hour component of the date.
func DateHour(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Hour()))
}

// DateIsDST is a Date method.
//
// isDST returns whether the date is a daylight savings time.
func DateIsDST(vm *VM, target, locals *Object, msg *Message) *Object {
	// Go doesn't have anything like this explicitly, so what we can do instead
	// is create a new time six months before and see whether it has a larger
	// UTC difference. No idea whether this will actually work, though. :)
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
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
	return vm.IoBool(s1 > s2)
}

// DateIsPast is a Date method.
//
// isPast returns true if the date is in the past.
func DateIsPast(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.IoBool(d.Before(time.Now()))
}

// DateIsValidTime is a Date method.
//
// isValidTime returns whether the given hour, minute, and second combination has
// valid values for each component.
func DateIsValidTime(vm *VM, target, locals *Object, msg *Message) *Object {
	h, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	m, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	s, exc, stop := msg.NumberArgAt(vm, locals, 2)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	if h < 0 {
		h += 24
	}
	if m < 0 {
		m += 60
	}
	if s < 0 {
		s += 60
	}
	return vm.IoBool(h >= 0 && h < 24 && m >= 0 && m < 60 && s >= 0 && s < 60)
}

// DateLocation is a Date method.
//
// location returns the system's time location, either "Local" or "UTC".
func DateLocation(vm *VM, target, locals *Object, msg *Message) *Object {
	return vm.NewString(time.Local.String())
}

// DateMinus is a Date method.
//
// - produces a Date that is before the receiver by the given Duration, or
// produces the Duration between the receiver and the given Date.
func DateMinus(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(r, stop)
	}
	switch dd := r.Value.(type) {
	case time.Time:
		return vm.NewDuration(d.Sub(dd))
	case time.Duration:
		return vm.NewDate(d.Add(-dd))
	}
	return vm.RaiseExceptionf("argument 0 to - must be Date or Duration, not %s", vm.TypeName(r))
}

// DateMinusEq is a Date method.
//
// -= sets the receiver to the date that is before the receiver by the given
// duration.
func DateMinusEq(vm *VM, target, locals *Object, msg *Message) *Object {
	dur, exc, stop := msg.DurationArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.Add(-dur)
	target.Unlock()
	return target
}

// DateMinute is a Date method.
//
// minute returns the minute portion of the date.
func DateMinute(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Minute()))
}

// DateMonth is a Date method.
//
// month returns the month portion of the date.
func DateMonth(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Month()))
}

// DateNow is a Date method.
//
// now sets the date to the current local time.
func DateNow(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	target.Value = time.Now()
	target.Unlock()
	return target
}

// DatePlusEq is a Date method.
//
// += sets the receiver to the date that is after the receiver by the given
// duration.
func DatePlusEq(vm *VM, target, locals *Object, msg *Message) *Object {
	dur, exc, stop := msg.DurationArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = target.Value.(time.Time).Add(dur)
	return target
}

// DateSecond is a Date method.
//
// second returns the fractional number of seconds within the minute of the
// date.
func DateSecond(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Second()) + float64(d.Nanosecond())/1e9)
}

// DateSecondsSince is a Date method.
//
// secondsSince returns the number of seconds between the receiver and the
// argument, i.e. receiver - argument.
func DateSecondsSince(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	dd, exc, stop := msg.DateArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(exc, stop)
	}
	dur := d.Sub(dd)
	return vm.NewNumber(dur.Seconds())
}

// DateSecondsSinceNow is a Date method.
//
// secondsSinceNow returns the number of seconds between now and the receiver.
func DateSecondsSinceNow(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	dur := time.Since(d)
	return vm.NewNumber(dur.Seconds())
}

// DateSetDay is a Date method.
//
// setDay sets the day of the date.
func DateSetDay(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), int(n), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// DateSetGmtOffset is a Date method.
//
// setGmtOffset sets the timezone of the date to the given number of minutes
// west of UTC.
func DateSetGmtOffset(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	sw := int(n * -60)
	mw := sw / 60
	var loc *time.Location
	if sw == 0 {
		loc = time.FixedZone("UTC", 0)
	} else {
		loc = time.FixedZone(fmt.Sprintf("UTC%+03d%02d", mw/-60, mw%60), sw)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), loc)
	target.Unlock()
	return target
}

// DateSetHour is a Date method.
//
// setHour sets the hour of the date.
func DateSetHour(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), int(n), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// DateSetMinute is a Date method.
//
// setMinute sets the minute of the date.
func DateSetMinute(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), int(n), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// DateSetMonth is a Date method.
//
// setMonth sets the month of the date.
func DateSetMonth(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), time.Month(n), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// DateSetSecond is a Date method.
//
// setSecond sets the (fractional) second of the date.
func DateSetSecond(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	s := int(n)
	ns := int((n - math.Floor(n)) * 1e9)
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), s, ns, d.Location())
	target.Unlock()
	return target
}

// DateSetToUTC is a Date method.
//
// setToUTC sets the location of the date to UTC.
func DateSetToUTC(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), time.UTC)
	target.Unlock()
	return target
}

// DateSetYear is a Date method.
//
// setYear sets the year of the date.
func DateSetYear(vm *VM, target, locals *Object, msg *Message) *Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(int(n), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// DateYear is a Date method.
//
// year returns the year of the date.
func DateYear(vm *VM, target, locals *Object, msg *Message) *Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Year()))
}
