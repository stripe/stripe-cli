//go:build !windows
// +build !windows

package useragent

import (
	"bytes"
	"fmt"

	"golang.org/x/sys/unix"
)

func trimNulls(input []byte) []byte {
	return bytes.Trim(input, "\x00")
}

func getUname() string {
	u := new(unix.Utsname)

	err := unix.Uname(u)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s %s %s %s %s", trimNulls(u.Sysname[:]), trimNulls(u.Nodename[:]), trimNulls(u.Release[:]), trimNulls(u.Version[:]), trimNulls(u.Machine[:]))
}
