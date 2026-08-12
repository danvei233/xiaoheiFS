package realname

import (
	"context"
	"strings"
	"testing"
)

func TestIDCardCNProvider_Verify(t *testing.T) {
	p := &IDCardCNProvider{}
	// A locally well-formed ID card must NOT grant verified: it is input
	// pre-validation only and must route to pending/manual review.
	ok, reason, err := p.Verify(context.Background(), "Zhang San", "11010519491231002X")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected pending (not verified), got ok=%v reason=%q err=%v", ok, reason, err)
	}
	if !strings.HasPrefix(reason, "pending") {
		t.Fatalf("expected pending reason, got %q", reason)
	}
	ok, reason, err = p.Verify(context.Background(), "", "11010519491231002X")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected name required error")
	}
	ok, reason, err = p.Verify(context.Background(), "Zhang", "123")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected length invalid")
	}
	ok, reason, err = p.Verify(context.Background(), "Zhang", "110105194912310021")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected checksum invalid")
	}
	ok, reason, err = p.Verify(context.Background(), "Zhang", "11010519991332002X")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected birth invalid")
	}
}

func TestIDCardCNProvider_Verify_RejectsFutureBirth(t *testing.T) {
	p := &IDCardCNProvider{}
	ok, reason, err := p.Verify(context.Background(), "Zhang", "11010520991231002X")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected future birth date rejected, got ok=%v reason=%q err=%v", ok, reason, err)
	}
}

func TestIDCardCNProvider_Verify_RejectsYearZero(t *testing.T) {
	p := &IDCardCNProvider{}
	ok, reason, err := p.Verify(context.Background(), "Zhang", "110101000001010014")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected year-zero birth rejected, got ok=%v reason=%q err=%v", ok, reason, err)
	}
}

func TestIDCardCNProvider_Verify_RejectsInvalidRegion(t *testing.T) {
	p := &IDCardCNProvider{}
	ok, reason, err := p.Verify(context.Background(), "Zhang", "990105194912310015")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected region-invalid rejected, got ok=%v reason=%q err=%v", ok, reason, err)
	}
}

func TestIDCardCNProvider_Verify_RejectsNonASCIIDigit(t *testing.T) {
	p := &IDCardCNProvider{}
	// Last byte is a double-byte digit, must not pass the ASCII-only check.
	ok, reason, err := p.Verify(context.Background(), "Zhang", "11010519491231001٢")
	if err != nil || ok || reason == "" {
		t.Fatalf("expected non-ASCII digit rejected, got ok=%v reason=%q err=%v", ok, reason, err)
	}
}
