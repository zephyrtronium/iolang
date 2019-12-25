File do(
	with := method(p, self clone setPath(p))

	streamDestination ::= nil
	streamReadSize := 65536
	startStreaming := method(streamTo(streamDestination))
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

	stat := method(self size; self)

	create := method(if(open, close, nil))

	baseName := method(name split(".") slice(0, -1) join("."))

	thisSourceFile := method(File with(call message label))

	containingDirectory := parentDirectory := method(Directory with(path pathComponent))

	// These are meaningless without popen.
	exitStatus := nil
	termSignal := nil
)
