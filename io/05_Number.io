Number do(
	combinations := method(k,
		n := self + 1
		k = k min(self - k)
		v := 1
		for(i, 1, k, v = v * (n - i) / i)
		v
	)
	permutations := method(k,
		v := 1
		for(i, 0, k-1, v = v * (self - i))
		v
	)

	asHex := method(toBaseWholeBytes(16))
	asBinary := method(toBaseWholeBytes(2))
	asOctal := method(toBaseWholeBytes(8))

	isInASequenceSet := method(
		Sequence sequenceSets foreach(set, if(in(set), return true))
		false
	)
)
