package safego

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestGo_PanicDoesNotStopSubsequentBackgroundTask(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	if !Run("panic-test", func() { panic("sensitive panic value") }) {
		t.Fatal("expected panic to be recovered")
	}

	secondDone := make(chan struct{})
	Go("follow-up-test", func() { close(secondDone) })
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("subsequent background task did not run")
	}

	output := logs.String()
	if !strings.Contains(output, `"operation":"panic-test"`) || !strings.Contains(output, `"panic_type":"string"`) {
		t.Fatalf("missing structured panic fields: %s", output)
	}
	if strings.Contains(output, "sensitive panic value") {
		t.Fatalf("panic value leaked to logs: %s", output)
	}
}
