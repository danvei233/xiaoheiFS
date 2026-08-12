package main

import "testing"

// TestIsWechatTransactionID covers the transaction_id vs out_trade_no
// discrimination used by Refund (M3). WeChat transaction ids are pure digits;
// merchant out_trade_no values in this demo are alphanumeric order numbers.
func TestIsWechatTransactionID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"42000012022022200000000000000000", true},
		{"4200001202", true},
		{"ORDER-123", false},
		{"ORD-abc", false},
		{"", false},
		{"   ", false},
	}
	for _, c := range cases {
		if got := isWechatTransactionID(c.in); got != c.want {
			t.Errorf("isWechatTransactionID(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestBuildRefundCreateRequest pins the field selection and amount semantics of
// the WeChat v3 refund request (M3):
//   - a transaction_id must go into TransactionId, never OutTradeNo;
//   - amount.total must be the original order total, not the refund amount.
func TestBuildRefundCreateRequest(t *testing.T) {
	// transaction_id path
	r := buildRefundCreateRequest("42000012022022200000000000000000", "REF-1", 500, 1200)
	if r == nil {
		t.Fatal("buildRefundCreateRequest returned nil")
	}
	if r.TransactionId == nil || *r.TransactionId != "42000012022022200000000000000000" {
		t.Fatalf("TransactionId = %v, want transaction id", r.TransactionId)
	}
	if r.OutTradeNo != nil {
		t.Fatalf("OutTradeNo = %v, want nil for transaction_id input", r.OutTradeNo)
	}
	if r.Amount == nil || r.Amount.Total == nil || *r.Amount.Total != 1200 {
		t.Fatalf("Amount.Total = %v, want original order total 1200", r.Amount.Total)
	}
	if r.Amount == nil || r.Amount.Refund == nil || *r.Amount.Refund != 500 {
		t.Fatalf("Amount.Refund = %v, want refund amount 500", r.Amount.Refund)
	}
	if r.OutRefundNo == nil || *r.OutRefundNo != "REF-1" {
		t.Fatalf("OutRefundNo = %v, want REF-1", r.OutRefundNo)
	}

	// out_trade_no path
	r2 := buildRefundCreateRequest("ORDER-123", "REF-2", 500, 1200)
	if r2.OutTradeNo == nil || *r2.OutTradeNo != "ORDER-123" {
		t.Fatalf("OutTradeNo = %v, want ORDER-123", r2.OutTradeNo)
	}
	if r2.TransactionId != nil {
		t.Fatalf("TransactionId = %v, want nil for out_trade_no input", r2.TransactionId)
	}
	if r2.Amount == nil || r2.Amount.Total == nil || *r2.Amount.Total != 1200 {
		t.Fatalf("Amount.Total = %v, want 1200", r2.Amount.Total)
	}
}
