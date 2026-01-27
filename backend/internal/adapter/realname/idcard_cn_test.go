package realname

import (
	"context"
	"testing"
)

func TestIDCardCNProvider_Verify(t *testing.T) {
	p := &IDCardCNProvider{}
	ok, reason, err := p.Verify(context.Background(), "Zhang San", "11010519491231002X")
	if err != nil || !ok || reason != "" {
		t.Fatalf("expected ok, got ok=%v reason=%q err=%v", ok, reason, err)
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
