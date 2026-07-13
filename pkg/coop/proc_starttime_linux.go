//go:build linux

package coop

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// userHZ is the kernel clock tick rate (sysconf(_SC_CLK_TCK)); 100 on virtually
// all Linux configurations. /proc/<pid>/stat reports starttime in these ticks.
const userHZ = 100

// processStartTime derives when the process began from /proc/<pid>/stat
// (starttime, in clock ticks since boot) plus the system boot time.
func processStartTime(pid int) (time.Time, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return time.Time{}, false
	}
	// comm (field 2) is parenthesized and may contain spaces or ')', so parse
	// the fields after the final ')'. starttime is field 22 overall, i.e. index
	// 19 of the fields following comm (which start at field 3, "state").
	stat := string(data)
	rparen := strings.LastIndexByte(stat, ')')
	if rparen < 0 {
		return time.Time{}, false
	}
	fields := strings.Fields(stat[rparen+1:])
	if len(fields) < 20 {
		return time.Time{}, false
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	boot, ok := bootTime()
	if !ok {
		return time.Time{}, false
	}
	return boot.Add(time.Duration(ticks) * time.Second / userHZ), true
}

// bootTime reads the system boot time from /proc/stat's "btime" line.
func bootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		rest, found := strings.CutPrefix(line, "btime ")
		if !found {
			continue
		}
		secs, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(secs, 0), true
	}
	return time.Time{}, false
}
