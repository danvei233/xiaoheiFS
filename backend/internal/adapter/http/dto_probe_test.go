package http

import "testing"

func TestToProbeSnapshotDTO_DoubleEncodedJSON(t *testing.T) {
	raw := "\"{\\\"system\\\":{\\\"hostname\\\":\\\"node-1\\\"},\\\"disks\\\":[{\\\"mount\\\":\\\"/\\\",\\\"total\\\":100}],\\\"ports\\\":[{\\\"port\\\":22}]}\""
	dto := toProbeSnapshotDTO(raw)
	if dto.System["hostname"] != "node-1" {
		t.Fatalf("unexpected hostname: %#v", dto.System["hostname"])
	}
	if len(dto.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(dto.Disks))
	}
	if len(dto.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(dto.Ports))
	}
}
