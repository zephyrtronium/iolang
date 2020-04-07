TestRunner := Object clone do(
	width ::= 70

	init := method(
		self cases := nil
		self exceptions := list()
		self runtime := 0
	)

	testCount := method(
		self cases values reduce(n, names, n + names size, 0)
	)

	name := method(
		if(self cases size > 1,
			System launchScript fileName
		,
			if(self cases size == 1,
				self cases keys first
			,
				""
			)
		)
	)

	linebreak := method(
		if(self ?dots, self dots = self dots + 1, self dots := 1)
		if(dots % width == 0, "\n" print)
	)
	success := method("." print; linebreak)
	error := method(name, exc,
		exceptions append(list(name, exc))
		"E" print
		linebreak
	)

	run := method(tests,
		self cases := tests
		self runtime := Date secondsToRun(
			tests foreach(name, slots,
				case := Lobby getSlot(name)
				slots foreach(slot,
					case setUp
					exc := try(case doString(slot))
					if(exc, error(name .. " " .. slot, exc), success)
					case tearDown
				)
			)
		)
		printExceptions
		printSummary
	)

	printExceptions := method(
		"\n" print
		exceptions foreach(exc,
			("=" repeated(width) .. "\nFAIL: " .. exc at(0) .. "\n" .. "-" repeated(width)) println
			exc showStack
		)
	)
	printSummary := method(
		"-" repeated(width) println
		("Ran " .. testCount .. " test" .. if(testCount != 1, "s", "") .. " in " .. runtime .. "s\n") println
		result := if(exceptions isEmpty not, "FAILED (failures #{exceptions size})" interpolate, "OK")
		(result .. name alignRight(width - result size) .. "\n") println
	)
)

RunnerMixIn := Object clone do(
	run := method(TestRunner clone run(prepare))
)

UnitTest := Object clone prependProto(RunnerMixIn) do(
	setUp := method(nil)
	tearDown := method(nil)

	testSlotNames := method(
		names := self slotNames select(beginsWithSeq("test"))
		if(names isEmpty, names, names sortByKey(name, self getSlot(name) message lineNumber))
	)

	prepare := method(Map with(self type, testSlotNames))

	fail := method(error, Exception raise(if(error, error, "fail")))

	assertEquals := method(a, b, m,
		m ifNil(m = call message)
		if(a != b, fail(
			"'#{m argAt(0)} != #{m argAt(1)}' --> '#{a asSimpleString} != #{b asSimpleString}'" interpolate
		))
	)
	assertNotEquals := method(a, b, m,
		m ifNil(m = call message)
		if(a == b, fail(
			"'#{m argAt(0)} == #{m argAt(1)}' --> '#{a asSimpleString} == #{b asSimpleString}'" interpolate
		))
	)
	assertSame := method(a, b, assertEquals(a uniqueId, b uniqueId, call message))
	assertNotSame := method(a, b, assertNotEquals(a uniqueId, b uniqueId, call message))
	assertNil := method(a, assertEquals(a, nil, call message))
	assertNotNil := method(a, assertNotEquals(a, nil, call message))
	assertTrue := method(a, assertEquals(a, true, call message))
	assertFalse := method(a, assertEquals(a, false, call message))
	assertRaisesException := method(
		try(call evalArgAt(0)) ifNil(
			fail("'#{call argAt(0)}' should have raised Exception" interpolate)
		)
	)
	assertEqualsWithinDelta := method(a, b, delta,
		if((a - b) abs > delta,
			fail("#{a} expected, but was #{b} (allowed delta: #{delta})")
		)
	)
	knownBug := method(fail("'#{call argAt(0)}' is a known bug" interpolate))
)

DirectoryCollector := TestSuite := Object clone prependProto(RunnerMixIn) do(
	path ::= lazySlot(System launchPath)
	with := method(p, self clone setPath(p))
	testFiles := method(Directory with(path) files select(name endsWithSeq("Test.io")))
	prepare := method(
		testFiles foreach(file, Lobby doString(file contents, file path))
		FileCollector prepare
	)
)

FileCollector := Object clone prependProto(RunnerMixIn) do(
	prepare := method(
		cases := Map clone
		Lobby foreachSlot(name, value,
			if(getSlot("value") isActivatable not and value isKindOf(UnitTest),
				cases mergeInPlace(value prepare)
			)
		)
		cases
	)
)
