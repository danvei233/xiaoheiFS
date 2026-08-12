// Package scrub redacts credentials and topology that can appear inside error
// strings returned by external providers before they are persisted or surfaced.
//
// Provider errors routinely embed request URLs (with query tokens), connection
// strings, passwords and internal host names. Storing those verbatim in meta,
// audit logs or user-visible messages would leak secrets and internal topology.
package scrub

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	// tokenValueRE matches `name=value` / `name: value` pairs whose name smells
	// like a credential (password, token, key, secret, sig, signature, pwd).
	tokenValueRE = regexp.MustCompile(`(?i)(\b(?:password|passwd|pwd|token|api[_-]?key|access[_-]?key|secret|signature|sig|authorization|credential)\b\s*[=:]\s*)([^\s,;]+)`)

	// longTokenRE matches long runs of hex/base64 that are almost certainly
	// machine-generated secrets rather than user data.
	longTokenRE = regexp.MustCompile(`\b(?:[A-Fa-f0-9]{24,}|[A-Za-z0-9+/_=-]{24,})\b`)

	// bearerRE matches an `Authorization: Bearer <token>` style header value.
	bearerRE = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`)

	// headerValueRE matches `Header: value` lines for common secret headers.
	headerValueRE = regexp.MustCompile(`(?i)(\b(?:x-api-key|authorization|x-auth-token)\b\s*[:=]\s*)([^\s,;]+)`)

	// queryKeyRE matches query string keys that carry credentials.
	queryKeyRE = regexp.MustCompile(`(?i)([?&](?:password|passwd|token|key|secret|sig|signature|auth|credential)=)[^&\s]+`)

	urlCredentialRE = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)([^/@\s]+)@`)

	// userInfoRE matches `user:password@host` appearing without a scheme (e.g.
	// inside "dial tcp admin:secret@10.0.0.5:443" errors). The password may not
	// contain '@' or whitespace; URL-encoded credentials are already handled by
	// redactURL.
	userInfoRE = regexp.MustCompile(`\b([^\s/@:]+):([^\s/@]+)@([^\s/@]+)`)
)

const redacted = "[REDACTED]"

// Redact returns s with credential-bearing substrings and internal topology
// replaced by a fixed marker. It is best-effort: it never removes text that is
// not suspicious, so a non-secret message survives unchanged.
func Redact(s string) string {
	if s == "" {
		return ""
	}
	s = redactURL(s)
	s = urlCredentialRE.ReplaceAllString(s, "${1}${2}@"+redacted)
	// Bearer/header tokens must be redacted as a whole before tokenValueRE runs,
	// otherwise `Authorization: Bearer <token>` would only redact the word
	// "Bearer" and leave the token behind.
	s = bearerRE.ReplaceAllString(s, "${1}"+redacted)
	s = headerValueRE.ReplaceAllString(s, "${1}"+redacted)
	s = userInfoRE.ReplaceAllString(s, "${1}:[REDACTED]@${3}")
	s = tokenValueRE.ReplaceAllString(s, "${1}"+redacted)
	s = queryKeyRE.ReplaceAllString(s, "${1}"+redacted)
	s = longTokenRE.ReplaceAllString(s, redacted)
	return s
}

// redactURL neutralises userinfo, query and fragment of any URL found in s so a
// provider error carrying a signed callback URL does not leak its token or the
// internal host it points at.
func redactURL(s string) string {
	idx := strings.Index(s, "://")
	if idx < 0 {
		return s
	}
	// Scan for URL-like substrings bounded by whitespace/quotes/punctuation.
	out := s
	for start := 0; ; {
		pos := strings.Index(out[start:], "://")
		if pos < 0 {
			break
		}
		abs := start + pos
		begin := abs
		for begin > 0 {
			c := out[begin-1]
			if c == ' ' || c == '\t' || c == '\n' || c == '"' || c == '\'' || c == '<' || c == '>' {
				break
			}
			begin--
		}
		end := abs + 3
		for end < len(out) {
			c := out[end]
			if c == ' ' || c == '\t' || c == '\n' || c == '"' || c == '\'' || c == '<' || c == '>' || c == ',' || c == ')' || c == ']' {
				break
			}
			end++
		}
		token := out[begin:end]
		if u, err := url.Parse(token); err == nil && u.Scheme != "" && u.Host != "" {
			replacement := redactURLToken(token)
			out = out[:begin] + replacement + out[end:]
			start = begin + len(replacement)
		} else {
			start = abs + 3
		}
	}
	return out
}

func redactURLToken(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = url.UserPassword(redacted, redacted)
	}
	if u.Query() != nil {
		q := u.Query()
		for k := range q {
			switch strings.ToLower(k) {
			case "password", "passwd", "pwd", "token", "key", "secret", "sig", "signature", "auth", "credential", "sign":
				q.Set(k, redacted)
			}
		}
		u.RawQuery = q.Encode()
	}
	// Keep a public host name for context but never a numeric internal address.
	if ip := net.ParseIP(u.Hostname()); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()) {
		u.Host = redacted
	}
	return u.String()
}

// SanitizeSummary produces a stable, short summary of an external failure
// suitable for persisting in meta/audit. It keeps only the error category and
// strips the rest, so provider-specific detail never leaks.
func SanitizeSummary(code string) string {
	switch code {
	case "invalid_input":
		return "cleanup request invalid"
	case "not_found":
		return "instance not found"
	case "provider_timeout":
		return "cleanup provider timed out"
	case "provider_unreachable":
		return "cleanup provider unreachable"
	case "provider_rejected":
		return "cleanup provider rejected the request"
	case "local_update_failed":
		return "cleanup completed but local state update failed"
	default:
		return "cleanup failed"
	}
}
