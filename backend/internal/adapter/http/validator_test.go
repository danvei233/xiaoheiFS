package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"xiaoheiplay/internal/domain"
)

// TestValidator_CustomRuleExecutes verifies P2-2: a custom rule registered on
// the shared validator instance actually executes — both through the explicit
// validatePayload call and through gin's ShouldBindJSON pass (which previously
// used a separate validator instance and silently ignored the rule).
func TestValidator_CustomRuleExecutes(t *testing.T) {
	const tag = "no_banana"
	if err := httpPayloadValidator.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return !strings.Contains(strings.ToLower(fl.Field().String()), "banana")
	}); err != nil {
		t.Fatalf("register custom rule: %v", err)
	}
	defer func() {
		// Re-register to remove the rule for other tests.
		_ = httpPayloadValidator.RegisterValidation(tag, func(fl validator.FieldLevel) bool { return true })
	}()

	type dto struct {
		Name string `json:"name" binding:"no_banana"`
	}

	// Direct validatePayload path.
	if err := validatePayload(&dto{Name: "banana-smoothie"}); err == nil {
		t.Fatalf("custom rule must reject a value containing banana via validatePayload")
	}
	if err := validatePayload(&dto{Name: "apple"}); err != nil {
		t.Fatalf("custom rule must accept a compliant value via validatePayload: %v", err)
	}

	// gin binding path uses the same registered instance.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v", func(c *gin.Context) {
		var p dto
		if err := bindJSON(c, &p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v", strings.NewReader(`{"name":"banana"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("gin binding must reject banana via the custom rule, got %d", rec.Code)
	}
}

// TestValidator_UnregisteredTagIsDetected verifies a DTO referencing an
// unregistered rule is surfaced as an error (failing the test/CI) rather than
// silently accepted — a typo'd tag cannot hide behind a runtime 400.
func TestValidator_UnregisteredTagIsDetected(t *testing.T) {
	type badDTO struct {
		Value string `json:"value" binding:"definitely_not_a_registered_rule"`
	}
	err := validatePayload(&badDTO{Value: "x"})
	if err == nil {
		t.Fatalf("an unregistered validation tag must be detected as an error")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("unregistered tag error must wrap ErrInvalidInput, got %v", err)
	}
}

// TestValidator_SingleInstanceSharedWithGin verifies gin's binding.Validator is
// backed by the same instance as validatePayload.
func TestValidator_SingleInstanceSharedWithGin(t *testing.T) {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatalf("gin's validator engine must be *validator.Validate")
	}
	if engine != httpPayloadValidator {
		t.Fatalf("gin binding must share the httpPayloadValidator instance")
	}
}

// TestBindJSON_DistinguishesBindFromValidationErrors locks P3-2: a malformed
// body must be classified as a bind error and a well-formed body that fails a
// field constraint as a validation error, so writeBodyError can answer with
// distinct responses instead of collapsing both into "invalid body".
func TestBindJSON_DistinguishesBindFromValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type dto struct {
		Name string `json:"name" binding:"required"`
	}

	makeCtx := func(body string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}

	// Malformed JSON → ErrBindFailed only.
	c := makeCtx(`{"name": `)
	var p dto
	err := bindJSON(c, &p)
	if err == nil {
		t.Fatal("malformed body must produce an error")
	}
	if !errors.Is(err, domain.ErrBindFailed) {
		t.Fatalf("malformed body must classify as ErrBindFailed, got %v", err)
	}
	if errors.Is(err, domain.ErrValidationFailed) {
		t.Fatalf("malformed body must not classify as ErrValidationFailed")
	}

	// Valid JSON that violates a constraint → ErrValidationFailed only.
	c = makeCtx(`{"name": ""}`)
	p = dto{}
	err = bindJSON(c, &p)
	if err == nil {
		t.Fatal("empty required field must produce an error")
	}
	if !errors.Is(err, domain.ErrValidationFailed) {
		t.Fatalf("validation failure must classify as ErrValidationFailed, got %v", err)
	}
	if errors.Is(err, domain.ErrBindFailed) {
		t.Fatalf("validation failure must not classify as ErrBindFailed")
	}

	// writeBodyError must emit a distinct response per classification.
	rec := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	writeBodyError(c, domain.ErrValidationFailed)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), domain.ErrValidationFailed.Error()) {
		t.Fatalf("validation error response: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	writeBodyError(c, domain.ErrBindFailed)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), domain.ErrInvalidBody.Error()) {
		t.Fatalf("bind error response: code=%d body=%s", rec.Code, rec.Body.String())
	}
}
