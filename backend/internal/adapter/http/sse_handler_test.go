package http_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestSSEHandler_Stream(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "sse2", "sse2@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-SSE-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := env.Repo.AppendEvent(context.Background(), order.ID, "order.test", "{}"); err != nil {
		t.Fatalf("append event: %v", err)
	}

	server := httptest.NewServer(env.Router)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/orders/"+testutil.Itoa(order.ID)+"/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Close = true
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Connection", "close")
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = env.Broker.Publish(context.Background(), order.ID, "order.test", map[string]any{"ok": true})
	}()

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()
	var resp *http.Response
	select {
	case resp = <-respCh:
	case err := <-errCh:
		t.Fatalf("do request: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting response")
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected event-stream, got %q", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("expected no-cache, got %q", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "" && conn != "keep-alive" {
		t.Fatalf("expected keep-alive, got %q", conn)
	}

	reader := bufio.NewReader(resp.Body)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "data:") || line == "\n" {
				return
			}
		}
	}()
	select {
	case <-readDone:
	case <-ctx.Done():
		t.Fatalf("timeout reading sse")
	}
	cancel()
	_ = resp.Body.Close()
	server.CloseClientConnections()
	_ = server.Listener.Close()
}
