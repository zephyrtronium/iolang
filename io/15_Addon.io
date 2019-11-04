tildeExpandsTo := if(System platform == "windows",
	method(System getEnvironmentVariable("UserProfile"))
,
	method(System getEnvironmentVariable("HOME"))
)

Addon do(
	path ::= ""
)

AddonLoader := Object clone do(
	searchPaths := list(
		System installPrefix cloneAppendPath("/eerie/base/addons") asOSPath,
		System installPrefix cloneAppendPath("/eerie/activeEnv/addons") asOSPath,
		tildeExpandsTo cloneAppendPath("/.eerie/base/addons") asOSPath,
		tildeExpandsTo cloneAppendPath("/.eerie/activeEnv/addons") asOSPath
	) selectInPlace(path, Directory with(path) exists)
	/* Don't actually do the below segment so that we don't randomly open every
	** plugin on the system.
	// Add $GOPATH/pkg/$GOOS_$GOARCH_dynlink/... for go install
	System getEnvironmentVariable("GOPATH") ?split(Path listSeparator) foreach(path,
		top := Directory with(path cloneAppendPath("/pkg/#{System platform}_#{System arch}_dynlink"))
		top exists ifFalse(continue)
		dirs := list(top)
		dirs foreach(p,
			searchPaths append(p path asOSPath)
			dirs appendSeq(p directories)
		)
	)
	*/
	// Initialize addon knowledge.
	searchPaths foreach(path, Addon scanForNewAddons(path))
)
