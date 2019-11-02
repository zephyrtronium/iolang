Range do(
    asList := method(
        l := list()
        self foreach(v, l append(v))
    )

    map := method(call delegateToMethod(self asList, "mapInPlace"))

    select := List getSlot("select")

    slice := method(start, stop, step,
        l := list()
        step = step ifNilEval(1)
        for(i, start, stop, step, l append(self at(i)))
    )
)

Core Number do(
	to := method(end, self toBy(end, 1))
	toBy := method(end, step, Range clone setRange(self, end, step))

    nextInSequence := method(skip,
        skip ifNil(skip = 1)
        self + skip
    )
)

Core Sequence do(
    nextInSequence := method(skip,
        if(self size == 0, return self clone)
        skip ifNil(skip = 1)
        leading := 0
        self foreach(c,
            have := false
            sequenceSets foreach(set, set contains(c) ifTrue(have = true; break))
            if(have, break, leading = leading + 1)
        )
        if(leading == self size, return self clone)
        str := self exSlice(leading) asMutable
        (str size - 1) toBy(0, -1) foreach(k,
            done := false
            sequenceSets foreach(name, set,
                x := set indexOf(str at(k)) ifNil(continue) + 1
                if(x < set size,
                    str atPut(k, set at(x))
                    done = true
                ,
                    str atPut(k, set at(0))
                    if(k == 0,
                        str prependSeq(set at(if(name == "digitSequence", 1, 0)) asCharacter)
                        done = true
                    )
                )
                break
            )
            done ifTrue(break)
        )
        self exSlice(0, leading) .. if(skip > 1, str nextInSequence(skip - 1), str)
    )

    levenshtein := method(
        if(other size < self size, return other levenshtein(self))
        v := 0 to(self size) asList
        other foreach(i, y,
            u := v
            v = list(i + 1)
            self foreach(j, x,
                v append((v at(j) + 1) min(u at(j + 1) + 1) min(u at(j) + if(x == y, 0, 1)))
            )
        )
        v last
    )
)
