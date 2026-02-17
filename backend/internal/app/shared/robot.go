package shared

import (
	"encoding/json"
	"strings"
)

func ParseRobotWebhookConfigs(raw string) []RobotWebhookConfig {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []RobotWebhookConfig
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func (cfg RobotWebhookConfig) MatchesEvent(event string) bool {
	if len(cfg.Events) == 0 {
		return true
	}
	for _, ev := range cfg.Events {
		if strings.EqualFold(strings.TrimSpace(ev), event) {
			return true
		}
	}
	return false
}
