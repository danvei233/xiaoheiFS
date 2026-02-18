package shared

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var plainTextSanitizer = bluemonday.StrictPolicy()

// SanitizePlainText strips HTML tags and unsafe control chars for plain-text fields.
func SanitizePlainText(value string) string {
	cleaned := plainTextSanitizer.Sanitize(value)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, cleaned)
}
