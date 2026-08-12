package order

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	appshared "xiaoheiplay/internal/app/shared"
)

const (
	maxLenUsername        = 64
	maxLenEmail           = 254
	maxLenQQ              = 32
	maxLenPhone           = 32
	maxLenBio             = 512
	maxLenIntro           = 1024
	maxLenPassword        = 128
	maxLenAvatarURL       = 1024
	maxLenTicketSubject   = 240
	maxLenTicketContent   = 10000
	maxLenTicketResName   = 128
	maxLenPaymentMethod   = 64
	maxLenPaymentTradeNo  = 128
	maxLenPaymentNote     = 1000
	maxLenPaymentImageURL = 1024
	maxLenReviewReason    = 1000
	maxLenRefundReason    = 1000
	maxLenPortMappingName = 64

	// maxOrderItemQty caps the quantity of a single order line. Prices are
	// server-computed, but qty comes from client/cart input and is expanded
	// into one OrderItem per unit, so an unbounded value is a memory- and
	// DB-write amplification vector.
	maxOrderItemQty = 100
	// maxOrderTotalItems caps the total number of expanded order items across
	// all lines of one order (defense in depth against oversized line counts).
	maxOrderTotalItems = 1000
)

var orderFieldValidator = validator.New()

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(appshared.SanitizePlainText(value))
	if err := orderFieldValidator.Var(trimmed, fmt.Sprintf("required,max=%d", maxLen)); err != nil {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}

func trimAndValidateOptional(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(appshared.SanitizePlainText(value))
	if err := orderFieldValidator.Var(trimmed, fmt.Sprintf("omitempty,max=%d", maxLen)); err != nil {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}
