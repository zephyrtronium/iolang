Directory do(
	with := method(path, self clone setPath(path))

	size := method(self items size)

	fileNamed := method(name,
		File with(Path with(self path, name))
	)
	directoryNamed := folderNamed := method(name,
		Directory with(Path with(self path, name))
	)

	parentDirectory := method(
		if(path == ".", return nil)
		p := path pathComponent
		if(p isEmpty, p = ".")
		return Directory with(p)
	)
	ancestorDirectories := parents := method(
		l := list
		d := self
		while(d = d parentDirectory, l append(d))
		l reverseInPlace
	)
	accessibleAncestors := accessibleParents := method(
		ancestorDirectories selectInPlace(isAccessible)
	)

	localItems := method(
		items selectInPlace(item, item name != "." and item name != "..")
	)
	directories := folders := method(
		items selectInPlace(item, item isKindOf(Directory) and item name != "." and item name != "..")
	)
	files := method(
		items selectInPlace(isKindOf(File))
	)
	fileNames := method(
		files mapInPlace(name)
	)
	filesWithExtension := method(ext,
		if(ext containsSeq(".") not, ext = "." .. ext)
		files selectInPlace(name endsWithSeq(ext))
	)

	createIfAbsent := method(
		if(self exists not,
			parentDirectory ?createIfAbsent
			self create
		)
		self
	)
	remove := method(
		localItems foreach(remove)
		File with(self path) remove
		self
	)
	moveTo := method(path,
		File with(self path) moveTo(path)
		self setPath(path)
	)

	walk := method(
		call delegateToMethod(items selectInPlace(item, item name != "." and item name != ".."), "map")
		dirs := directories
		if(dirs size > 0, dirs foreach(dir, call delegateTo(dir)))
		nil
	)
	recursiveFilesOfTypes := method(suffixes,
		l := list
		walk(item,
			if(item isKindOf(File) and suffixes detect(s, item name endsWith(s)),
				l append(item)
			)
		)
		l
	)

	isAccessible := method(
		try(items) ifNil(return true)
		false
	)
)
