Block do(
	asSimpleString := method(
		if(scope, "block", "method") .. "(" .. argumentNames append("...") join(", ") .. ")"
	)
	asString := method(Formatter clone formatBlock(getSlot("self")) buf)

	callWithArgList := method(args,
		getSlot("self") doMessage(args asMessage setName("call"))
	)

	Formatter := Object clone do(
		line    ::= 0
		isEmpty ::= true
		depth   ::= 0
		buf     ::= nil
		ops     ::= nil
		asgn    ::= nil

		init := method(
			buf = Sequence clone
			ops = OperatorTable operators
			asgn = OperatorTable reverseAssignOperators
			asgn atPut("setSlotWithType", ":=")
		)

		appendSeq := method(call delegateTo(buf); isEmpty = false)

		newLine := method(
			buf appendSeq("\n")
			line = line + 1
			isEmpty = true
		)
		newLinesTo := method(msg,
			(msg lineNumber - line) minMax(0, 2) repeat(newLine)
			line = msg lineNumber
		)

		indent := method(depth repeat(appendSeq("    ")))

		formatBlock := method(blk,
			msg := getSlot("blk") message

			// This won't work until CLI exists.
			// if(msg label != CLI commandLineLabel,
			appendSeq("# " .. msg label .. ":" .. msg lineNumber, "\n") // )

			appendSeq("method(")
			if(getSlot("blk") argumentNames size > 0,
				appendSeq(getSlot("blk") argumentNames join(", "), ",")
			)
			newLine
			line = msg lineNumber
			formatIndentedMessage(msg)
			newLine
			appendSeq(")")
			self
		)

		formatIndentedMessage := method(msg,
			depth = depth + 1
			formatMessage(msg)
			depth = depth - 1
		)

		formatMessage := method(msg,
			m := msg
			while(m,
				if(m isEndOfLine,
					if(line == m next ?lineNumber,
						appendSeq(m name)
					)
					m = m next
					continue
				)
				newLinesTo(m)
				if(isEmpty, indent, if(m != msg, appendSeq(" ")))
				if(asgn hasKey(m name)) then(
					args := m arguments
					if(args first hasCachedResult,
						appendSeq(args first cachedResult .. " " .. asgn at(m name) .. " ")
						if(args at(1), formatMessage(args at(1)))
					,
						appendSeq(m name)
						if(m argCount > 0, formatArguments(m))
					)
				) elseif(ops hasKey(m name)) then(
					appendSeq(m name, " ")
					if(m arguments first, formatMessage(m arguments first))
				) else(
					appendSeq(m name)
					if(m argCount > 0, formatArguments(m))
				)
				m = m next
			)
		)

		formatArguments := method(msg,
			appendSeq("(")
			start := line
			msg arguments foreach(i, arg,
				if(i > 0,
					if(line == start, appendSeq(", "), newLine; indent; appendSeq(","); newLine)
				)
				formatIndentedMessage(arg)
			)
			if(line != start, newLine; indent)
			appendSeq(")")
		)
	)
)

Core getLocalSlot("CFunction") ifNil(Exception raise) do(
	type := "CFunction"
)
