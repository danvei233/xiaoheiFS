package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"

	"github.com/gin-gonic/gin"
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
	case "Turnstile":
		return "Turnstile"
	default:
		return "image"
	}
}

func (h *Handler) verifyHumanCaptcha(c *gin.Context, settings authSettings, captchaID, captchaCode string, gt geeTestValidatePayload) error {
	switch normalizeCaptchaProvider(settings.CaptchaProvider) {
	case "geeTest":
		return h.verifyGeeTestCaptcha(settings, gt)
	case "Turnstile":
		return h.verifyTurnstileCaptcha(c, settings, captchaCode)
	default:
		return h.authSvc.VerifyCaptcha(c, captchaID, captchaCode)
	}
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Failed to close response body: %v", err)
		}
	}(resp.Body)
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

func (h *Handler) verifyTurnstileCaptcha(c *gin.Context, settings authSettings, token string) error {
	stripPort := func(host string) string {
		if idx := strings.Index(host, ":"); idx > 0 {
			return host[:idx]
		}
		return host
	}

	getRemoteIP := func() string {
		if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
			return ip
		}
		if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
			return strings.Split(ip, ", ")[0]
		}
		return c.ClientIP()
	}

	getSiteverifyURL := func() string {
		if settings.CaptchaCtxForTurnstile.APIEndpoint != nil {
			return settings.CaptchaCtxForTurnstile.APIEndpoint.String()
		}
		return "https://challenges.cloudflare.com/turnstiles/v0/siteverify"
	}

	buildRequestBody := func() []byte {
		bodyRaw := struct {
			Secret   string `json:"secret"`
			Response string `json:"response"`
			Remoteip string `json:"remoteip"`
		}{
			Secret:   settings.CaptchaCtxForTurnstile.Secret,
			Response: token,
			Remoteip: getRemoteIP(),
		}
		body, err := json.Marshal(bodyRaw)
		if err != nil {
			log.Panicf("Failed to marshal json: %v", err)
		}
		return body
	}

	siteverifyURL := getSiteverifyURL()
	req, err := http.NewRequest(http.MethodPost, siteverifyURL, bytes.NewReader(buildRequestBody()))
	if err != nil {
		return domain.ErrCaptchaFailed
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp == nil {
		if resp != nil {
			defer resp.Body.Close()
		}
		return domain.ErrCaptchaFailed
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return domain.ErrCaptchaFailed
	}

	var result struct {
		Success     bool      `json:"success"`
		ChallengeTs time.Time `json:"challenge_ts"`
		Hostname    string    `json:"hostname"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return domain.ErrCaptchaFailed
	}

	expectedHostname := stripPort(c.Request.Host)
	actualHostname := stripPort(result.Hostname)
	if !result.Success || actualHostname != expectedHostname {
		return domain.ErrCaptchaFailed
	}
	return nil
}
func hmacEncodeSHA256Hex(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
