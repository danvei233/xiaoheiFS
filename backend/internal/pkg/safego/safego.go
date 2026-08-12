package safego

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// Run executes one asynchronous unit and reports whether it recovered a panic.
// Callers that own durable state can use the result to mark work retryable.
func Run(operation string, fn func()) (recovered bool) {
	defer func() {
		if value := recover(); value != nil {
			recovered = true
			LogPanic(operation, value)
		}
	}()
	fn()
	return false
}

func LogPanic(operation string, value any) {
	slog.Error("background task panic recovered",
		"operation", operation,
		"panic_type", fmt.Sprintf("%T", value),
		"stack", string(debug.Stack()),
	)
}

// Go starts fn as a guarded background task. It is deliberately unsuitable for
// synchronous request handling, where a panic must remain visible to the HTTP
// recovery middleware.
func Go(operation string, fn func()) {
	go Run(operation, fn)
}
