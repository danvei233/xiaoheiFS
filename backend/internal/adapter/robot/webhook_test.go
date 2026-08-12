package robot

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func testEvent() domain.OrderEvent {
	return domain.OrderEvent{OrderID: 1, Seq: 1, Type: "order.pending_review", DataJSON: `{"no":"A-1"}`, CreatedAt: time.Now()}
}

func TestWebhookNotifier_NoWebhooks(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	notifier := NewWebhookNotifier(repo)
	if err := notifier.NotifyOrderEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// TestValidateWebhookURL guards F10: only http(s) URLs resolving to public
// addresses are accepted.
func TestValidateWebhookURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		ok   bool
	}{
		{"metadata link-local", "http://169.254.169.254/latest/meta-data", false},
		{"loopback", "http://127.0.0.1:8080/admin/clear-cache", false},
		{"private 10", "http://10.0.0.1/hook", false},
		{"private 192.168", "http://192.168.1.50/hook", false},
		{"bad scheme", "ftp://example.com/hook", false},
		{"no host", "http:///path", false},
		{"not a url", "not a url", false},
		{"public https", "https://8.8.8.8/hook", true},
		{"public http", "http://8.8.8.8/hook", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookURL(tc.url)
			if tc.ok && err != nil {
				t.Fatalf("expected %q to be allowed, got %v", tc.url, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected %q to be rejected", tc.url)
			}
		})
	}
}

// TestWebhookNotifier_RejectsBlockedURL verifies the SSRF guard is enforced in
// the notify flow: a webhook pointing at a blocked address is skipped without
// aborting the whole notify.
func TestWebhookNotifier_RejectsBlockedURL(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "robot_webhooks", ValueJSON: `[{"name":"bad","url":"http://169.254.169.254/latest/meta-data","enabled":true}]`})
	notifier := NewWebhookNotifier(repo)
	if err := notifier.NotifyOrderEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("blocked webhook should be skipped, not fatal: %v", err)
	}
}

// TestWebhookNotifier_SendsToPublicURL verifies a valid webhook is delivered
// with signature and event headers, through a fake transport.
func TestWebhookNotifier_SendsToPublicURL(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "robot_webhooks", ValueJSON: `[{"name":"ok","url":"https://8.8.8.8/hook","secret":"s3cr3t","enabled":true}]`})
	var got *http.Request
	hc := newHTTPClient()
	hc.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}, Request: req}, nil
	})
	notifier := &WebhookNotifier{settings: repo, http: hc}
	if err := notifier.NotifyOrderEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if got == nil {
		t.Fatalf("no request captured")
	}
	if got.Header.Get("X-Event") != "order.pending_review" {
		t.Fatalf("missing X-Event header")
	}
	if got.Header.Get("X-Signature") == "" {
		t.Fatalf("missing X-Signature header")
	}
}

// TestWebhookNotifier_ContinuesAfterFailedHook guards F11: a failing webhook
// must not prevent the remaining configured webhooks from being delivered.
func TestWebhookNotifier_ContinuesAfterFailedHook(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "robot_webhooks", ValueJSON: `[{"name":"bad","url":"http://169.254.169.254/hook","enabled":true},{"name":"ok","url":"https://8.8.8.8/hook","enabled":true}]`})
	var got *http.Request
	hc := newHTTPClient()
	hc.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}, Request: req}, nil
	})
	notifier := &WebhookNotifier{settings: repo, http: hc}
	if err := notifier.NotifyOrderEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if got == nil || got.URL.Host != "8.8.8.8" {
		t.Fatalf("expected the valid webhook to still be delivered, got %v", got)
	}
}

// TestWebhookNotifier_BlocksRedirectToPrivate guards the redirect case of F10:
// a redirect to a private address must abort delivery, i.e. no request may be
// issued to the private target.
func TestWebhookNotifier_BlocksRedirectToPrivate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "robot_webhooks", ValueJSON: `[{"name":"redir","url":"https://8.8.8.8/hook","enabled":true}]`})
	var requests []string
	hc := newHTTPClient()
	hc.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.String())
		return &http.Response{
			StatusCode: 302,
			Status:     "302 Found",
			Header:     http.Header{"Location": []string{"http://169.254.169.254/latest/meta-data"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	notifier := &WebhookNotifier{settings: repo, http: hc}
	if err := notifier.NotifyOrderEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if len(requests) == 0 {
		t.Fatalf("expected the initial request to be made")
	}
	for _, r := range requests {
		if strings.Contains(r, "169.254.169.254") {
			t.Fatalf("redirect to private address was followed: %v", requests)
		}
	}
}
