package payment

import (
	"net/url"
	"strings"

	appshared "xiaoheiplay/internal/app/shared"
)

func rawToParams(req appshared.RawHTTPRequest) map[string]string {
	out := map[string]string{}
	if req.RawQuery != "" {
		if q, err := url.ParseQuery(req.RawQuery); err == nil {
			for k, v := range q {
				if len(v) > 0 {
					out[k] = v[0]
				}
			}
		}
	}
	if len(req.Body) == 0 {
		return out
	}
	ct := ""
	if values := req.Headers["Content-Type"]; len(values) > 0 {
		ct = values[0]
	}
	if strings.Contains(strings.ToLower(ct), "application/x-www-form-urlencoded") || strings.Contains(strings.ToLower(ct), "multipart/form-data") || strings.Contains(string(req.Body), "=") {
		if q, err := url.ParseQuery(string(req.Body)); err == nil {
			for k, v := range q {
				if len(v) > 0 && out[k] == "" {
					out[k] = v[0]
				}
			}
		}
	}
	return out
}
