package shared

import (
	"bytes"
	"html/template"
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
			return content
		}
		if err := t.Execute(&buf, data); err != nil {
			return content
		}
		return buf.String()
	}
	t, err := texttmpl.New("text").Parse(content)
	if err != nil {
		return content
	}
	if err := t.Execute(&buf, data); err != nil {
		return content
	}
	return buf.String()
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
