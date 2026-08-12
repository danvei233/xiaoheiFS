package domain

import "testing"

// TestDuplicateSentinelsConsolidated asserts that semantically duplicate
// sentinel errors resolve to the same error value, so any layer matching with
// errors.Is / == for "invalid credential", "user not found" or "wrong current
// password" matches regardless of which identifier the caller or callee used.
// These pairs were duplicated at HEAD:
//
//	ErrInvalidCredential / ErrInvalidCredentials
//	ErrAccountNotFound   / ErrUserNotFound
//	ErrInvalidOldPassword / ErrCurrentPasswordInvalid
func TestDuplicateSentinelsConsolidated(t *testing.T) {
	if ErrInvalidCredential != ErrInvalidCredentials {
		t.Fatalf("ErrInvalidCredential (%q) and ErrInvalidCredentials (%q) must be the same sentinel", ErrInvalidCredential, ErrInvalidCredentials)
	}
	if ErrAccountNotFound != ErrUserNotFound {
		t.Fatalf("ErrAccountNotFound (%q) and ErrUserNotFound (%q) must be the same sentinel", ErrAccountNotFound, ErrUserNotFound)
	}
	if ErrInvalidOldPassword != ErrCurrentPasswordInvalid {
		t.Fatalf("ErrInvalidOldPassword (%q) and ErrCurrentPasswordInvalid (%q) must be the same sentinel", ErrInvalidOldPassword, ErrCurrentPasswordInvalid)
	}
}
