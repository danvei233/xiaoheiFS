package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSensitiveFieldsNotSerialized guards the domain models that carry
// credential material (password hashes, TOTP secrets, API-key hashes, reset
// tokens, probe secrets). These structs are never meant to be serialized
// wholesale, so the sensitive fields must be excluded from JSON output to
// prevent future handlers/services/logging from accidentally leaking them
// (constitution principle VI: never log sensitive data).
func TestSensitiveFieldsNotSerialized(t *testing.T) {
	const (
		pwHash      = "PW_HASH_SECRET_1"
		totpSecret  = "TOTP_SECRET_2"
		pendingTotp = "PENDING_TOTP_SECRET_3"
		codeHash    = "CODE_HASH_SECRET_4"
		keyHash     = "KEY_HASH_SECRET_5"
		resetToken  = "RESET_TOKEN_SECRET_6"
		secretHash  = "SECRET_HASH_SECRET_7"
		tokenHash   = "TOKEN_HASH_SECRET_8"
		pushToken   = "PUSH_TOKEN_SECRET_9"
	)

	cases := []struct {
		name   string
		value  any
		secret []string
	}{
		{
			name: "User credential fields",
			value: User{
				PasswordHash:         pwHash,
				TOTPSecretEnc:        totpSecret,
				TOTPPendingSecretEnc: pendingTotp,
			},
			secret: []string{pwHash, totpSecret, pendingTotp},
		},
		{
			name:   "Captcha code hash",
			value:  Captcha{CodeHash: codeHash},
			secret: []string{codeHash},
		},
		{
			name:   "VerificationCode code hash",
			value:  VerificationCode{CodeHash: codeHash},
			secret: []string{codeHash},
		},
		{
			name:   "APIKey key hash",
			value:  APIKey{KeyHash: keyHash},
			secret: []string{keyHash},
		},
		{
			name:   "UserAPIKey key hash",
			value:  UserAPIKey{KeyHash: keyHash},
			secret: []string{keyHash},
		},
		{
			name:   "PasswordResetToken token",
			value:  PasswordResetToken{Token: resetToken},
			secret: []string{resetToken},
		},
		{
			name:   "PasswordResetTicket token",
			value:  PasswordResetTicket{Token: resetToken},
			secret: []string{resetToken},
		},
		{
			name:   "ProbeNode secret hash",
			value:  ProbeNode{SecretHash: secretHash},
			secret: []string{secretHash},
		},
		{
			name:   "ProbeEnrollToken token hash",
			value:  ProbeEnrollToken{TokenHash: tokenHash},
			secret: []string{tokenHash},
		},
		{
			name:   "PushToken token",
			value:  PushToken{Token: pushToken},
			secret: []string{pushToken},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			for _, s := range tc.secret {
				if strings.Contains(string(raw), s) {
					t.Fatalf("sensitive value %q leaked into JSON: %s", s, raw)
				}
			}
		})
	}
}
