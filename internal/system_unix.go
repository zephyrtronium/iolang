// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package internal

import (
	"bytes"
	"fmt"

	"golang.org/x/sys/unix"
)

// platformVersion is the string to use for the System platformVersion slot. It
// is declared in each platform-specific file so that a compilation error
// occurs on any platform on which it is not implemented.
var platformVersion string

func initPV() {
	var uname unix.Utsname
	if unix.Uname(&uname) == nil {
		v, r := uname.Version[:], uname.Release[:]
		platformVersion = fmt.Sprintf("%s.%s", bytes.Trim(v, "\x00"), bytes.Trim(r, "\x00"))
	}
	// If uname failed, we don't have anything else to try.
}
