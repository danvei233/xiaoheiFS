package automation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	if strings.Contains(err.Error(), "http_trace=") {
		// Keep full upstream trace for automation logs.
		return err
	}
	msg := extractRPCErrorMessage(err.Error())
	if strings.TrimSpace(msg) == "" || msg == err.Error() {
		return err
	}
	return fmt.Errorf("%s", msg)
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
