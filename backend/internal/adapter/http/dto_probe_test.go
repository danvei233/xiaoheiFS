package http

import "testing"

func TestToProbeSnapshotDTO_DoubleEncodedJSON(t *testing.T) {
	raw := "\"{\\\"system\\\":{\\\"hostname\\\":\\\"node-1\\\"},\\\"disks\\\":[{\\\"mount\\\":\\\"/\\\",\\\"total\\\":100}],\\\"ports\\\":[{\\\"port\\\":22}]}\""
	dto := toProbeSnapshotDTO(raw)
	if len(dto.System) != 0 {
		t.Fatalf("expected empty system map, got %#v", dto.System)
	}
	if len(dto.Disks) != 0 {
		t.Fatalf("expected 0 disk, got %d", len(dto.Disks))
	}
	if len(dto.Ports) != 0 {
		t.Fatalf("expected 0 port, got %d", len(dto.Ports))
	}
}
