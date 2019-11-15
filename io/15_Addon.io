tildeExpandsTo := if(System platform == "windows",
	method(System getEnvironmentVariable("UserProfile"))
,
	method(System getEnvironmentVariable("HOME"))
)

Addon do(
	path ::= ""
)

AddonLoader := Object clone do(
	initing := Object clone do(ip := System installPrefix; te := tildeExpandsTo)
	searchPaths := list(
		initing ip ifNonNilEval(initing ip cloneAppendPath("/eerie/base/addons") asOSPath),
		initing ip ifNonNilEval(initing ip cloneAppendPath("/eerie/activeEnv/addons") asOSPath),
		initing te ifNonNilEval(initing te cloneAppendPath("/.eerie/base/addons") asOSPath),
		initing te ifNonNilEval(initing te cloneAppendPath("/.eerie/activeEnv/addons") asOSPath)
	) selectInPlace(path, path ifNonNilEval(Directory with(path) exists))
	removeSlot("initing")
	// Initialize addon knowledge.
	searchPaths foreach(path, Addon scanForNewAddons(path))
)
