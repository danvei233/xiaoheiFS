package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xiaoheiplay/internal/domain"
)

type geeTestValidatePayload struct {
	LotNumber     string
	CaptchaOutput string
	PassToken     string
	GenTime       string
}

func normalizeCaptchaProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "geetest":
		return "geetest"
	default:
		return "image"
	}
}

func (h *Handler) verifyHumanCaptcha(c *gin.Context, settings authSettings, captchaID, captchaCode string, gt geeTestValidatePayload) error {
	if normalizeCaptchaProvider(settings.CaptchaProvider) == "geetest" {
		return h.verifyGeeTestCaptcha(settings, gt)
	}
	return h.authSvc.VerifyCaptcha(c.Request.Context(), captchaID, captchaCode)
}

func (h *Handler) verifyGeeTestCaptcha(settings authSettings, payload geeTestValidatePayload) error {
	captchaID := strings.TrimSpace(settings.GeeTestCaptchaID)
	captchaKey := strings.TrimSpace(settings.GeeTestCaptchaKey)
	apiServer := strings.TrimRight(strings.TrimSpace(settings.GeeTestAPIServer), "/")
	if captchaID == "" || captchaKey == "" || apiServer == "" {
		return domain.ErrCaptchaFailed
	}
	lotNumber := strings.TrimSpace(payload.LotNumber)
	captchaOutput := strings.TrimSpace(payload.CaptchaOutput)
	passToken := strings.TrimSpace(payload.PassToken)
	genTime := strings.TrimSpace(payload.GenTime)
	if lotNumber == "" || captchaOutput == "" || passToken == "" || genTime == "" {
		return domain.ErrCaptchaFailed
	}

	// Strict freshness gate (P3-20): gen_time must be a legal Unix-seconds
	// number. Missing, non-numeric, zero, negative, too-far-in-the-future or
	// too-old values are all rejected — a captured success payload must not be
	// replayable long after issuance and a malformed value must not skip the
	// check entirely.
	if err := validateGeeTestGenTime(genTime, time.Now()); err != nil {
		return err
	}

	signToken := hmacEncodeSHA256Hex(captchaKey, lotNumber)
	formData := make(url.Values)
	formData.Set("lot_number", lotNumber)
	formData.Set("captcha_output", captchaOutput)
	formData.Set("pass_token", passToken)
	formData.Set("gen_time", genTime)
	formData.Set("sign_token", signToken)

	validateURL := apiServer + "/validate" + "?captcha_id=" + url.QueryEscape(captchaID)
	cli := http.Client{Timeout: 5 * time.Second}
	resp, err := cli.PostForm(validateURL, formData)
	if err != nil {
		return domain.ErrCaptchaFailed
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.ErrCaptchaFailed
	}

	var resMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&resMap); err != nil {
		return domain.ErrCaptchaFailed
	}
	if strings.TrimSpace(strings.ToLower(toString(resMap["result"]))) != "success" {
		return domain.ErrCaptchaFailed
	}
	return nil
}

// captchaFreshness limits bound how old (or how far in the future) a GeeTest
// gen_time may be. gen_time is a Unix-seconds timestamp.
var (
	maxCaptchaAgeSec     = int64((10 * time.Minute).Seconds())
	maxCaptchaFutureSkew = int64((2 * time.Minute).Seconds())
)

// validateGeeTestGenTime requires gen_time to be a legal Unix-seconds number
// within the allowed age and clock-skew window. It is extracted as a pure
// function so every malformed/edge case is directly unit-testable.
func validateGeeTestGenTime(genTime string, now time.Time) error {
	gen, err := strconv.ParseInt(strings.TrimSpace(genTime), 10, 64)
	if err != nil || gen <= 0 {
		return domain.ErrCaptchaFailed
	}
	nowSec := now.Unix()
	if gen > nowSec+maxCaptchaFutureSkew || nowSec-gen > maxCaptchaAgeSec {
		return domain.ErrCaptchaFailed
	}
	return nil
}

func hmacEncodeSHA256Hex(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
