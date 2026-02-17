package passwordreset

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	appshared "xiaoheiplay/internal/app/shared"
)

const (
	maxLenPassword = 128
)

var passwordResetFieldValidator = validator.New()

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := passwordResetFieldValidator.Var(trimmed, fmt.Sprintf("required,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}
