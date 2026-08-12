package shared

import (
	"strings"
	"unicode/utf8"
)

// SMSMessage is the request payload for outbound SMS delivery.
type SMSMessage struct {
	TemplateID string
	Content    string
	Vars       map[string]string
	Phones     []string
}

type SMSDelivery struct {
	MessageID string
}

const (
	// MaxSMSPhonesPerSend bounds the recipient count per delivery to prevent
	// bulk-bombing through a single request.
	MaxSMSPhonesPerSend = 50
	// MaxSMSContentRunes bounds the content length (runes) per delivery.
	MaxSMSContentRunes = 1000
	// MinSMSPhoneDigits / MaxSMSPhoneDigits bound the plain digit count of a
	// phone number.
	MinSMSPhoneDigits = 5
	MaxSMSPhoneDigits = 15
)

// Validate performs structural validation on the message. It rejects empty
// recipient lists, oversized recipient counts, malformed phone numbers, empty
// content, and content beyond the size cap.
func (m SMSMessage) Validate() error {
	if len(m.Phones) == 0 {
		return ErrSMSNoRecipient
	}
	if len(m.Phones) > MaxSMSPhonesPerSend {
		return ErrSMSTooManyRecipients
	}
	for _, phone := range m.Phones {
		if !validPhoneShape(strings.TrimSpace(phone)) {
			return ErrSMSInvalidPhone
		}
	}
	if strings.TrimSpace(m.Content) == "" && strings.TrimSpace(m.TemplateID) == "" {
		return ErrSMSContentRequired
	}
	if utf8.RuneCountInString(m.Content) > MaxSMSContentRunes {
		return ErrSMSContentTooLong
	}
	return nil
}

// validPhoneShape accepts an optional leading '+' followed by digits with
// separators (space, dash, parens). The bare digit count must be within bounds.
func validPhoneShape(phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}
	if phone[0] == '+' {
		phone = phone[1:]
	}
	digits := 0
	for _, r := range phone {
		if r == ' ' || r == '-' || r == '(' || r == ')' {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
		digits++
	}
	return digits >= MinSMSPhoneDigits && digits <= MaxSMSPhoneDigits
}
