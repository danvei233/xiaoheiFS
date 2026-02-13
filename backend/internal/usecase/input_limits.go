package usecase

import (
	"strings"
	"unicode/utf8"
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
)

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxLen {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}

func trimAndValidateOptional(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if utf8.RuneCountInString(trimmed) > maxLen {
		return "", ErrInvalidInput
	}
	return trimmed, nil
}
