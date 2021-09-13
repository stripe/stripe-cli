//go:build windows
// +build windows

package useragent

func getUname() string {
	// TODO: if there is appetite for it in the community
	// add support for Windows GetSystemInfo
	return ""
}
