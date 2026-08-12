package shared

import (
	"bytes"
	"html/template"
	"log"
	"strings"
	texttmpl "text/template"
)

func RenderTemplate(content string, vars any, html bool) string {
	if content == "" {
		return ""
	}
	var data any
	switch v := vars.(type) {
	case map[string]any:
		data = v
	case map[string]string:
		m := make(map[string]any, len(v))
		for key, value := range v {
			m[key] = value
		}
		data = m
	default:
		data = map[string]any{}
	}

	var buf bytes.Buffer
	if html {
		t, err := template.New("html").Parse(content)
		if err != nil {
			log.Printf("render template: html parse error: %v", err)
			return content
		}
		if err := t.Execute(&buf, data); err != nil {
			log.Printf("render template: html execute error: %v", err)
			return content
		}
		return buf.String()
	}
	t, err := texttmpl.New("text").Parse(content)
	if err != nil {
		log.Printf("render template: text parse error: %v", err)
		return content
	}
	if err := t.Execute(&buf, data); err != nil {
		log.Printf("render template: text execute error: %v", err)
		return content
	}
	// Strip CR/LF from plain-text output so template-driven subject fields can
	// never carry SMTP header injection into the mailer.
	return stripCRLF(buf.String())
}

// stripCRLF removes carriage-return and line-feed bytes from plain-text output.
func stripCRLF(value string) string {
	if !strings.ContainsAny(value, "\r\n") {
		return value
	}
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, value)
}

func IsHTMLContent(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<div") ||
		strings.Contains(lower, "<table") ||
		strings.Contains(lower, "<p") ||
		strings.Contains(lower, "<br")
}
