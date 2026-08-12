package http

import (
	"bytes"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
)

func TestReadPaymentCallbackRequestPreservesSigningInputs(t *testing.T) {
	requests := []struct {
		name        string
		contentType string
		body        []byte
	}{
		{
			name:        "json",
			contentType: "application/json",
			body:        []byte(`{"status":"body","cipher":"a=b"}`),
		},
		{
			name:        "multipart",
			contentType: "multipart/form-data; boundary=raw-boundary",
			body:        []byte("--raw-boundary\r\nContent-Disposition: form-data; name=\"status\"\r\n\r\nbody\r\n--raw-boundary--\r\n"),
		},
	}

	for _, tt := range requests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(
				"POST",
				"http://example.test/api/v1/payments/notify/demo?status=query&opaque=%2f",
				bytes.NewReader(tt.body),
			)
			req.Header["Content-Type"] = []string{tt.contentType}
			req.Header["X-Signature"] = []string{"first", "second"}
			ctx.Request = req

			got, err := readPaymentCallbackRequest(ctx)
			if err != nil {
				t.Fatalf("read callback: %v", err)
			}
			wantHeaders := req.Header.Clone()
			wantHeaders["Host"] = []string{"example.test"}
			want := appshared.RawHTTPRequest{
				Method:   "POST",
				Path:     "/api/v1/payments/notify/demo",
				RawQuery: "status=query&opaque=%2f",
				Headers:  wantHeaders,
				Body:     tt.body,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("callback signing inputs changed\nwant: %#v\n got: %#v", want, got)
			}
		})
	}
}
