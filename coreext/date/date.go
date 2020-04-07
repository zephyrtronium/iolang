//go:generate go run ../../cmd/gencore date_init.go date ./io
//go:generate gofmt -s -w date_init.go

package date

import (
	"fmt"
	"math"
	"time"

	"github.com/zephyrtronium/iolang"
	"github.com/zephyrtronium/iolang/coreext/duration"
	"github.com/zephyrtronium/iolang/internal"

	"gitlab.com/variadico/lctime"
)

// DateTag is the Tag for Date objects.
const DateTag = iolang.BasicTag("Date")

// New creates a new Date object with the given time.
func New(vm *iolang.VM, date time.Time) *iolang.Object {
	return vm.ObjectWith(nil, vm.CoreProto("Date"), date, DateTag)
}

// ArgAt evaluates the nth argument and returns it as a time.Time. If a
// stop occurs during evaluation, the time will be zero, and the stop status
// and result will be returned. If the evaluated result is not a Date, the
// result will be zero, and an exception will be returned with an
// ExceptionStop.
func ArgAt(vm *iolang.VM, m *iolang.Message, locals *iolang.Object, n int) (time.Time, *iolang.Object, iolang.Stop) {
	v, s := m.EvalArgAt(vm, locals, n)
	if s == iolang.NoStop {
		v.Lock()
		d, ok := v.Value.(time.Time)
		v.Unlock()
		if ok {
			return d, nil, iolang.NoStop
		}
		// Not the expected type, so return an error.
		v = vm.NewExceptionf("argument %d to %s must be Date, not %s", n, m.Text, vm.TypeName(v))
		s = iolang.ExceptionStop
	}
	return time.Time{}, v, s
}

func init() {
	internal.Register(initDate)
}

func initDate(vm *iolang.VM) {
	slots := iolang.Slots{
		"+=":                vm.NewCFunction(plusEq, DateTag),
		"-":                 vm.NewCFunction(minus, DateTag),
		"-=":                vm.NewCFunction(minusEq, DateTag),
		"asNumber":          vm.NewCFunction(asNumber, DateTag),
		"asString":          vm.NewCFunction(asString, DateTag),
		"clock":             vm.NewCFunction(clock, nil),
		"convertToLocal":    vm.NewCFunction(convertToLocal, DateTag),
		"convertToLocation": vm.NewCFunction(convertToLocation, DateTag),
		"convertToUTC":      vm.NewCFunction(convertToUTC, DateTag),
		"copy":              vm.NewCFunction(dateCopy, DateTag),
		"cpuSecondsToRun":   vm.NewCFunction(cpuSecondsToRun, nil),
		"day":               vm.NewCFunction(day, DateTag),
		"fromNumber":        vm.NewCFunction(fromNumber, DateTag),
		"fromString":        vm.NewCFunction(fromString, DateTag),
		"gmtOffset":         vm.NewCFunction(gmtOffset, DateTag),
		"gmtOffsetSeconds":  vm.NewCFunction(gmtOffsetSeconds, DateTag),
		"hour":              vm.NewCFunction(hour, DateTag),
		"isDST":             vm.NewCFunction(isDST, DateTag),
		"isPast":            vm.NewCFunction(isPast, DateTag),
		"isValidTime":       vm.NewCFunction(isValidTime, nil),
		"location":          vm.NewCFunction(location, nil),
		"minute":            vm.NewCFunction(minute, DateTag),
		"month":             vm.NewCFunction(month, DateTag),
		"now":               vm.NewCFunction(now, DateTag),
		"second":            vm.NewCFunction(second, DateTag),
		"secondsSince":      vm.NewCFunction(secondsSince, DateTag),
		"secondsSinceNow":   vm.NewCFunction(secondsSinceNow, DateTag),
		"setDay":            vm.NewCFunction(setDay, DateTag),
		"setGmtOffset":      vm.NewCFunction(setGmtOffset, DateTag),
		"setHour":           vm.NewCFunction(setHour, DateTag),
		"setMinute":         vm.NewCFunction(setMinute, DateTag),
		"setMonth":          vm.NewCFunction(setMonth, DateTag),
		"setSecond":         vm.NewCFunction(setSecond, DateTag),
		"setToUTC":          vm.NewCFunction(setToUTC, DateTag),
		"setYear":           vm.NewCFunction(setYear, DateTag),
		"type":              vm.NewString("Date"),
		"year":              vm.NewCFunction(year, DateTag),
	}
	// isDST and isDaylightSavingsTime are distinct in Io, but they seem to
	// serve the same purpose, with the former inspecting the struct timezone
	// and the latter creating a new time instance off the timestamp to check.
	// Since we don't have a forward-facing DST concept in Go, there isn't any
	// obvious reason to have them be distinct in this implementation.
	slots["isDaylightSavingsTime"] = slots["isDST"]
	internal.CoreInstall(vm, "Date", slots, time.Now(), DateTag)
	internal.Ioz(vm, coreIo, coreFiles)
}

