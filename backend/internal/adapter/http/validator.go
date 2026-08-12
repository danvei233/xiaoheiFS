package http

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"xiaoheiplay/internal/domain"
)

// httpPayloadValidator is the single validator instance shared by gin's request
// binding (ShouldBindJSON) and the explicit validatePayload pass. Custom rules
// are registered once here, so they execute in both passes — the previous
// two-instance split made custom rule registration a silent no-op in gin's
// pass (Issue #51 P2-2).
var httpPayloadValidator = newPayloadValidator()

func newPayloadValidator() *validator.Validate {
	v := validator.New()
	// DTOs declare their rules with gin's `binding` tag; the shared instance
	// must read the same tag name as gin or the explicit validatePayload pass
	// would silently no-op (the pre-unification bug).
	v.SetTagName("binding")
	registerCustomValidations(v)
	return v
}

// registerCustomValidations is the single place custom validator rules are
// registered. A rule added here is active for every handler DTO.
func registerCustomValidations(v *validator.Validate) {
	// No custom rules are required today; a rule added here is immediately
	// testable via validatePayload and the HTTP binding pass.
}

// ginValidator adapts httpPayloadValidator to gin's binding.StructValidator so
// ShouldBindJSON and validatePayload use the exact same instance. It mirrors
// gin's default validator in skipping non-struct payloads (maps, slices of
// scalars, etc.) which the go-playground Struct API would reject.
type ginValidator struct {
	v *validator.Validate
}

func (gv ginValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		return gv.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return gv.v.Struct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		for i := 0; i < count; i++ {
			if err := gv.ValidateStruct(value.Index(i).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (gv ginValidator) Engine() any {
	return gv.v
}

func init() {
	binding.Validator = ginValidator{v: httpPayloadValidator}
}

func validatePayload(v any) (err error) {
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
	// A panic inside the validator (e.g. an unregistered rule) must not crash
	// the process. The panic is logged loudly rather than silently mapped to a
	// 400: the unregistered-tag regression test fails the build, so a typo'd
	// rule is caught in CI, not at runtime.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("validator panic: %v (an unregistered validation tag is a bug; fix the DTO or register the rule)", r)
			err = fmt.Errorf("%w: validator panic: %v", domain.ErrInvalidInput, r)
		}
	}()
	return httpPayloadValidator.Struct(v)
}

// normalizeBindError maps a body read that exceeded the request cap to the
// stable ErrRequestBodyTooLarge sentinel so the caller can respond with 413 and
// never leak the underlying *http.MaxBytesError (which embeds no body, but its
// text varies by transport) or the request payload into logs/errors.
func normalizeBindError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return domain.ErrRequestBodyTooLarge
	}
	return err
}

// classifiedError tags an error with a classification sentinel (ErrBindFailed or
// ErrValidationFailed) while preserving the underlying error's message and
// unwrap chain. Handlers that echo err.Error() directly therefore see exactly
// the message they saw before the classification was introduced, while
// writeBodyError can tell a malformed body apart from a field-level validation
// failure (P3-2).
type classifiedError struct {
	err  error
	kind error
}

func (e *classifiedError) Error() string { return e.err.Error() }
func (e *classifiedError) Unwrap() error { return e.err }
func (e *classifiedError) Is(target error) bool {
	return target == e.kind
}

// classifyBindError tags a body-binding failure (malformed JSON, wrong type,
// unexpected EOF, ...) as ErrBindFailed. Gin's ShouldBindJSON also runs field
// validation, so a validator.ValidationErrors surfacing on the bind path is
// re-classified as a validation failure (P3-2). The request-too-large sentinel
// is passed through untouched so writeBodyError still answers 413.
func classifyBindError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrRequestBodyTooLarge) {
		return err
	}
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		return classifyValidationError(err)
	}
	return &classifiedError{err: err, kind: domain.ErrBindFailed}
}

// classifyValidationError tags a field-level validation failure as
// ErrValidationFailed.
func classifyValidationError(err error) error {
	if err == nil {
		return nil
	}
	return &classifiedError{err: err, kind: domain.ErrValidationFailed}
}

func bindJSON(c *gin.Context, payload any) error {
	if err := c.ShouldBindJSON(payload); err != nil {
		return classifyBindError(normalizeBindError(err))
	}
	return classifyValidationError(validatePayload(payload))
}

func bindJSONOptional(c *gin.Context, payload any) error {
	if err := c.ShouldBindJSON(payload); err != nil {
		if errors.Is(err, io.EOF) {
			return classifyValidationError(validatePayload(payload))
		}
		return classifyBindError(normalizeBindError(err))
	}
	return classifyValidationError(validatePayload(payload))
}

// writeBodyError writes the controlled error response for a failed body bind:
// 413 when the request body exceeded the configured cap, 400 with a
// validation-specific message for a field-level validation failure, and the
// standard invalid-body 400 for a malformed body. It never echoes the
// underlying error text or the request body.
func writeBodyError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrRequestBodyTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": domain.ErrRequestBodyTooLarge.Error()})
		return
	}
	if errors.Is(err, domain.ErrValidationFailed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrValidationFailed.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
}
