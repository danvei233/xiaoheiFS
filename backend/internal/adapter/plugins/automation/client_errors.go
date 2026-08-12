package automation

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/internal/adapter/plugins/core"
	appshared "xiaoheiplay/internal/app/shared"
)

func mapUnimplemented(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	if st.Code() == codes.Unimplemented {
		msg := strings.TrimSpace(st.Message())
		if msg == "" {
			msg = "not supported"
		}
		return fmt.Errorf("%w: %s", appshared.ErrNotSupported, msg)
	}
	return err
}

func mapRPCBusinessError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, appshared.ErrNotSupported) {
		// mapUnimplemented already classified this as a typed "not supported"
		// error; keep the wrapper so downstream errors.Is checks still match.
		return err
	}
	if strings.Contains(err.Error(), "http_trace=") {
		// Keep full upstream trace for automation logs.
		return err
	}
	msg := extractRPCErrorMessage(err.Error())
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			// Plugin input errors are definitive: retrying with the same
			// request cannot succeed. Expose them as a domain error so the
			// order worker's createHostFailureIsDefinitive path can act.
			return fmt.Errorf("%w: %s", appshared.ErrInvalidInput, rpcErrorText(msg, st.Message()))
		}
	}
	if strings.TrimSpace(msg) == "" || msg == err.Error() {
		return err
	}
	return fmt.Errorf("%s", msg)
}

// sanitizeReturnError scrubs an error before it is returned to callers (and
// from there into user-visible HTTP/SSE responses), while preserving the typed
// sentinel chain so errors.Is still matches downstream (e.g. ErrInvalidInput for
// the order worker's createHostFailureIsDefinitive path). Logs keep the full
// error via logRPC; only the caller-visible copy is scrubbed.
func sanitizeReturnError(err error) error {
	if err == nil {
		return nil
	}
	raw := err.Error()
	clean := plugins.SanitizePluginMessage(messageWithoutHTTPTrace(raw))
	if clean == raw {
		return err
	}
	if strings.TrimSpace(clean) == "" {
		clean = "upstream request failed"
	}
	switch {
	case errors.Is(err, appshared.ErrNotSupported):
		return fmt.Errorf("%w: %s", appshared.ErrNotSupported, clean)
	case errors.Is(err, appshared.ErrInvalidInput):
		return fmt.Errorf("%w: %s", appshared.ErrInvalidInput, clean)
	default:
		return fmt.Errorf("%s", clean)
	}
}

func rpcErrorText(extracted, raw string) string {
	if strings.TrimSpace(extracted) != "" {
		return extracted
	}
	return strings.TrimSpace(raw)
}

func extractRPCErrorMessage(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed := parseRPCErrorJSON(trimmed); parsed != "" {
		return parsed
	}
	if idx := strings.Index(trimmed, "{"); idx >= 0 {
		if parsed := parseRPCErrorJSON(trimmed[idx:]); parsed != "" {
			return parsed
		}
	}
	re := regexp.MustCompile(`msg":"([^"]+)"`)
	matches := re.FindStringSubmatch(trimmed)
	if len(matches) == 2 && strings.TrimSpace(matches[1]) != "" {
		return matches[1]
	}
	return ""
}

func parseRPCErrorJSON(raw string) string {
	var obj map[string]any
	if json.Unmarshal([]byte(raw), &obj) != nil {
		return ""
	}
	for _, key := range []string{"msg", "message", "error"} {
		if v, ok := obj[key]; ok {
			msg := strings.TrimSpace(fmt.Sprint(v))
			if msg != "" && msg != "<nil>" {
				return msg
			}
		}
	}
	if other, ok := obj["other"].(map[string]any); ok {
		if v, ok := other["msg"]; ok {
			msg := strings.TrimSpace(fmt.Sprint(v))
			if msg != "" && msg != "<nil>" {
				return msg
			}
		}
	}
	return ""
}
