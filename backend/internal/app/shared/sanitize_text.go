package shared

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var plainTextSanitizer = bluemonday.StrictPolicy()

// isSpoofableFormatRune reports whether r is a Unicode bidi/zero-width/format
// control that can be used to spoof text or smuggle hidden content into plain
// text fields.
func isSpoofableFormatRune(r rune) bool {
	switch r {
	case 0x00AD, 0x061C, 0x180E, 0x200B, 0x200C, 0x200D, 0x200E, 0x200F,
		0x202A, 0x202B, 0x202C, 0x202D, 0x202E, 0x2066, 0x2067, 0x2068,
		0x2069, 0xFEFF:
		return true
	}
	return r >= 0x206A && r <= 0x206F
}

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
		if isSpoofableFormatRune(r) {
			return -1
		}
		return r
	}, cleaned)
}
