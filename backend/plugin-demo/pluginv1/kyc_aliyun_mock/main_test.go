package main

import "testing"

// TestValidCallbackURL covers M12: only absolute http(s) URLs with a host may
// be forwarded to the KYC provider as a callback/redirect target.
func TestValidCallbackURL(t *testing.T) {
	valid := []string{
		"https://host.example/kyc/cb",
		"http://host.example/kyc/cb?x=1",
		"https://host.example",
	}
	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"//host.example/path",       // protocol-relative, no scheme
		"/relative/path",            // no scheme/host
		"https://",                  // empty host
		"not a url",                 // no scheme
		"ftp://host.example/path",   // non-http scheme
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
