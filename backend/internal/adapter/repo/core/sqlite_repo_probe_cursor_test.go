package repo

import (
	"context"
	"strconv"
	"testing"

	"xiaoheiplay/internal/domain"
)

// seedProbeNodes creates n probe nodes with consecutive IDs.
func seedProbeNodes(t *testing.T, g *GormRepo, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		node := &domain.ProbeNode{
			Name:     "node-" + strconv.FormatInt(int64(i), 10),
			AgentID:  "agent-" + strconv.FormatInt(int64(i), 10),
			OSType:   "linux",
			TagsJSON: "[]",
		}
		if err := g.CreateProbeNode(context.Background(), node); err != nil {
			t.Fatalf("create node: %v", err)
		}
	}
}

// TestProbeCursorPagesBeyondOldCap verifies ListProbeNodesByCursor reaches every
// node even when the table has more rows than the old 500-cap / id-DESC list.
func TestProbeCursorPagesBeyondOldCap(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	const total = 1200
	seedProbeNodes(t, g, total)

	var afterID int64
	seen := 0
	lastID := int64(0)
	for {
		rows, err := g.ListProbeNodesByCursor(context.Background(), afterID, 200)
		if err != nil {
			t.Fatalf("cursor page: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			seen++
			if r.ID <= afterID {
				t.Fatalf("cursor went backwards: afterID=%d got id=%d", afterID, r.ID)
			}
			if r.ID <= lastID {
				t.Fatalf("ids not strictly increasing: %d then %d", lastID, r.ID)
			}
			lastID = r.ID
		}
		afterID = rows[len(rows)-1].ID
	}
	if seen != total {
		t.Fatalf("cursor paging visited %d nodes, want %d", seen, total)
	}
	if lastID != int64(total) {
		t.Fatalf("last visited id = %d, want %d", lastID, total)
	}
}

// TestProbeCursorHandlesBoundaryDeletion verifies deleting nodes mid-scan does
// not cause an infinite loop or skip surviving nodes: the cursor advances by the
// last seen id, so removed rows cannot reset it.
func TestProbeCursorHandlesBoundaryDeletion(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	seedProbeNodes(t, g, 50)

	// Delete a block of low-id nodes before scanning; they must not be visited
	// and must not break the cursor.
	for id := int64(5); id <= 10; id++ {
		if err := g.DeleteProbeNode(context.Background(), id); err != nil {
			t.Fatalf("delete node %d: %v", id, err)
		}
	}
	var afterID int64
	seen := map[int64]bool{}
	for {
		rows, err := g.ListProbeNodesByCursor(context.Background(), afterID, 10)
		if err != nil {
			t.Fatalf("cursor page: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			if r.ID >= 5 && r.ID <= 10 {
				t.Fatalf("deleted node %d was returned", r.ID)
			}
			if seen[r.ID] {
				t.Fatalf("node %d visited twice (infinite loop risk)", r.ID)
			}
			seen[r.ID] = true
		}
		afterID = rows[len(rows)-1].ID
	}
	if len(seen) != 44 {
		t.Fatalf("expected 44 surviving nodes visited, got %d", len(seen))
	}
}

// TestProbeCursorRespondsToCancellation verifies a cancelled context aborts the
// paging call.
func TestProbeCursorRespondsToCancellation(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	seedProbeNodes(t, g, 20)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.ListProbeNodesByCursor(ctx, 0, 10); err == nil {
		t.Fatalf("expected cancelled context to abort the query")
	}
}
