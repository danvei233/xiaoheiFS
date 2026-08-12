package http_test

import (
	"context"
	"fmt"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// TestHandlers_ConcurrentRequests_NoGinContextReuseRace fires overlapping
// requests through the router while other requests are still in flight. Handlers
// must derive their request context from c.Request.Context() (not the pooled
// *gin.Context), otherwise the race detector flags GORM's Rows.awaitDone
// goroutine reading a reused gin context.
func TestHandlers_ConcurrentRequests_NoGinContextReuseRace(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "buyer", "buyer@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	if err := env.Repo.CreateBillingCycle(context.Background(), &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}); err != nil {
		t.Fatalf("create cycle: %v", err)
	}

	var wg sync.WaitGroup
	const workers = 12
	const iterations = 8
	errCh := make(chan error, workers*iterations*3)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// password-reset-options is durably rate-limited per source IP
			// (10/10min). Give each worker a distinct source address so the
			// concurrency probe does not trip the rate limiter.
			remoteAddr := fmt.Sprintf("192.0.2.%d:1234", 10+workerID)
			for i := 0; i < iterations; i++ {
				rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
				if rec.Code != http.StatusOK {
					errCh <- &statusError{path: "/api/v1/catalog", code: rec.Code}
					return
				}
				rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/cart", map[string]any{
					"package_id": seed.Package.ID,
					"system_id":  seed.SystemImage.ID,
					"spec":       map[string]any{},
					"qty":        1,
				}, token)
				if rec.Code != http.StatusOK {
					errCh <- &statusError{path: "/api/v1/cart", code: rec.Code}
					return
				}
				rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/cart", nil, token)
				if rec.Code != http.StatusOK {
					errCh <- &statusError{path: "/api/v1/cart", code: rec.Code}
					return
				}
				rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, token)
				if rec.Code != http.StatusOK {
					errCh <- &statusError{path: "/api/v1/me", code: rec.Code}
					return
				}
				// Public password-reset-options exercises the settings and
				// findUserByAccount helper paths (authSvc.GetUserForPasswordReset
				// and getSettingByContext) that previously passed the pooled
				// *gin.Context as a context.Context.
				rec = testutil.DoJSONRemoteAddr(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/options", map[string]any{
					"account": "buyer@example.com",
				}, "", remoteAddr)
				if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
					errCh <- &statusError{path: "/api/v1/auth/password-reset/options", code: rec.Code}
					return
				}
				// Decode to force the response body to be fully read before the
				// next request reuses the connection/context.
				var tmp map[string]any
				_ = json.Unmarshal(rec.Body.Bytes(), &tmp)
			}
		}(w)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent request failed: %v", err)
	}
}

type statusError struct {
	path string
	code int
}

func (e *statusError) Error() string {
	return "unexpected status " + e.path + ": " + http.StatusText(e.code)
}