// asNumber is a Date method.
//
// asNumber converts the date into seconds since 1970-01-01 00:00:00 UTC.
func asNumber(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	s := d.UnixNano()
	return vm.NewNumber(float64(s) / 1e9)
}

// asString is a Date method.
//
// asString converts the date to a string representation using ANSI C datetime
// formatting. See https://godoc.org/github.com/variadico/lctime for the full
// list of supported directives.
func asString(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	format := "%Y-%m-%d %H:%M:%S %Z"
	if len(msg.Args) > 0 {
		s, exc, stop := msg.StringArgAt(vm, locals, 0)
		if stop != iolang.NoStop {
			return vm.Stop(exc, stop)
		}
		format = s
	}
	return vm.NewString(lctime.Strftime(format, d))
}

// clock is a Date method.
//
// clock returns the number of seconds since Io initialization as a Number.
func clock(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dur := time.Since(vm.StartTime)
	return vm.NewNumber(dur.Seconds())
}

// convertToLocal is a Date method.
//
// convertToLocal converts the date to the local timezone.
func convertToLocal(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.Local()
	target.Unlock()
	return target
}

// convertToLocation is a Date method.
//
// convertToLocation converts the time to have the given IANA Time Zone
// database location, e.g. "America/New_York". See
// https://golang.org/pkg/time/#LoadLocation for more information.
func convertToLocation(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	// I'm providing this as an alternative to Io's Date convertToZone, because
	// that would be a lot of effort to support and less consistent.
	s, exc, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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

// convertToUTC is a Date method.
//
// convertToUTC converts the date to UTC.
func convertToUTC(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.UTC()
	target.Unlock()
	return target
}

// dateCopy is a Date method.
//
// copy sets the receiver to the same date as the argument.
func dateCopy(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dd, exc, stop := ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = dd
	target.Unlock()
	return target
}

// cpuSecondsToRun is a Date method.
//
// cpuSecondsToRun returns the duration taken to evaluate its argument.
func cpuSecondsToRun(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	m := msg.ArgAt(0)
	t := time.Now()
	r, stop := m.Eval(vm, locals)
	if stop == iolang.ExceptionStop {
		return vm.Stop(r, stop)
	}
	dur := time.Since(t)
	return vm.NewNumber(float64(dur) / 1e9)
}

// day is a Date method.
//
// day returns the day of the month of the date.
func day(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Day()))
}

// fromNumber is a Date method.
//
// fromNumber sets the date to the date corresponding to the given number of
// seconds since the Unix epoch.
func fromNumber(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	target.Value = time.Unix(0, int64(n*1e9))
	target.Unlock()
	return target
}

// fromString is a Date method.
//
// fromString creates a date from the given string representation.
func fromString(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	str, err, stop := msg.StringArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}

	format, err, stop := msg.StringArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}

	longDate := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.FixedZone("MST", -7*60*60))
	longForm := lctime.Strftime(format, longDate)

	v, r := time.Parse(longForm, str)
	if r != nil {
		return vm.RaiseExceptionf("argument 0 to - must be a valid date string (%s)", longForm)
	}

	target.Lock()
	target.Value = v
	target.Unlock()

	return target
}

// gmtOffset is a Date method.
//
// gmtOffset returns the date's timezone offset to UTC as a string.
func gmtOffset(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	_, s := d.Zone()
	// Go's convention is seconds east of UTC, but Io's (C's?) is minutes west.
	return vm.NewString(fmt.Sprintf("%+03d%02d", s/-3600, s/60%60))
}

// gmtOffsetSeconds is a Date method.
//
// gmtOffsetSeconds returns the date's timezone offset to UTC in seconds.
func gmtOffsetSeconds(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	_, s := d.Zone()
	return vm.NewNumber(-float64(s))
}

