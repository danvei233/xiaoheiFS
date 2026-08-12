package walletorder

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	appshared "xiaoheiplay/internal/app/shared"
)

const (
	maxLenPaymentNote  = 1000
	maxLenReviewReason = 1000
	maxLenRefundReason = 1000

	// maxWalletOrderAmountCents is a defensive ceiling for a single
	// recharge/withdraw order (1 billion CNY in cents). It exists only to
	// reject absurd amounts; business-grade limits belong in settings.
	maxWalletOrderAmountCents int64 = 100_000_000_000
)

var walletOrderFieldValidator = validator.New()

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(appshared.SanitizePlainText(value))
	if err := walletOrderFieldValidator.Var(trimmed, fmt.Sprintf("required,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}

func trimAndValidateOptional(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(appshared.SanitizePlainText(value))
	if err := walletOrderFieldValidator.Var(trimmed, fmt.Sprintf("omitempty,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}
