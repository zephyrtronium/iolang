Date do(
	setSlot("+", method(dur, self clone += dur))

	// The original implementation of Date today starts with Date now, which
	// means it modifies the Core Date instead of the receiver...
	today := method(now setHour(0) setMinute(0) setSecond(0))
	isToday := method(
		n := Date clone now
		n year == year and n month == month and n day == day
	)

	// The only real difference between secondsToRun and cpuSecondsToRun is
	// that the former relays its stop status, so any control flow in the
	// message will cause it to quit instead. If we were to make the latter
	// actually measure CPU time only, it would be a different story.
	secondsToRun := method(
		t := Date clone now
		call evalArgAt(0)
		t secondsSinceNow
	) setPassStops(true)

	asAtomDate := method(clone convertToUTC asString("%Y-%m-%dT%H:%M:%SZ"))
	asJson := method(asString asJson)

	asNumberString := method(asNumber asString alignLeft(27, "0"))
	timeStampString := method(Date clone now asNumberString)
)
