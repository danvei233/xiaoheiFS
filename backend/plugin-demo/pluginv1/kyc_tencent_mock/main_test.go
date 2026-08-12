package main

import "testing"

// TestValidCallbackURL covers M12: only absolute http(s) URLs with a host may
// be forwarded to the KYC provider as a callback/redirect target.
func TestValidCallbackURL(t *testing.T) {
	valid := []string{
		"https://host.example/kyc/cb",
		"http://host.example/kyc/cb?x=1",
	}
	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"//host.example/path",
		"/relative/path",
		"https://",
		"ftp://host.example/path",
	}
	for _, u := range valid {
		if !validCallbackURL(u) {
			t.Errorf("validCallbackURL(%q) = false, want true", u)
		}
	}
	for _, u := range invalid {
		if validCallbackURL(u) {
			t.Errorf("validCallbackURL(%q) = true, want false", u)
		}
	}
}
