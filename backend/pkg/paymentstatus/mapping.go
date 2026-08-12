package paymentstatus

import (
	"strings"

	pluginv1 "xiaoheiplay/plugin/v1"
)

func WeChatTradeStateToStatus(tradeState string) pluginv1.PaymentStatus {
	switch strings.ToUpper(strings.TrimSpace(tradeState)) {
	case "SUCCESS":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PAID
	case "NOTPAY", "USERPAYING":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case "REFUND":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	case "CLOSED", "REVOKED":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED
	case "PAYERROR":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_FAILED
	default:
		return pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func AlipayTradeStatusToStatus(tradeStatus string) pluginv1.PaymentStatus {
	switch strings.ToUpper(strings.TrimSpace(tradeStatus)) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PAID
	case "WAIT_BUYER_PAY":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case "TRADE_CLOSED":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED
	default:
		return pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
