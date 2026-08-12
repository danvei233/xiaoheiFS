package main

import "testing"

// TestBackupID_StableAcrossListOrder covers M10: backup/snapshot ids must be
// derived from the provider's stable object name, not from the list position.
// A list-position id breaks when the list changes between List and Delete.
func TestBackupID_StableAcrossListOrder(t *testing.T) {
	a := BackupInfo{BackupName: "backup-20260811-1"}
	b := BackupInfo{BackupName: "backup-20260811-2"}

	idA := backupID(a.BackupName)
	idB := backupID(b.BackupName)
	if idA == idB {
		t.Fatalf("different backup names must produce different ids")
	}
	if idA < 0 || idB < 0 {
		t.Fatalf("backup ids must be non-negative: %d %d", idA, idB)
	}

	// Lookup must find the same object regardless of list order.
	if idx := findBackupIndex([]BackupInfo{a, b}, idA); idx != 0 {
		t.Fatalf("findBackupIndex order1 = %d, want 0", idx)
	}
	if idx := findBackupIndex([]BackupInfo{b, a}, idA); idx != 1 {
		t.Fatalf("findBackupIndex order2 = %d, want 1 (id is stable, not positional)", idx)
	}
	if idx := findBackupIndex([]BackupInfo{b}, idA); idx != -1 {
		t.Fatalf("findBackupIndex missing = %d, want -1", idx)
	}
}

func TestSnapshotID_StableAcrossListOrder(t *testing.T) {
	s1 := SnapshotInfo{SnapshotName: "snap-20260811-1"}
	s2 := SnapshotInfo{SnapshotName: "snap-20260811-2"}

	id1 := snapshotID(s1.SnapshotName)
	id2 := snapshotID(s2.SnapshotName)
	if id1 == id2 {
		t.Fatalf("different snapshot names must produce different ids")
	}
	if idx := findSnapshotIndex([]SnapshotInfo{s1, s2}, id2); idx != 1 {
		t.Fatalf("findSnapshotIndex = %d, want 1", idx)
	}
	if idx := findSnapshotIndex([]SnapshotInfo{s2, s1}, id2); idx != 0 {
		t.Fatalf("findSnapshotIndex reordered = %d, want 0", idx)
	}
}
