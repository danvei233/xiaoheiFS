package shared

import "xiaoheiplay/internal/domain"

var (
	ErrForbidden           = domain.ErrForbidden
	ErrNotFound            = domain.ErrNotFound
	ErrUnauthorized        = domain.ErrUnauthorized
	ErrCaptchaFailed       = domain.ErrCaptchaFailed
	ErrConflict            = domain.ErrConflict
	ErrInvalidInput        = domain.ErrInvalidInput
	ErrInsufficientBalance = domain.ErrInsufficientBalance
	ErrNoPaymentRequired   = domain.ErrNoPaymentRequired
	ErrRealNameRequired    = domain.ErrRealNameRequired
	ErrNotSupported        = domain.ErrNotSupported
	ErrResizeDisabled      = domain.ErrResizeDisabled
	ErrResizeInProgress    = domain.ErrResizeInProgress
)
