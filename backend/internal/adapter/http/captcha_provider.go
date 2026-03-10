package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		return h.verifyTurnstileCaptcha(settings, captchaCode)
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

func (h *Handler) verifyTurnstileCaptcha(settings authSettings, token string) error {
	if settings.CaptchaCtxForTurnstile.APIEndpoint == nil {
		settings.CaptchaCtxForTurnstile.APIEndpoint = &url.URL{
			Scheme: "https",
			Host:   "challenges.cloudflare.com",
			Path:   "turnstiles/v0/siteverify",
		}

	}
	if func() error { // 网络连通性测试，cloudflare 在中国大陆的访问可能会有问题
		_, err := http.Post(settings.CaptchaCtxForTurnstile.APIEndpoint.String(), "", strings.NewReader(""))
		return err
	}() != nil {
		return domain.ErrCaptchaFailed
	}
	time.Sleep(time.Second * 2) // 防止 cloudflare 的风控限制
	// WIP: 将在下一个提交中实现。
	return nil
}
func hmacEncodeSHA256Hex(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
