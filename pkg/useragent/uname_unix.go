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

	return fmt.Sprintf("%s %s %s %s %s", u.Sysname, u.Nodename, u.Release, u.Version, u.Machine)
}
