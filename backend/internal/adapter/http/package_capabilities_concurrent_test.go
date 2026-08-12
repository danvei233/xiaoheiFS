package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	appadmin "xiaoheiplay/internal/app/admin"
	"xiaoheiplay/internal/testutil"
)

// M-2: saveCapabilityPolicy performs a read-modify-write on the whole capability
// JSON document. Two concurrent saves of different items must both survive; a
// lost update would silently drop one admin's configuration. The save path must
// be serialised so the read-modify-write is atomic.
func TestConcurrentSaveCapabilityPolicy_NoLostUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, repo := testutil.NewTestDB(t, false)
	adminSvc := appadmin.NewService(repo, repo, repo, repo, repo, repo, repo)
	h := &Handler{adminSvc: adminSvc}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("user_id", int64(1))
	ctx.Request = &http.Request{}

	const items = 8
	var wg sync.WaitGroup
	for i := int64(1); i <= items; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			resize := true
			if err := h.saveGoodsTypeCapabilityPolicy(ctx, id, packageCapabilityPolicy{ResizeEnabled: &resize}); err != nil {
				t.Errorf("save policy for item %d: %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	setting, err := repo.GetSetting(context.Background(), goodsTypeCapabilitiesSettingKey)
	if err != nil {
		t.Fatalf("read setting: %v", err)
	}
	var stored map[string]packageCapabilityPolicy
	if err := json.Unmarshal([]byte(setting.ValueJSON), &stored); err != nil {
		t.Fatalf("decode stored policies: %v", err)
	}
	if len(stored) != items {
		t.Fatalf("expected %d policies to survive concurrent saves, got %d: %v", items, len(stored), stored)
	}
	for i := int64(1); i <= items; i++ {
		key := strconv.FormatInt(i, 10)
		if _, ok := stored[key]; !ok {
			t.Fatalf("policy for item %d was lost by a concurrent save", i)
		}
	}
}
