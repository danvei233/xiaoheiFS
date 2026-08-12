package shared

import "xiaoheiplay/internal/domain"

var (
	ErrForbidden             = domain.ErrForbidden
	ErrNotFound              = domain.ErrNotFound
	ErrUnauthorized          = domain.ErrUnauthorized
	ErrCaptchaFailed         = domain.ErrCaptchaFailed
	ErrConflict              = domain.ErrConflict
	ErrInvalidInput          = domain.ErrInvalidInput
	ErrInsufficientBalance   = domain.ErrInsufficientBalance
	ErrNoPaymentRequired     = domain.ErrNoPaymentRequired
	ErrPaymentAmountMismatch = domain.ErrPaymentAmountMismatch
	ErrRealNameRequired      = domain.ErrRealNameRequired
	ErrNotSupported          = domain.ErrNotSupported
	ErrResizeDisabled        = domain.ErrResizeDisabled
	ErrResizeInProgress      = domain.ErrResizeInProgress
	ErrTooManyAttempts       = domain.ErrTooManyAttempts
	ErrServiceUnavailable    = domain.ErrServiceUnavailable

	// SMS validation sentinels.
	ErrSMSNoRecipient       = domain.ErrSMSNoRecipient
	ErrSMSTooManyRecipients = domain.ErrSMSTooManyRecipients
	ErrSMSInvalidPhone      = domain.ErrSMSInvalidPhone
	ErrSMSContentRequired   = domain.ErrSMSContentRequired
	ErrSMSContentTooLong    = domain.ErrSMSContentTooLong
)
