package http

import (
	"encoding/json"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type ProbeNodeDTO struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	AgentID         string            `json:"agent_id"`
	Status          string            `json:"status"`
	OSType          string            `json:"os_type"`
	Tags            []string          `json:"tags"`
	LastHeartbeatAt *time.Time        `json:"last_heartbeat_at"`
	LastSnapshotAt  *time.Time        `json:"last_snapshot_at"`
	Snapshot        *ProbeSnapshotDTO `json:"snapshot,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type ProbeSnapshotDTO struct {
	System map[string]any   `json:"system"`
	CPU    map[string]any   `json:"cpu"`
	Memory map[string]any   `json:"memory"`
	Disks  []map[string]any `json:"disks"`
	Ports  []map[string]any `json:"ports"`
	Raw    map[string]any   `json:"raw"`
}

type ProbeSLADTO struct {
	WindowFrom    time.Time             `json:"window_from"`
	WindowTo      time.Time             `json:"window_to"`
	TotalSeconds  int64                 `json:"total_seconds"`
	OnlineSeconds int64                 `json:"online_seconds"`
	UptimePercent float64               `json:"uptime_percent"`
	Events        []ProbeStatusEventDTO `json:"events"`
}

type ProbeStatusEventDTO struct {
	ID        int64     `json:"id"`
	ProbeID   int64     `json:"probe_id"`
	Status    string    `json:"status"`
	At        time.Time `json:"at"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type ProbeLogChunkDTO struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id,omitempty"`
	Data      string    `json:"data,omitempty"`
	At        time.Time `json:"at"`
}

func toProbeNodeDTO(node domain.ProbeNode) ProbeNodeDTO {
	dto := ProbeNodeDTO{
		ID:              node.ID,
		Name:            node.Name,
		AgentID:         node.AgentID,
		Status:          string(node.Status),
		OSType:          node.OSType,
		Tags:            decodeStringArray(node.TagsJSON),
		LastHeartbeatAt: node.LastHeartbeatAt,
		LastSnapshotAt:  node.LastSnapshotAt,
		CreatedAt:       node.CreatedAt,
		UpdatedAt:       node.UpdatedAt,
	}
	if strings.TrimSpace(node.LastSnapshotJSON) != "" && node.LastSnapshotJSON != "{}" {
		snapshot := toProbeSnapshotDTO(node.LastSnapshotJSON)
		dto.Snapshot = &snapshot
	}
	return dto
}

func toProbeStatusEventDTO(ev domain.ProbeStatusEvent) ProbeStatusEventDTO {
	return ProbeStatusEventDTO{
		ID:        ev.ID,
		ProbeID:   ev.ProbeID,
		Status:    string(ev.Status),
		At:        ev.At,
		Reason:    ev.Reason,
		CreatedAt: ev.CreatedAt,
	}
}

func toProbeSnapshotDTO(raw string) ProbeSnapshotDTO {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload) == 0 {
		// Backward compatibility: tolerate snapshots that were stored as a JSON string.
		var nested string
		if err2 := json.Unmarshal([]byte(raw), &nested); err2 == nil {
			nested = strings.TrimSpace(nested)
			if nested != "" {
				_ = json.Unmarshal([]byte(nested), &payload)
			}
		}
	}
	out := ProbeSnapshotDTO{
		System: mapStringAny(payload["system"]),
		CPU:    mapStringAny(payload["cpu"]),
		Memory: mapStringAny(payload["memory"]),
		Disks:  sliceMapStringAny(payload["disks"]),
		Ports:  sliceMapStringAny(payload["ports"]),
		Raw:    payload,
	}
	return out
}

func mapStringAny(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if out, ok := v.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func sliceMapStringAny(v any) []map[string]any {
	raw, ok := v.([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, mapStringAny(item))
	}
	return out
}

func decodeStringArray(raw string) []string {
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}
