package shared

import "strings"

// IsHostNotFoundError reports whether a DeleteHost error means the remote host
// is already gone. Treating "already gone" as an idempotent success is what lets
// a crash between the remote delete and the local state update converge on
// retry: re-running DeleteHost against a destroyed host must not block the local
// row cleanup. Mirrors the error text the automation plugins surface for a
// missing instance.
func IsHostNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "not exist") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "不存在") ||
		strings.Contains(msg, "404")
}
