package order

import (
	"crypto/rand"
	"encoding/json"
	"strconv"
	"strings"
)

func randomToken(n int) string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf)
}

func redactAutomationPayload(payload any) any {
	raw, err := json.Marshal(payload)
	if err != nil {
		return payload
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return payload
	}
	return redactAutomationValue(normalized)
}

func redactAutomationValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(current))
		for key, child := range current {
			normalizedKey := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if strings.Contains(normalizedKey, "password") || strings.Contains(normalizedKey, "pwd") {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactAutomationValue(child)
		}
		return redacted
	case []any:
		redacted := make([]any, len(current))
		for i := range current {
			redacted[i] = redactAutomationValue(current[i])
		}
		return redacted
	case string:
		trimmed := strings.TrimSpace(current)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var nested any
			if json.Unmarshal([]byte(trimmed), &nested) == nil {
				return redactAutomationValue(nested)
			}
		}
	}
	return value
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseHostID(v string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return id
}
