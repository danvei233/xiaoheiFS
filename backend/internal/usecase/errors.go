package usecase

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrForbidden           = errors.New("forbidden")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrConflict            = errors.New("conflict")
	ErrInvalidInput        = errors.New("invalid input")
	ErrNoPaymentRequired   = errors.New("no payment required")
	ErrNoChanges           = errors.New("no changes")
	ErrResizeSamePlan      = errors.New("resize target matches current plan")
	ErrResizeDisabled      = errors.New("resize disabled")
	ErrResizeInProgress    = errors.New("resize already in progress")
	ErrCaptchaFailed       = errors.New("captcha failed")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrRealNameRequired    = errors.New("real name required")
	ErrProvisioning        = errors.New("provisioning")
)
