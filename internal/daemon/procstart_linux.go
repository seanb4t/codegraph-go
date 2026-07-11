//go:build linux

package daemon

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// procStatClockTicks is USER_HZ, the units /proc/<pid>/stat's starttime
// field is measured in. It is effectively always 100 on Linux
// (sysconf(_SC_CLK_TCK) is a kernel-compile-time constant on every mainline
// distribution kernel) — a fixed value here rather than an actual sysconf
// call avoids a CGo dependency for a value that has been unanimously 100
// across every target platform this project builds for.
const procStatClockTicks = 100

// processStartTime returns pid's start time as a wall-clock time.Time,
// derived from /proc/<pid>/stat's starttime field (clock ticks since boot,
// field 22) plus /proc/stat's btime line (system boot time, seconds since
// the Unix epoch) — the standard Linux corroboration source for WR-02's
// PID-reuse mitigation. ok is false if any read/parse step fails (missing
// /proc, permission denied, unexpected format): the caller (isStale) falls
// back to liveness-only staleness in that case, never treating "can't
// corroborate" as itself a staleness signal.
func processStartTime(pid int) (time.Time, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, false
	}
	text := string(data)

	// The comm field (field 2) is parenthesized and may itself contain
	// ')' or whitespace; per the kernel's own documented format the LAST
	// ')' in the line is always the true end of the comm field, so every
	// subsequent field can be split on whitespace safely.
	closeParen := strings.LastIndexByte(text, ')')
	if closeParen < 0 || closeParen+2 > len(text) {
		return time.Time{}, false
	}
	fields := strings.Fields(text[closeParen+2:])
	// fields[0] here is state (field 3 overall); starttime is field 22
	// overall, i.e. fields[22-3] = fields[19].
	const starttimeIdx = 22 - 3
	if len(fields) <= starttimeIdx {
		return time.Time{}, false
	}
	ticks, err := strconv.ParseInt(fields[starttimeIdx], 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	bootUnix, ok := bootTimeUnix()
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(bootUnix+ticks/procStatClockTicks, 0).UTC(), true
}

// bootTimeUnix reads /proc/stat's btime line (system boot time, seconds
// since the Unix epoch).
func bootTimeUnix() (int64, bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		const prefix = "btime "
		line := sc.Text()
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(line[len(prefix):]), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}
