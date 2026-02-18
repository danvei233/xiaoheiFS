package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"io"
	"reflect"
)

var httpPayloadValidator = validator.New()

func validatePayload(v any) error {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		// Validator tags apply to struct payloads only.
		return nil
	}
	return httpPayloadValidator.Struct(v)
}

func bindJSON(c *gin.Context, payload any) error {
	if err := c.ShouldBindJSON(payload); err != nil {
		return err
	}
	return validatePayload(payload)
}

func bindJSONOptional(c *gin.Context, payload any) error {
	if err := c.ShouldBindJSON(payload); err != nil {
		if errors.Is(err, io.EOF) {
			return validatePayload(payload)
		}
		return err
	}
	return validatePayload(payload)
}
