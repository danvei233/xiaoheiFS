package main

import (
	"strings"
	"testing"
)

// TestSanitizeEndpoint covers H4: create_host URLs carry sys_pwd/vnc_pwd as
// query parameters; emitLog must not write those passwords into the http_trace
// returned to the host (nor into any downstream log that consumes the trace).
func TestSanitizeEndpoint(t *testing.T) {
	ep := "https://panel.example.com/index.php/api/cloud/create_host?line_id=7&os=linux&cpu=2&sys_pwd=sys-secret&vnc_pwd=vnc-secret&host_name=myhost"
	got := sanitizeEndpoint(ep)
	if strings.Contains(got, "sys-secret") || strings.Contains(got, "vnc-secret") {
		t.Fatalf("sanitizeEndpoint leaked password in URL: %s", got)
	}
	if !strings.Contains(got, "sys_pwd") || !strings.Contains(got, "vnc_pwd") {
		t.Fatalf("sanitizeEndpoint dropped param keys: %s", got)
	}
	if !strings.Contains(got, "line_id=7") || !strings.Contains(got, "cpu=2") {
		t.Fatalf("sanitizeEndpoint altered non-sensitive params: %s", got)
	}
	// Unparseable / non-URL strings must be passed through untouched.
	if got := sanitizeEndpoint("not a url"); got != "not a url" {
		t.Fatalf("sanitizeEndpoint(non-url) = %q", got)
	}
	if got := sanitizeEndpoint(""); got != "" {
		t.Fatalf("sanitizeEndpoint(\"\") = %q", got)
	}
}

// TestSanitizeFormBody covers P0-13 residual: create_host sends sys_pwd/vnc_pwd
// in the POST form body (application/x-www-form-urlencoded), which carries no
// "?" marker. emitLog must mask those values in the recorded body.
func TestSanitizeFormBody(t *testing.T) {
	body := "line_id=7&os=linux&cpu=2&sys_pwd=sys-body-secret&vnc_pwd=vnc-body-secret&panel_password=panel-body-secret&host_name=myhost"
	got := sanitizeFormBody(body)
	if strings.Contains(got, "sys-body-secret") || strings.Contains(got, "vnc-body-secret") || strings.Contains(got, "panel-body-secret") {
		t.Fatalf("sanitizeFormBody leaked password in body: %s", got)
	}
	// url.Values.Encode() percent-encodes the mask: *** becomes %2A%2A%2A.
	if !strings.Contains(got, "sys_pwd=%2A%2A%2A") || !strings.Contains(got, "vnc_pwd=%2A%2A%2A") || !strings.Contains(got, "panel_password=%2A%2A%2A") {
		t.Fatalf("sanitizeFormBody did not mask secret keys: %s", got)
	}
	if !strings.Contains(got, "line_id=7") || !strings.Contains(got, "cpu=2") {
		t.Fatalf("sanitizeFormBody altered non-sensitive fields: %s", got)
	}
	// Plain text and non-sensitive forms must pass through untouched.
	for _, in := range []string{"not a form", "line_id=7&os=linux", ""} {
		if got := sanitizeFormBody(in); got != in {
			t.Fatalf("sanitizeFormBody(%q) = %q", in, got)
		}
	}
}
