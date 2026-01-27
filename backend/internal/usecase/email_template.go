package usecase

import (
	"bytes"
	"html/template"
	"strings"
	texttmpl "text/template"
)

func RenderTemplate(input string, data any, html bool) string {
	if input == "" {
		return ""
	}
	var buf bytes.Buffer
	if html {
		t, err := template.New("html").Parse(input)
		if err != nil {
			return input
		}
		if err := t.Execute(&buf, data); err != nil {
			return input
		}
		return buf.String()
	}
	t, err := texttmpl.New("text").Parse(input)
	if err != nil {
		return input
	}
	if err := t.Execute(&buf, data); err != nil {
		return input
	}
	return buf.String()
}

func IsHTMLContent(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<div") ||
		strings.Contains(lower, "<table") ||
		strings.Contains(lower, "<p") ||
		strings.Contains(lower, "<br")
}
