package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// LiveProcessesByExe returns the PIDs of live processes whose executable path
// contains marker. It reads /proc directly, so it also observes processes whose
// executable was unlinked (e.g. after t.TempDir cleanup) as long as the marker
// is embedded in the original path. Returns nil on platforms without /proc.
func LiveProcessesByExe(marker string) []int {
	if runtime.GOOS != "linux" {
		return nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", e.Name(), "exe"))
		if err != nil {
			continue
		}
		if strings.Contains(exe, marker) {
			pids = append(pids, pid)
		}
	}
	return pids
}

// WaitNoProcessByExe fails the test if any live process whose executable path
// contains marker survives past timeout. It polls /proc and only returns once no
// matching executable is present, so a leaked child is always detected rather
// than masked by a fixed sleep.
func WaitNoProcessByExe(t *testing.T, marker string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if pids := LiveProcessesByExe(marker); len(pids) == 0 {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("process(es) still alive after %s (marker %q): pids=%v", timeout, marker, pids)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