// hour is a Date method.
//
// hour returns the hour component of the date.
func hour(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Hour()))
}

// isDST is a Date method.
//
// isDST returns whether the date is a daylight savings time.
func isDST(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
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

// isPast is a Date method.
//
// isPast returns true if the date is in the past.
func isPast(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.IoBool(d.Before(time.Now()))
}

// isValidTime is a Date method.
//
// isValidTime returns whether the given hour, minute, and second combination has
// valid values for each component.
func isValidTime(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	h, exc, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	m, exc, stop := msg.NumberArgAt(vm, locals, 1)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	s, exc, stop := msg.NumberArgAt(vm, locals, 2)
	if stop != iolang.NoStop {
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

// location is a Date method.
//
// location returns the system's time location, either "Local" or "UTC".
func location(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	return vm.NewString(time.Local.String())
}

// minus is a Date method.
//
// - produces a Date that is before the receiver by the given Duration, or
// produces the Duration between the receiver and the given Date.
func minus(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	r, stop := msg.EvalArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(r, stop)
	}
	switch dd := r.Value.(type) {
	case time.Time:
		return duration.New(vm, d.Sub(dd))
	case time.Duration:
		return New(vm, d.Add(-dd))
	}
	return vm.RaiseExceptionf("argument 0 to - must be Date or Duration, not %s", vm.TypeName(r))
}

// minusEq is a Date method.
//
// -= sets the receiver to the date that is before the receiver by the given
// duration.
func minusEq(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dur, exc, stop := duration.ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = d.Add(-dur)
	target.Unlock()
	return target
}

// minute is a Date method.
//
// minute returns the minute portion of the date.
func minute(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Minute()))
}

// month is a Date method.
//
// month returns the month portion of the date.
func month(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Month()))
}

// now is a Date method.
//
// now sets the date to the current local time.
func now(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	target.Value = time.Now()
	target.Unlock()
	return target
}

// plusEq is a Date method.
//
// += sets the receiver to the date that is after the receiver by the given
// duration.
func plusEq(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	dur, exc, stop := duration.ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	target.Lock()
	target.Value = target.Value.(time.Time).Add(dur)
	return target
}

// second is a Date method.
//
// second returns the fractional number of seconds within the minute of the
// date.
func second(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Second()) + float64(d.Nanosecond())/1e9)
}

// secondsSince is a Date method.
//
// secondsSince returns the number of seconds between the receiver and the
// argument, i.e. receiver - argument.
func secondsSince(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	dd, exc, stop := ArgAt(vm, msg, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(exc, stop)
	}
	dur := d.Sub(dd)
	return vm.NewNumber(dur.Seconds())
}

// secondsSinceNow is a Date method.
//
// secondsSinceNow returns the number of seconds between now and the receiver.
func secondsSinceNow(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	dur := time.Since(d)
	return vm.NewNumber(dur.Seconds())
}

// setDay is a Date method.
//
// setDay sets the day of the date.
func setDay(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), int(n), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// setGmtOffset is a Date method.
//
// setGmtOffset sets the timezone of the date to the given number of minutes
// west of UTC.
func setGmtOffset(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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

// setHour is a Date method.
//
// setHour sets the hour of the date.
func setHour(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), int(n), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// setMinute is a Date method.
//
// setMinute sets the minute of the date.
func setMinute(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), int(n), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// setMonth is a Date method.
//
// setMonth sets the month of the date.
func setMonth(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), time.Month(n), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// setSecond is a Date method.
//
// setSecond sets the (fractional) second of the date.
func setSecond(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
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

// setToUTC is a Date method.
//
// setToUTC sets the location of the date to UTC.
func setToUTC(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), time.UTC)
	target.Unlock()
	return target
}

// setYear is a Date method.
//
// setYear sets the year of the date.
func setYear(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	n, err, stop := msg.NumberArgAt(vm, locals, 0)
	if stop != iolang.NoStop {
		return vm.Stop(err, stop)
	}
	target.Lock()
	d := target.Value.(time.Time)
	target.Value = time.Date(int(n), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	target.Unlock()
	return target
}

// year is a Date method.
//
// year returns the year of the date.
func year(vm *iolang.VM, target, locals *iolang.Object, msg *iolang.Message) *iolang.Object {
	target.Lock()
	d := target.Value.(time.Time)
	target.Unlock()
	return vm.NewNumber(float64(d.Year()))
}
