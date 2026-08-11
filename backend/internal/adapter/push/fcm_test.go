package push

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func fakeResponse(status int, body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
		Request:    req,
	}
}

// TestRedactTokenInBody guards F9: FCM error bodies must not echo the
// recipient token back into logs via the returned error.
func TestRedactTokenInBody(t *testing.T) {
	body := `{"error":{"message":"Invalid token: abc123def requested entity"}}`
	got := redactTokenInBody(body, "abc123def")
	if strings.Contains(got, "abc123def") {
		t.Fatalf("token not redacted: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", got)
	}
	// Empty token must leave the body untouched.
	original := `{"error":{"status":"UNAVAILABLE"}}`
	if redactTokenInBody(original, "") != original {
		t.Fatalf("empty token changed the body")
	}
}

// TestFCMSender_LegacySend exercises the legacy batch path through a fake
// transport and verifies the auth header and request body are built correctly.
func TestFCMSender_LegacySend(t *testing.T) {
	var got *http.Request
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			got = req
			return fakeResponse(200, `{"success":1}`, req), nil
		}),
	}
	s := &FCMSender{http: client}
	err := s.Send(context.Background(),
		appshared.PushConfig{LegacyServerKey: "LEGACY-KEY"},
		[]string{"tok-1", "tok-2"},
		appshared.PushPayload{Title: "title", Body: "body", Data: map[string]string{"k": "v"}},
	)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if got == nil {
		t.Fatalf("no request captured")
	}
	if auth := got.Header.Get("Authorization"); auth != "key=LEGACY-KEY" {
		t.Fatalf("unexpected auth header %q", auth)
	}
	if got.URL.String() != fcmLegacyEndpoint {
		t.Fatalf("unexpected endpoint %q", got.URL.String())
	}
}

// TestFCMSender_LegacySendStatusError guards the 500 failure path.
func TestFCMSender_LegacySendStatusError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return fakeResponse(500, `{"error":"boom"}`, req), nil
		}),
	}
	s := &FCMSender{http: client}
	err := s.Send(context.Background(),
		appshared.PushConfig{LegacyServerKey: "LEGACY-KEY"},
		[]string{"tok-1"},
		appshared.PushPayload{Title: "t", Body: "b"},
	)
	if err == nil {
		t.Fatalf("expected error on 500")
	}
}

// TestFCMSender_EmptyTokens and NoConfig are guards for the early-return paths.
func TestFCMSender_EmptyTokens(t *testing.T) {
	s := &FCMSender{http: &http.Client{}}
	if err := s.Send(context.Background(), appshared.PushConfig{}, nil, appshared.PushPayload{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFCMSender_NoConfig(t *testing.T) {
	s := &FCMSender{http: &http.Client{}}
	if err := s.Send(context.Background(), appshared.PushConfig{}, []string{"tok-1"}, appshared.PushPayload{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
