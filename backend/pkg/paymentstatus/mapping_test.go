package paymentstatus

import (
	"testing"

	pluginv1 "xiaoheiplay/plugin/v1"
)

func TestWeChatTradeStateToStatus(t *testing.T) {
	cases := map[string]pluginv1.PaymentStatus{
		"SUCCESS":    pluginv1.PaymentStatus_PAYMENT_STATUS_PAID,
		"NOTPAY":     pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING,
		"USERPAYING": pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING,
		"REFUND":     pluginv1.PaymentStatus_PAYMENT_STATUS_REFUNDED,
		"CLOSED":     pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED,
		"REVOKED":    pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED,
		"PAYERROR":   pluginv1.PaymentStatus_PAYMENT_STATUS_FAILED,
		"":           pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := WeChatTradeStateToStatus(in); got != want {
			t.Fatalf("WeChatTradeStateToStatus(%q)=%v want %v", in, got, want)
		}
	}
}

func TestAlipayTradeStatusToStatus(t *testing.T) {
	cases := map[string]pluginv1.PaymentStatus{
		"TRADE_SUCCESS":  pluginv1.PaymentStatus_PAYMENT_STATUS_PAID,
		"TRADE_FINISHED": pluginv1.PaymentStatus_PAYMENT_STATUS_PAID,
		"WAIT_BUYER_PAY": pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING,
		"TRADE_CLOSED":   pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED,
		"UNKNOWN":        pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED,
		"":               pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := AlipayTradeStatusToStatus(in); got != want {
			t.Fatalf("AlipayTradeStatusToStatus(%q)=%v want %v", in, got, want)
		}
	}
}

func TestWeChatTradeStateToStatus_RefundMapsToRefunded(t *testing.T) {
	got := WeChatTradeStateToStatus("REFUND")
	if got != pluginv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("expected REFUNDED for trade state REFUND, got %v", got)
	}
}
