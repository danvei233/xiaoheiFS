package shared

import (
	"strings"
	"testing"
)

func TestSMSMessageValidate(t *testing.T) {
	valid := SMSMessage{
		TemplateID: "tpl",
		Content:    "hello",
		Phones:     []string{"+8613800138000"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}
}

func TestSMSMessageValidateRejectsNoRecipient(t *testing.T) {
	msg := SMSMessage{TemplateID: "tpl", Content: "hello", Phones: nil}
	if err := msg.Validate(); err == nil {
		t.Fatalf("expected no-recipient error")
	}
}

func TestSMSMessageValidateRejectsTooManyRecipients(t *testing.T) {
	phones := make([]string, MaxSMSPhonesPerSend+1)
	for i := range phones {
		phones[i] = "13800138000"
	}
	msg := SMSMessage{TemplateID: "tpl", Content: "hello", Phones: phones}
	if err := msg.Validate(); err == nil {
		t.Fatalf("expected too-many-recipients error")
	}
}

func TestSMSMessageValidateRejectsMalformedPhone(t *testing.T) {
	msg := SMSMessage{TemplateID: "tpl", Content: "hello", Phones: []string{"not-a-phone"}}
	if err := msg.Validate(); err == nil {
		t.Fatalf("expected invalid-phone error")
	}
}

func TestSMSMessageValidateRejectsEmptyContent(t *testing.T) {
	msg := SMSMessage{Content: "   ", Phones: []string{"13800138000"}}
	if err := msg.Validate(); err == nil {
		t.Fatalf("expected content-required error")
	}
}

func TestSMSMessageValidateRejectsOverlongContent(t *testing.T) {
	msg := SMSMessage{TemplateID: "tpl", Content: strings.Repeat("a", MaxSMSContentRunes+1), Phones: []string{"13800138000"}}
	if err := msg.Validate(); err == nil {
		t.Fatalf("expected content-too-long error")
	}
}
