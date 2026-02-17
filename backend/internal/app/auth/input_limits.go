package auth

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	appshared "xiaoheiplay/internal/app/shared"
)

const (
	maxLenUsername = 64
	maxLenEmail    = 254
	maxLenQQ       = 32
	maxLenPhone    = 32
	maxLenBio      = 512
	maxLenIntro    = 1024
	maxLenPassword = 128
)

var authFieldValidator = validator.New()

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := authFieldValidator.Var(trimmed, fmt.Sprintf("required,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}

func trimAndValidateOptional(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := authFieldValidator.Var(trimmed, fmt.Sprintf("omitempty,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}
