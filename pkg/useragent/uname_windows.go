// go:build windows
//go:build windows
// +build windows

// Package useragent provides system-identifying strings for CLI tools and services.
// This Windows implementation matches the Unix getUname format.
//
// Note:
// - We use x/sys/windows where available (e.g., GetComputerName, UTF16ToString).
// - RtlGetVersion and SYSTEM_INFO are NOT exposed by x/sys/windows, so we define them manually.
// - The layout matches the Windows API exactly to ensure safe syscall usage.

package useragent

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Constants for processor architectures (not exposed by x/sys/windows)
const (
	PROCESSOR_ARCHITECTURE_INTEL = 0
	PROCESSOR_ARCHITECTURE_ARM64 = 12
	PROCESSOR_ARCHITECTURE_IA64  = 6
	PROCESSOR_ARCHITECTURE_AMD64 = 9
)

// SYSTEM_INFO struct (not available in x/sys/windows)
// Source: https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/ns-sysinfoapi-system_info
type systemInfo struct {
	wProcessorArchitecture      uint16
	wReserved                   uint16
	dwPageSize                  uint32
	lpMinimumApplicationAddress uintptr
	lpMaximumApplicationAddress uintptr
	dwActiveProcessorMask       uintptr
	dwNumberOfProcessors        uint32
	dwProcessorType             uint32
	dwAllocationGranularity     uint32
	wProcessorLevel             uint16
	wProcessorRevision          uint16
}

// OSVERSIONINFOEXW for RtlGetVersion (not exposed in x/sys/windows)
// Source: https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-osversioninfoexw
type osVersionInfoEx struct {
	dwOSVersionInfoSize uint32
	dwMajorVersion      uint32
	dwMinorVersion      uint32
	dwBuildNumber       uint32
	dwPlatformId        uint32
	szCSDVersion        [128]uint16
	wServicePackMajor   uint16
	wServicePackMinor   uint16
	wSuiteMask          uint16
	wProductType        byte
	wReserved           byte
}

func getUname() string {
	sysname := "Windows"

	// Get nodename (hostname)
	var computerName [256]uint16
	size := uint32(len(computerName))
	err := windows.GetComputerName(&computerName[0], &size)
	if err != nil {
		panic(err)
	}
	nodename := windows.UTF16ToString(computerName[:])

	// Get OS version using RtlGetVersion (more accurate than GetVersionEx)
	modntdll := syscall.NewLazyDLL("ntdll.dll")
	procRtlGetVersion := modntdll.NewProc("RtlGetVersion")

	var osVer osVersionInfoEx
	osVer.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osVer))
	ret, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&osVer)))
	if ret != 0 {
		panic("RtlGetVersion failed")
	}

	release := fmt.Sprintf("%d.%d.%d", osVer.dwMajorVersion, osVer.dwMinorVersion, osVer.dwBuildNumber)
	version := fmt.Sprintf("Build %d", osVer.dwBuildNumber)

	// Get machine architecture using GetNativeSystemInfo
	var sysInfo systemInfo
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetNativeSystemInfo := kernel32.NewProc("GetNativeSystemInfo")
	procGetNativeSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))

	var machine string
	switch sysInfo.wProcessorArchitecture {
	case PROCESSOR_ARCHITECTURE_AMD64:
		machine = "x86_64"
	case PROCESSOR_ARCHITECTURE_ARM64:
		machine = "arm64"
	case PROCESSOR_ARCHITECTURE_IA64:
		machine = "ia64"
	case PROCESSOR_ARCHITECTURE_INTEL:
		machine = "x86"
	default:
		machine = "unknown"
	}

	return fmt.Sprintf("%s %s %s %s %s", sysname, nodename, release, version, machine)
}
