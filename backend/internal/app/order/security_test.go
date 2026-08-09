package order

import (
	"context"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

type recordingAutomationLogRepo struct {
	logs []domain.AutomationLog
}

func (r *recordingAutomationLogRepo) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	r.logs = append(r.logs, *log)
	return nil
}

func (r *recordingAutomationLogRepo) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	return r.logs, len(r.logs), nil
}

func (r *recordingAutomationLogRepo) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	return nil
}

func TestAutomationRequestLogRedactsProvisioningPasswords(t *testing.T) {
	repo := &recordingAutomationLogRepo{}
	svc := &OrderService{autoLogs: repo}
	req := AutomationCreateHostRequest{LineID: 1, OS: "linux", SysPwd: "system-secret", VNCPwd: "vnc-secret"}
	svc.logAutomation(context.Background(), 1, 2, "create_host", req, map[string]any{
		"os_password":  "response-system-secret",
		"vnc_password": "response-vnc-secret",
	}, true, "ok")

	if len(repo.logs) != 1 {
		t.Fatalf("expected one log, got %d", len(repo.logs))
	}
	serialized := repo.logs[0].RequestJSON + repo.logs[0].ResponseJSON
	for _, secret := range []string{"system-secret", "vnc-secret", "response-system-secret", "response-vnc-secret"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("automation log contains secret %q: %s", secret, serialized)
		}
	}
	if !strings.Contains(serialized, "[REDACTED]") {
		t.Fatalf("automation log does not contain redaction marker: %s", serialized)
	}
}
