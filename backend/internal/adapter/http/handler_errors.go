package http

import (
	"errors"

	"xiaoheiplay/internal/domain"
)

// safeClientSentinels are the domain errors that may be surfaced verbatim to
// admin clients. Anything not in this set (e.g. a raw repository or driver
// failure) is replaced with a generic message, so internal error details never
// leak through the API (Issue #51 P2-11).
var safeClientSentinels = []error{
	domain.ErrInvalidInput,
	domain.ErrNotFound,
	domain.ErrConflict,
	domain.ErrForbidden,
	domain.ErrInvalidId,
	domain.ErrInvalidStatus,
	domain.ErrInvalidExpireAt,
	domain.ErrPhoneAlreadyExists,
	domain.ErrEmailAlreadyExists,
	domain.ErrNotSupported,
	domain.ErrCannotDeleteSelf,
}

// clientErrMessage maps a service error to a client-safe message. Recognized
// domain sentinels keep their canonical text (the frontend and tests depend on
// those exact strings), while any unexpected error is reported as a generic
// request failure so implementation details are never disclosed.
func clientErrMessage(err error) string {
	if err == nil {
		return domain.ErrRequestFailed.Error()
	}
	for _, safe := range safeClientSentinels {
		if errors.Is(err, safe) {
			return err.Error()
		}
	}
	return domain.ErrRequestFailed.Error()
}
