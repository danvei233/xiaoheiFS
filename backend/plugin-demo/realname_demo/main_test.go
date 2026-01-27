package main

import (
	"context"
	"testing"
)

func TestDemoProviderVerify(t *testing.T) {
	p := &DemoProvider{}
	ok, reason, err := p.Verify(context.Background(), "", "")
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if ok || reason == "" {
		t.Fatalf("expected missing fields failure")
	}
	ok, reason, err = p.Verify(context.Background(), "Alice", "123")
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if ok || reason == "" {
		t.Fatalf("expected short id failure")
	}
	ok, reason, err = p.Verify(context.Background(), "Alice", "ABC123456")
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !ok || reason != "" {
		t.Fatalf("expected success")
	}
}
