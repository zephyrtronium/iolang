package iolang

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// platformVersion is the string to use for the System platformVersion slot. It
// is declared in each platform-specific file so that a compilation error
// occurs on any platform on which it is not implemented.
var platformVersion string

func init() {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		// Presumably, we aren't on Windows NT, which means GetVersion should
		// give us what we want. Even if we are and the registry failed for
		// another reason, GetVersion is still better than giving up.
		initWinVerGV()
		return
	}
	defer k.Close()
	platformVersion, _, err = k.GetStringValue("CurrentVersion")
	if err != nil {
		// Again, GetVersion is probably the best idea here.
		initWinVerGV()
		return
	}
}

func initWinVerGV() {
	v, err := windows.GetVersion()
	if err != nil {
		// Leaving platformVersion blank is better than panicking, I think.
		platformVersion = ""
		return
	}
	platformVersion = fmt.Sprintf("%d.%d", v&0xff, v>>8&0xff)
}
