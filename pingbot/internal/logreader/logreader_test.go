package logreader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStreamFileReadsTailNotWholeFile verifies a log file much larger than the
// read cap is not loaded whole: only the tail is scanned, so the first line of
// the file is never emitted and the last line always is. This guards against
// the old os.ReadFile behavior that could OOM the agent on a very large log.
func TestStreamFileReadsTailNotWholeFile(t *testing.T) {
	old := maxLogFileTailBytes
	maxLogFileTailBytes = 2048
	defer func() { maxLogFileTailBytes = old }()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.log")
	var b strings.Builder
	const totalLines = 4000
	for i := 0; i < totalLines; i++ {
		fmt.Fprintf(&b, "line-%05d %s\n", i, strings.Repeat("x", 120)) // ~130 bytes/line
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var emitted []string
	err := Stream(context.Background(), "file:"+path, "", 500, false, func(line string) bool {
		emitted = append(emitted, line)
		return true
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if len(emitted) == 0 {
		t.Fatalf("no lines emitted")
	}
	last := emitted[len(emitted)-1]
	if !strings.Contains(last, fmt.Sprintf("line-%05d", totalLines-1)) {
		t.Fatalf("last line not emitted: %q", last)
	}
	for _, line := range emitted {
		if strings.Contains(line, "line-00000") {
			t.Fatalf("first line emitted (%q) — whole file was read into memory", line)
		}
	}
}
