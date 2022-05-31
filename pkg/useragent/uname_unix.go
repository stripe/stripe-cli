//go:build !windows
// +build !windows

package useragent

import (
	"bytes"
)

func trimNulls(input []byte) []byte {
	return bytes.Trim(input, "\x00")
}

func getUname() string {
	return "wasm"
	// u := new(unix.Utsname)

	// err := unix.Uname(u)
	// if err != nil {
	// 	panic(err)
	// }

	// return fmt.Sprintf("%s %s %s %s %s", trimNulls(u.Sysname[:]), trimNulls(u.Nodename[:]), trimNulls(u.Release[:]), trimNulls(u.Version[:]), trimNulls(u.Machine[:]))
}
