File do(
	with := method(p, self clone setPath(p))

	streamReadSize := 65536
	streamTo := method(dst,
		b := Sequence clone
		open
		while(isAtEnd not,
			b empty
			readToBufferLength(b, streamReadSize)
			dst write(b)
			?yield
		)
	)
	streamToWithoutYielding := method(dst,
		b := Sequence clone
		open
		while(isAtEnd not,
			b empty
			readToBufferLength(b, streamReadSize)
			dst write(b)
		)
	)

	copyToPath := method(p,
		dst := File with(p) open
		open streamTo(dst)
		dst close
		close
	)
	copyToPathWithoutYielding := method(p,
		dst := File with(p) open
		open streamToWithoutYielding(dst)
		dst close
		close
	)

	setContents := method(v, truncateToSize(0) open write(v) close)
	appendToContents := method(
		openForAppending
		call evalArgs foreach(v, write(v))
		close
	)

	create := method(if(open, close, nil))

	// This won't work until Sequence split exists.
	baseName := method(name split(".") slice(0, -1) join("."))

	thisSourceFile := method(File with(call message label))

	// This won't work until Sequence pathComponent exists.
	parentDirectory := method(Directory with(path pathComponent))
)
