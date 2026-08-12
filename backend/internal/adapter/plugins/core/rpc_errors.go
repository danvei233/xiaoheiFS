package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"xiaoheiplay/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapRPCError normalizes plugin gRPC transport errors into business-friendly
// errors. Plugin-supplied detail is never passed through verbatim: transport
// failures are collapsed to "<pluginType> unavailable", and business failures
// are sanitized so internal endpoints, paths and tokens cannot leak into API
// responses.
func MapRPCError(err error, pluginType string) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	msg := strings.TrimSpace(st.Message())
	switch st.Code() {
	case codes.InvalidArgument:
		if msg == "" {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, SanitizePluginMessage(msg))
	case codes.FailedPrecondition:
		if msg == "" {
			msg = "failed precondition"
		}
		return fmt.Errorf("%s", SanitizePluginMessage(msg))
	case codes.Unavailable, codes.DeadlineExceeded:
		pt := strings.TrimSpace(pluginType)
		if pt == "" {
			pt = "plugin"
		}
		return fmt.Errorf("%s unavailable", pt)
	default:
		if msg == "" {
			msg = st.Code().String()
		}
		return fmt.Errorf("%s", SanitizePluginMessage(msg))
	}
}

var (
	pluginMsgURLRe   = regexp.MustCompile(`https?://[^\s]+`)
	pluginMsgIPRe    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d{1,5})?\b`)
	pluginMsgPathRe  = regexp.MustCompile(`(?:/[^\s,;:"']+)+`)
	pluginMsgTokenRe = regexp.MustCompile(`\b[0-9a-zA-Z_\-]{24,}\b`)
)

// SanitizePluginMessage scrubs likely-sensitive detail (URLs, IP addresses,
// absolute paths, long token-like runs) from plugin-supplied error text before
// it is surfaced to API clients, while preserving a short human-readable hint.
// It is exported so sibling plugin families (automation) apply the SAME scrub
// to error strings that travel to users, keeping the redaction policy uniform.
func SanitizePluginMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	msg = pluginMsgURLRe.ReplaceAllString(msg, "[url]")
	msg = pluginMsgIPRe.ReplaceAllString(msg, "[ip]")
	msg = pluginMsgPathRe.ReplaceAllString(msg, "[path]")
	msg = pluginMsgTokenRe.ReplaceAllString(msg, "[token]")
	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > 200 {
		msg = msg[:200] + "…"
	}
	return msg
}
