package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const maxPaymentCallbackBodyBytes int64 = 1 << 20

func readPaymentCallbackRequest(c *gin.Context) (appshared.RawHTTPRequest, error) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxPaymentCallbackBodyBytes))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return appshared.RawHTTPRequest{}, domain.ErrRequestBodyTooLarge
		}
		return appshared.RawHTTPRequest{}, domain.ErrInvalidBody
	}

	headers := make(map[string][]string, len(c.Request.Header))
	for key, values := range c.Request.Header {
		headers[key] = append([]string(nil), values...)
	}
	// net/http stores the Host header in Request.Host instead of Header. Keep it
	// in the raw callback envelope because some gateway signatures cover the
	// authority along with the request target.
	if c.Request.Host != "" {
		headers["Host"] = []string{c.Request.Host}
	}
	return appshared.RawHTTPRequest{
		Method:   c.Request.Method,
		Path:     c.Request.URL.Path,
		RawQuery: c.Request.URL.RawQuery,
		Headers:  headers,
		Body:     body,
	}, nil
}

func writePaymentCallbackReadError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, domain.ErrRequestBodyTooLarge) {
		status = http.StatusRequestEntityTooLarge
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
