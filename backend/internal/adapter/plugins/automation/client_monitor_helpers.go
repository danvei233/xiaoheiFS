package automation

import (
	"encoding/json"
	"strconv"
	"strings"
)

func parseNetworkStats(raw json.RawMessage) (int64, int64) {
	if len(raw) == 0 {
		return 0, 0
	}
	var legacy struct {
		BytesSentPersec     int64 `json:"BytesSentPersec"`
		BytesReceivedPersec int64 `json:"BytesReceivedPersec"`
	}
	if err := json.Unmarshal(raw, &legacy); err == nil && (legacy.BytesSentPersec != 0 || legacy.BytesReceivedPersec != 0) {
		return legacy.BytesReceivedPersec, legacy.BytesSentPersec
	}
	var series [][]any
	if err := json.Unmarshal(raw, &series); err != nil || len(series) == 0 {
		return 0, 0
	}
	last := series[len(series)-1]
	if len(last) < 3 {
		return 0, 0
	}
	return parseInt64(last[1]), parseInt64(last[2])
}

func parseInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case string:
		id, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return id
	default:
		return 0
	}
}
