package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/domain"
)

type notifyRateSettingsStub struct {
	allowed bool
	err     error
	calls   int
}

func (s *notifyRateSettingsStub) Get(context.Context, string) (domain.Setting, error) {
	return domain.Setting{}, nil
}
func (s *notifyRateSettingsStub) List(context.Context) ([]domain.Setting, error) { return nil, nil }
func (s *notifyRateSettingsStub) ListEmailTemplates(context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}
func (s *notifyRateSettingsStub) AllowRateLimit(context.Context, string, int, time.Duration) (bool, error) {
	s.calls++
	return s.allowed, s.err
}

func TestNotifyRateLimitIsDurableAndFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name       string
		settings   *notifyRateSettingsStub
		wantStatus int
		wantNext   bool
	}{
		{name: "allowed", settings: &notifyRateSettingsStub{allowed: true}, wantStatus: http.StatusNoContent, wantNext: true},
		{name: "exhausted", settings: &notifyRateSettingsStub{}, wantStatus: http.StatusTooManyRequests},
		{name: "storage error", settings: &notifyRateSettingsStub{err: errors.New("storage unavailable")}, wantStatus: http.StatusServiceUnavailable},
		{name: "missing storage", wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var settings SettingsService
			if tc.settings != nil {
				settings = tc.settings
			}
			h := &Handler{settingsSvc: settings}
			next := false
			router := gin.New()
			router.GET("/callback", h.notifyRateLimit, func(c *gin.Context) {
				next = true
				c.Status(http.StatusNoContent)
			})
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/callback", nil)
			req.RemoteAddr = "192.0.2.10:1234"
			router.ServeHTTP(recorder, req)
			if recorder.Code != tc.wantStatus || next != tc.wantNext {
				t.Fatalf("status=%d next=%v, want status=%d next=%v", recorder.Code, next, tc.wantStatus, tc.wantNext)
			}
			if tc.settings != nil && tc.settings.calls != 1 {
				t.Fatalf("expected one durable budget call, got %d", tc.settings.calls)
			}
		})
	}
}
