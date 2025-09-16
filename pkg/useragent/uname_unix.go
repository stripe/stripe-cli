//go:build !windows
// +build !windows

package useragent

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func getUname() string {
	u := new(unix.Utsname)

	err := unix.Uname(u)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s %s %s %s %s", trimNulls(u.Sysname[:]), trimNulls(u.Nodename[:]), trimNulls(u.Release[:]), trimNulls(u.Version[:]), trimNulls(u.Machine[:]))
}
