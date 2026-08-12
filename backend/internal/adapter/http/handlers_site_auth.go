package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (h *Handler) loadAuthSettings(ctx context.Context) authSettings {
	get := func(key string) string {
		s, err := h.getSettingByContext(ctx, key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(s.ValueJSON)
	}
	getBool := func(key string, def bool) bool {
		val := strings.ToLower(strings.TrimSpace(get(key)))
		if val == "" {
			return def
		}
		return val == "true" || val == "1" || val == "yes"
	}
	getInt := func(key string, def int) int {
		val := strings.TrimSpace(get(key))
		if val == "" {
			return def
		}
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
		return def
	}
	getString := func(key, def string) string {
		val := strings.TrimSpace(get(key))
		if val == "" {
			return def
		}
		return val
	}
	getCodeLength := func(key string, def int) int {
		n := getInt(key, def)
		if n < 4 {
			return 4
		}
		if n > 12 {
			return 12
		}
		return n
	}
	getCodeComplexity := func(key, def string) string {
		v := strings.ToLower(strings.TrimSpace(getString(key, def)))
		switch v {
		case appshared.CodeComplexityDigits, appshared.CodeComplexityLetters, appshared.CodeComplexityAlnum:
			return v
		default:
			return def
		}
	}
	getStringSlice := func(key string, def []string) []string {
		raw := get(key)
		if raw == "" {
			return def
		}
		var out []string
		if err := json.Unmarshal([]byte(raw), &out); err == nil {
			return out
		}
		return def
	}

	verifyType := strings.ToLower(getString("auth_register_verify_type", "none"))
	if verifyType != "email" && verifyType != "sms" {
		verifyType = "none"
	}
	verifyChannels := normalizeChannels(getStringSlice("auth_register_verify_channels", nil))
	if len(verifyChannels) == 0 {
		switch verifyType {
		case "email":
			verifyChannels = []string{"email"}
		case "sms":
			verifyChannels = []string{"sms"}
		default:
			verifyChannels = []string{}
		}
	}

	return authSettings{
		RegisterEnabled:                getBool("auth_register_enabled", true),
		RegisterRequiredFields:         getStringSlice("auth_register_required_fields", []string{"username", "password"}),
		RegisterEmailRequired:          getBool("auth_register_email_required", true),
		PasswordMinLen:                 getInt("auth_password_min_len", 6),
		PasswordRequireUpper:           getBool("auth_password_require_upper", false),
		PasswordRequireLower:           getBool("auth_password_require_lower", false),
		PasswordRequireNumber:          getBool("auth_password_require_number", false),
		PasswordRequireSymbol:          getBool("auth_password_require_symbol", false),
		RegisterVerifyType:             verifyType,
		RegisterVerifyChannels:         verifyChannels,
		RegisterVerifyTTL:              time.Duration(getInt("auth_register_verify_ttl_sec", 600)) * time.Second,
		RegisterCaptchaEnabled:         getBool("auth_register_captcha_enabled", true),
		CaptchaProvider:                normalizeCaptchaProvider(getString("auth_captcha_provider", "image")),
		GeeTestCaptchaID:               getString("auth_geetest_captcha_id", ""),
		GeeTestCaptchaKey:              getString("auth_geetest_captcha_key", ""),
		GeeTestAPIServer:               strings.TrimRight(getString("auth_geetest_api_server", "https://gcaptcha4.geetest.com"), "/"),
		RegisterEmailSubject:           getString("auth_register_email_subject", "Your verification code"),
		RegisterEmailBody:              getString("auth_register_email_body", "Your verification code is: {{code}}"),
		RegisterSMSPluginID:            getString("auth_register_sms_plugin_id", getString("sms_plugin_id", "")),
		RegisterSMSInstanceID:          getString("auth_register_sms_instance_id", getString("sms_instance_id", "default")),
		RegisterSMSTemplateID:          getString("auth_register_sms_template_id", getString("sms_provider_template_id", "")),
		LoginCaptchaEnabled:            getBool("auth_login_captcha_enabled", false),
		LoginRateLimitEnabled:          getBool("auth_login_rate_limit_enabled", true),
		LoginRateLimitWindow:           time.Duration(getInt("auth_login_rate_limit_window_sec", 300)) * time.Second,
		LoginRateLimitMax:              getInt("auth_login_rate_limit_max_attempts", 5),
		LoginNotifyEnabled:             getBool("auth_login_notify_enabled", true),
		LoginNotifyOnFirst:             getBool("auth_login_notify_on_first_login", true),
		LoginNotifyOnIPChange:          getBool("auth_login_notify_on_ip_change", true),
		LoginNotifyChannels:            normalizeChannels(getStringSlice("auth_login_notify_channels", []string{"email"})),
		PasswordResetEnabled:           getBool("auth_password_reset_enabled", true),
		PasswordResetChannels:          normalizeChannels(getStringSlice("auth_password_reset_channels", []string{"email"})),
		PasswordResetVerifyTTL:         time.Duration(getInt("auth_password_reset_verify_ttl_sec", 600)) * time.Second,
		SMSCodeLength:                  getCodeLength("auth_sms_code_len", 6),
		SMSCodeComplexity:              getCodeComplexity("auth_sms_code_complexity", appshared.CodeComplexityDigits),
		EmailCodeLength:                getCodeLength("auth_email_code_len", 6),
		EmailCodeComplexity:            getCodeComplexity("auth_email_code_complexity", appshared.CodeComplexityAlnum),
		CaptchaLength:                  getCodeLength("auth_captcha_code_len", 5),
		CaptchaComplexity:              getCodeComplexity("auth_captcha_code_complexity", appshared.CodeComplexityAlnum),
		EmailBindEnabled:               getBool("auth_email_bind_enabled", true),
		PhoneBindEnabled:               getBool("auth_phone_bind_enabled", true),
		ContactChangeNotifyOldEnabled:  getBool("auth_contact_change_notify_old_enabled", true),
		ContactBindVerifyTTL:           time.Duration(getInt("auth_contact_bind_verify_ttl_sec", 600)) * time.Second,
		BindRequirePasswordWhenNo2FA:   getBool("auth_bind_require_password_when_no_2fa", false),
		RebindRequirePasswordWhenNo2FA: getBool("auth_rebind_require_password_when_no_2fa", true),
		TwoFAEnabled:                   getBool("auth_2fa_enabled", true),
		TwoFABindEnabled:               getBool("auth_2fa_bind_enabled", true),
		TwoFARebindEnabled:             getBool("auth_2fa_rebind_enabled", true),
		GeoIPMMDBPath:                  getString("auth_geoip_mmdb_path", ""),
	}
}

func validatePasswordBySettings(password string, s authSettings) error {
	if strings.TrimSpace(password) == "" {
		return appshared.ErrInvalidInput
	}
	if s.PasswordMinLen > 0 && len(password) < s.PasswordMinLen {
		return domain.ErrInvalidInput
	}
	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasNumber = true
		default:
			hasSymbol = true
		}
	}
	if s.PasswordRequireUpper && !hasUpper {
		return domain.ErrInvalidInput
	}
	if s.PasswordRequireLower && !hasLower {
		return domain.ErrInvalidInput
	}
	if s.PasswordRequireNumber && !hasNumber {
		return domain.ErrInvalidInput
	}
	if s.PasswordRequireSymbol && !hasSymbol {
		return domain.ErrInvalidInput
	}
	return nil
}

func (h *Handler) Captcha(c *gin.Context) {
	// Bound captcha issuance per source so the captcha table cannot be flooded
	// into an unbounded size (P3-18). The counter is durable and cross-instance
	// so N replicas cannot each issue 30 captchas for the same source (P1-1).
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	if !h.durableAllow(c.Request.Context(), "captcha:ip:"+ip, 30, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	if settings.CaptchaProvider == "geetest" {
		c.JSON(http.StatusOK, gin.H{
			"captcha_provider": "geetest",
			"captcha_id":       settings.GeeTestCaptchaID,
			"api_server":       settings.GeeTestAPIServer,
		})
		return
	}
	captcha, code, err := h.authSvc.CreateCaptchaWithPolicy(c.Request.Context(), 5*time.Minute, settings.CaptchaLength, settings.CaptchaComplexity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrCaptchaError.Error()})
		return
	}
	img := renderCaptcha(code)
	var buf strings.Builder
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	defer func(enc io.WriteCloser) {
		_ = enc.Close()
	}(enc)
	if err := png.Encode(enc, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrCaptchaEncodeError.Error()})
		return
	}
	_ = enc.Close()
	c.JSON(http.StatusOK, gin.H{
		"captcha_provider": "image",
		"captcha_id":       captcha.ID,
		"image_base64":     buf.String(),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var payload struct {
		Username      string `json:"username" binding:"required"`
		Email         string `json:"email"`
		QQ            string `json:"qq"`
		Phone         string `json:"phone"`
		Password      string `json:"password" binding:"required"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaCode   string `json:"captcha_code"`
		LotNumber     string `json:"lot_number"`
		CaptchaOutput string `json:"captcha_output"`
		PassToken     string `json:"pass_token"`
		GenTime       string `json:"gen_time"`
		VerifyCode    string `json:"verify_code"`
		VerifyChannel string `json:"verify_channel"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	if !settings.RegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrRegistrationDisabled.Error()})
		return
	}
	required := map[string]bool{
		"username": true,
		"password": true,
	}
	for _, f := range settings.RegisterRequiredFields {
		key := strings.ToLower(strings.TrimSpace(f))
		if key == "email" {
			continue
		}
		required[key] = true
	}
	if settings.RegisterEmailRequired {
		required["email"] = true
	}
	requestedVerifyChannel := strings.ToLower(strings.TrimSpace(payload.VerifyChannel))
	if requestedVerifyChannel == "" && len(settings.RegisterVerifyChannels) == 1 {
		requestedVerifyChannel = settings.RegisterVerifyChannels[0]
	}
	if requestedVerifyChannel == "sms" {
		required["email"] = false
	}
	if required["username"] && strings.TrimSpace(payload.Username) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUsernameRequired.Error()})
		return
	}
	if required["email"] && strings.TrimSpace(payload.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailRequired.Error()})
		return
	}
	if required["phone"] && strings.TrimSpace(payload.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
		return
	}
	if required["qq"] && strings.TrimSpace(payload.QQ) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrQqRequired.Error()})
		return
	}
	if err := validatePasswordBySettings(payload.Password, settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(settings.RegisterVerifyChannels) > 0 && settings.RegisterCaptchaEnabled {
		if err := h.verifyHumanCaptcha(c, settings, payload.CaptchaID, payload.CaptchaCode, geeTestValidatePayload{
			LotNumber:     payload.LotNumber,
			CaptchaOutput: payload.CaptchaOutput,
			PassToken:     payload.PassToken,
			GenTime:       payload.GenTime,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCaptchaFailed.Error()})
			return
		}
	}
	verifiedChannel := ""
	if len(settings.RegisterVerifyChannels) > 0 {
		code := strings.TrimSpace(payload.VerifyCode)
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrVerificationCodeRequired.Error()})
			return
		}
		ch := strings.ToLower(strings.TrimSpace(payload.VerifyChannel))
		if ch == "" && len(settings.RegisterVerifyChannels) == 1 {
			ch = settings.RegisterVerifyChannels[0]
		}
		if !hasChannel(settings.RegisterVerifyChannels, ch) {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrVerifyChannelNotAllowed.Error()})
			return
		}
		switch ch {
		case "email":
			if strings.TrimSpace(payload.Email) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailRequired.Error()})
				return
			}
			if err := h.authSvc.VerifyVerificationCode(c.Request.Context(), "email", strings.TrimSpace(payload.Email), "register", code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidVerificationCode.Error()})
				return
			}
		case "sms":
			if strings.TrimSpace(payload.Phone) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
				return
			}
			if err := h.authSvc.VerifyVerificationCode(c.Request.Context(), "sms", strings.TrimSpace(payload.Phone), "register", code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidVerificationCode.Error()})
				return
			}
		}
		verifiedChannel = ch
	}
	switch verifiedChannel {
	case "email":
		payload.Phone = ""
	case "sms":
		payload.Email = ""
	}
	captchaID := strings.TrimSpace(payload.CaptchaID)
	captchaCode := strings.TrimSpace(payload.CaptchaCode)
	captchaRequired := settings.RegisterCaptchaEnabled && len(settings.RegisterVerifyChannels) == 0
	// OTP-based registration already verifies captcha in handler; avoid double consume.
	if len(settings.RegisterVerifyChannels) > 0 {
		captchaID = ""
		captchaCode = ""
	}
	user, err := h.authSvc.Register(c.Request.Context(), appshared.RegisterInput{
		Username:        payload.Username,
		Email:           payload.Email,
		QQ:              payload.QQ,
		Phone:           payload.Phone,
		Password:        payload.Password,
		CaptchaID:       captchaID,
		CaptchaCode:     captchaCode,
		CaptchaRequired: captchaRequired,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrRealNameRequired) || errors.Is(err, appshared.ErrForbidden) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
}

func (h *Handler) Login(c *gin.Context) {
	var payload struct {
		Username      string `json:"username" binding:"required"`
		Password      string `json:"password" binding:"required"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaCode   string `json:"captcha_code"`
		LotNumber     string `json:"lot_number"`
		CaptchaOutput string `json:"captcha_output"`
		PassToken     string `json:"pass_token"`
		GenTime       string `json:"gen_time"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	if settings.LoginRateLimitEnabled {
		// Durable cross-instance login rate limit: a single account+IP budget is
		// shared by all replicas so an attacker spraying across instances cannot
		// multiply the allowed attempts (P1-1).
		key := "login:" + strings.ToLower(strings.TrimSpace(payload.Username)) + ":" + strings.TrimSpace(c.ClientIP())
		if !h.durableAllow(c.Request.Context(), key, settings.LoginRateLimitMax, settings.LoginRateLimitWindow) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyAttempts.Error()})
			return
		}
	}
	if settings.LoginCaptchaEnabled {
		if err := h.verifyHumanCaptcha(c, settings, payload.CaptchaID, payload.CaptchaCode, geeTestValidatePayload{
			LotNumber:     payload.LotNumber,
			CaptchaOutput: payload.CaptchaOutput,
			PassToken:     payload.PassToken,
			GenTime:       payload.GenTime,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCaptchaFailed.Error()})
			return
		}
	}
	user, err := h.authSvc.LoginUser(c.Request.Context(), payload.Username, payload.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidCredentials.Error()})
		return
	}
	h.postLoginSecurityHook(c, user, settings)
	familyID := newTokenID()
	accessToken, err := h.signAuthToken(user.ID, string(user.Role), user.PasswordVersion, 24*time.Hour, "access", familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthToken(user.ID, string(user.Role), user.PasswordVersion, 7*24*time.Hour, "refresh", familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    tokenExpirySeconds,
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"role":         user.Role,
			"totp_enabled": user.TOTPEnabled,
			"email_bound":  strings.TrimSpace(user.Email) != "",
			"phone_bound":  strings.TrimSpace(user.Phone) != "",
			"email_masked": maskEmail(user.Email),
			"phone_masked": maskPhone(user.Phone),
		},
	})
}

func (h *Handler) Logout(c *gin.Context) {
	familyID, _ := c.Get("token_family_id")
	family, _ := familyID.(string)
	if strings.TrimSpace(family) != "" {
		if err := h.authSvc.RevokeRefreshTokenFamily(c.Request.Context(), family, time.Now().Add(8*24*time.Hour)); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AuthSettings(c *gin.Context) {
	settings := h.loadAuthSettings(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"register_enabled":                         settings.RegisterEnabled,
		"register_required_fields":                 settings.RegisterRequiredFields,
		"register_email_required":                  settings.RegisterEmailRequired,
		"register_verify_ttl_sec":                  int(settings.RegisterVerifyTTL / time.Second),
		"password_min_len":                         settings.PasswordMinLen,
		"password_require_upper":                   settings.PasswordRequireUpper,
		"password_require_lower":                   settings.PasswordRequireLower,
		"password_require_number":                  settings.PasswordRequireNumber,
		"password_require_symbol":                  settings.PasswordRequireSymbol,
		"register_verify_type":                     settings.RegisterVerifyType,
		"register_verify_channels":                 settings.RegisterVerifyChannels,
		"register_captcha_enabled":                 settings.RegisterCaptchaEnabled,
		"login_captcha_enabled":                    settings.LoginCaptchaEnabled,
		"captcha_provider":                         settings.CaptchaProvider,
		"auth_geetest_captcha_id":                  settings.GeeTestCaptchaID,
		"auth_geetest_api_server":                  settings.GeeTestAPIServer,
		"auth_login_notify_enabled":                settings.LoginNotifyEnabled,
		"auth_login_notify_on_first_login":         settings.LoginNotifyOnFirst,
		"auth_login_notify_on_ip_change":           settings.LoginNotifyOnIPChange,
		"auth_login_notify_channels":               settings.LoginNotifyChannels,
		"auth_password_reset_enabled":              settings.PasswordResetEnabled,
		"auth_password_reset_channels":             settings.PasswordResetChannels,
		"auth_password_reset_verify_ttl_sec":       int(settings.PasswordResetVerifyTTL / time.Second),
		"auth_sms_code_len":                        settings.SMSCodeLength,
		"auth_sms_code_complexity":                 settings.SMSCodeComplexity,
		"auth_email_code_len":                      settings.EmailCodeLength,
		"auth_email_code_complexity":               settings.EmailCodeComplexity,
		"auth_captcha_code_len":                    settings.CaptchaLength,
		"auth_captcha_code_complexity":             settings.CaptchaComplexity,
		"auth_email_bind_enabled":                  settings.EmailBindEnabled,
		"auth_phone_bind_enabled":                  settings.PhoneBindEnabled,
		"auth_contact_change_notify_old_enabled":   settings.ContactChangeNotifyOldEnabled,
		"auth_contact_bind_verify_ttl_sec":         int(settings.ContactBindVerifyTTL / time.Second),
		"auth_bind_require_password_when_no_2fa":   settings.BindRequirePasswordWhenNo2FA,
		"auth_rebind_require_password_when_no_2fa": settings.RebindRequirePasswordWhenNo2FA,
		"auth_2fa_enabled":                         settings.TwoFAEnabled,
		"auth_2fa_bind_enabled":                    settings.TwoFABindEnabled,
		"auth_2fa_rebind_enabled":                  settings.TwoFARebindEnabled,
	})
}

func (h *Handler) RegisterCode(c *gin.Context) {
	var payload struct {
		Channel       string `json:"channel"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaCode   string `json:"captcha_code"`
		LotNumber     string `json:"lot_number"`
		CaptchaOutput string `json:"captcha_output"`
		PassToken     string `json:"pass_token"`
		GenTime       string `json:"gen_time"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	if !settings.RegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrRegistrationDisabled.Error()})
		return
	}
	if settings.RegisterCaptchaEnabled {
		if err := h.verifyHumanCaptcha(c, settings, payload.CaptchaID, payload.CaptchaCode, geeTestValidatePayload{
			LotNumber:     payload.LotNumber,
			CaptchaOutput: payload.CaptchaOutput,
			PassToken:     payload.PassToken,
			GenTime:       payload.GenTime,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCaptchaFailed.Error()})
			return
		}
	}

	channel := strings.ToLower(strings.TrimSpace(payload.Channel))
	if channel == "" {
		if len(settings.RegisterVerifyChannels) == 1 {
			channel = settings.RegisterVerifyChannels[0]
		} else if settings.RegisterVerifyType != "none" {
			channel = settings.RegisterVerifyType
		}
	}
	if !hasChannel(settings.RegisterVerifyChannels, channel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrChannelNotEnabled.Error()})
		return
	}
	switch channel {
	case "email":
		emailVal := strings.TrimSpace(payload.Email)
		if emailVal == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailRequired.Error()})
			return
		}
		if !h.durableAllow(c.Request.Context(), "register_code:email:"+strings.ToLower(emailVal), 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
			return
		}
		code, err := h.authSvc.CreateVerificationCodeWithPolicy(c.Request.Context(), "email", emailVal, "register", settings.RegisterVerifyTTL, settings.EmailCodeLength, settings.EmailCodeComplexity)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		subject, body, ok := h.renderEmailTemplateByName(c, "register_verify_code", map[string]string{
			"code":  code,
			"email": emailVal,
		})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailTemplateRegisterVerifyCodeNotConfigured.Error()})
			return
		}
		if h.emailSender == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailSenderNotConfigured.Error()})
			return
		}
		if err := h.emailSender.Send(c.Request.Context(), emailVal, subject, body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailSendFailed.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	case "sms":
		phoneVal := strings.TrimSpace(payload.Phone)
		if phoneVal == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
			return
		}
		if settings.RegisterSMSPluginID == "" || h.smsSender == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSMSPluginNotConfigured.Error()})
			return
		}
		if !h.durableAllow(c.Request.Context(), "register_code:sms:"+phoneVal, 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
			return
		}
		code, err := h.authSvc.CreateVerificationCodeWithPolicy(c.Request.Context(), "sms", phoneVal, "register", settings.RegisterVerifyTTL, settings.SMSCodeLength, settings.SMSCodeComplexity)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		msg := appshared.SMSMessage{
			TemplateID: settings.RegisterSMSTemplateID,
			Vars: map[string]string{
				"code": code,
			},
			Phones: []string{phoneVal},
		}
		content, ok := h.renderSMSTemplateByName(c, "register_verify_code", map[string]any{
			"code":  code,
			"phone": phoneVal,
			"now":   time.Now().Format(time.RFC3339),
		})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSmsTemplateRegisterVerifyCodeNotConfigured.Error()})
			return
		}
		msg.Content = content
		cCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		if _, err := h.smsSender.Send(cCtx, settings.RegisterSMSPluginID, settings.RegisterSMSInstanceID, msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSmsSendFailed.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrVerificationNotEnabled.Error()})
		return
	}
}

func (h *Handler) postLoginSecurityHook(c *gin.Context, user domain.User, settings authSettings) {
	if !settings.LoginNotifyEnabled {
		return
	}
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	firstLogin := user.LastLoginAt == nil
	ipChanged := !firstLogin && strings.TrimSpace(user.LastLoginIP) != "" && strings.TrimSpace(user.LastLoginIP) != ip
	shouldNotify := (settings.LoginNotifyOnFirst && firstLogin) || (settings.LoginNotifyOnIPChange && ipChanged)
	if !shouldNotify {
		_ = h.authSvc.UpdateLoginSecurity(c.Request.Context(), user.ID, ip, user.LastLoginCity, user.LastLoginTZ, time.Now())
		return
	}
	city, tz := h.resolveGeoByIP(c.Request.Context(), ip, settings.GeoIPMMDBPath)
	loginTime := time.Now()
	timeText := loginTime.Format("01/02 15:04")
	if strings.TrimSpace(tz) == "" {
		tz = "GMT+00:00"
	}
	_ = h.sendSecurityMessage(c, settings.LoginNotifyChannels, "login_ip_change_alert", user, map[string]string{
		"ip":       ip,
		"city":     city,
		"tz":       tz,
		"time":     fmt.Sprintf("%s (%s)", timeText, tz),
		"username": user.Username,
	})
	_ = h.authSvc.UpdateLoginSecurity(c.Request.Context(), user.ID, ip, city, tz, loginTime)
}

func (h *Handler) resolveGeoByIP(ctx context.Context, ip, mmdbPath string) (string, string) {
	defaultTZ := time.Now().Format("GMT-07:00")
	resolver := h.geoResolver
	if resolver == nil {
		resolver = NewMMDBGeoResolver()
		h.geoResolver = resolver
	}
	path := strings.TrimSpace(mmdbPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("AUTH_GEOIP_MMDB_PATH"))
	}
	if path == "" {
		path = strings.TrimSpace(os.Getenv("GEOIP_MMDB_PATH"))
	}
	if path == "" {
		path = strings.TrimSpace(os.Getenv("GEOIP_DB_PATH"))
	}
	city, tz, err := resolver.Resolve(ctx, ip, path)
	if err != nil {
		return "未知地区", defaultTZ
	}
	if strings.TrimSpace(city) == "" {
		city = "未知地区"
	}
	if strings.TrimSpace(tz) == "" {
		tz = defaultTZ
	}
	return city, tz
}

func (h *Handler) sendSecurityMessage(c *gin.Context, channels []string, templateName string, user domain.User, vars map[string]string) error {
	if len(channels) == 0 {
		return domain.ErrNoMessageChannelConfigured
	}
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return domain.ErrTemplateNameRequired
	}
	sent := 0
	var lastErr error
	for _, ch := range channels {
		switch ch {
		case "email":
			emailAddr := strings.TrimSpace(user.Email)
			if emailAddr == "" {
				lastErr = domain.ErrEmailNotBound
				continue
			}
			subject, body, ok := h.renderEmailTemplateByName(c, templateName, vars)
			if !ok {
				lastErr = fmt.Errorf("%w: email template %s not configured", domain.ErrTemplateNotFound, templateName)
				continue
			}
			if h.emailSender == nil {
				lastErr = domain.ErrEmailSenderNotConfigured
				continue
			}
			if err := h.emailSender.Send(c.Request.Context(), emailAddr, subject, body); err != nil {
				lastErr = err
				continue
			}
			sent++
		case "sms":
			if h.smsSender == nil {
				lastErr = domain.ErrSMSPluginManagerUnavailable
				continue
			}
			phone := strings.TrimSpace(user.Phone)
			if phone == "" {
				lastErr = domain.ErrPhoneNotBound
				continue
			}
			pluginID := strings.TrimSpace(h.getSettingValueByKey(c, "sms_plugin_id"))
			instanceID := strings.TrimSpace(h.getSettingValueByKey(c, "sms_instance_id"))
			if instanceID == "" {
				instanceID = "default"
			}
			if pluginID == "" {
				lastErr = domain.ErrSMSPluginNotConfigured
				continue
			}
			providerTemplateID := strings.TrimSpace(h.getSettingValueByKey(c, "sms_provider_template_id"))
			m := map[string]any{}
			for k, v := range vars {
				m[k] = v
			}
			content, ok := h.renderSMSTemplateByName(c, templateName, m)
			if !ok {
				lastErr = fmt.Errorf("%w: sms template %s not configured", domain.ErrTemplateNotFound, templateName)
				continue
			}
			msgVars := map[string]string{}
			for k, v := range vars {
				msgVars[k] = v
			}
			_, err := h.smsSender.Send(c.Request.Context(), pluginID, instanceID, appshared.SMSMessage{
				TemplateID: providerTemplateID,
				Content:    content,
				Vars:       msgVars,
				Phones:     []string{phone},
			})
			if err != nil {
				lastErr = err
				continue
			}
			sent++
		}
	}
	if sent > 0 {
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return domain.ErrNoAvailableChannel
}

func (h *Handler) renderEmailTemplateByName(ctx *gin.Context, name string, vars map[string]string) (string, string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false
	}
	templates, err := h.listEmailTemplates(ctx)
	if err != nil {
		return "", "", false
	}
	for _, tmpl := range templates {
		if strings.TrimSpace(tmpl.Name) != name || !tmpl.Enabled {
			continue
		}
		subjectTpl := normalizeSimpleTemplateVars(tmpl.Subject)
		bodyTpl := normalizeSimpleTemplateVars(tmpl.Body)
		subject := appshared.RenderTemplate(subjectTpl, vars, false)
		body := appshared.RenderTemplate(bodyTpl, vars, appshared.IsHTMLContent(bodyTpl))
		return subject, body, true
	}
	return "", "", false
}

func (h *Handler) Refresh(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	claims, err := h.parseRefreshToken(c.Request.Context(), payload.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleUser) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	familyID, _ := claims["fid"].(string)
	if strings.TrimSpace(familyID) == "" {
		// Adopt refresh tokens issued before family tracking was deployed. Their
		// unique jti becomes the root family so upgrades do not force every active
		// user to log in again.
		familyID, _ = claims["jti"].(string)
		if strings.TrimSpace(familyID) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
	}
	user, err := h.authSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	accessToken, err := h.signAuthToken(userID, role, user.PasswordVersion, 24*time.Hour, "access", familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	newRefreshToken, err := h.signAuthToken(userID, role, user.PasswordVersion, 7*24*time.Hour, "refresh", familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	rotated, familyRevoked, err := h.authSvc.RotateRefreshToken(
		c.Request.Context(), payload.RefreshToken, newRefreshToken, familyID,
		expiresAt.Time, time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
		return
	}
	if !rotated || familyRevoked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    tokenExpirySeconds,
	})
}

// Token issuer and audience. These are written into every issued token and
// strictly re-validated on parse so a token minted for another deployment or
// another audience is rejected, not just decoded (Issue #51 P3-22 hardening).
const (
	jwtIssuer   = "xiaoheiplay"
	jwtAudience = "xiaoheiplay"
)

// newTokenID returns a random JWT id used to identify a single issued token.
func newTokenID() string {
	return randomToken(16)
}

// baseAuthClaims builds the shared claim set. iat is a NumericDate (integer
// Unix seconds, per RFC 7519) and each token carries iss/aud/jti so tokens are
// attributable and revocable-by-id. pvv is the user's password version at
// issuance; a later password change increments the stored version, which
// revokes this token (see tokenInvalidatedByPasswordChange).
func baseAuthClaims(userID int64, role, tokenType string, pvv int64, ttl time.Duration) jwt.MapClaims {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"type":    tokenType,
		"iss":     jwtIssuer,
		"aud":     jwtAudience,
		"jti":     newTokenID(),
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
	}
	if pvv > 0 {
		claims["pvv"] = pvv
	}
	return claims
}

func (h *Handler) signAuthToken(userID int64, role string, pvv int64, ttl time.Duration, tokenType string, familyID ...string) (string, error) {
	claims := baseAuthClaims(userID, role, tokenType, pvv, ttl)
	family := ""
	if len(familyID) > 0 {
		family = strings.TrimSpace(familyID[0])
	}
	if family == "" && tokenType == "refresh" {
		family = newTokenID()
	}
	if family != "" {
		claims["fid"] = family
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// signImpersonationToken signs an access token that acts as another (regular)
// user, tagging it with the impersonating admin's id in the `imp` claim so the
// middleware can surface the operator in RequestActor and audit logs can
// distinguish "the user themselves" from "an admin acting as the user" (round-4
// auth review Finding 7). The token still carries the target user's role/pvv so
// the existing per-request state checks apply unchanged.
func (h *Handler) signImpersonationToken(operatorID, operatorPVV, userID int64, role string, pvv int64, ttl time.Duration) (string, error) {
	claims := baseAuthClaims(userID, role, "access", pvv, ttl)
	// purpose=impersonation distinguishes a real login token from a
	// impersonation token so the frontend only honors impersonation for the
	// latter (web_auth_surface F1).
	claims["purpose"] = "impersonation"
	if operatorID > 0 {
		claims["imp"] = operatorID
		claims["imp_pvv"] = operatorPVV
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) signAuthTokenWithMFA(userID int64, role string, pvv int64, ttl time.Duration, tokenType string, mfa int, familyID ...string) (string, error) {
	claims := baseAuthClaims(userID, role, tokenType, pvv, ttl)
	claims["mfa"] = mfa
	family := ""
	if len(familyID) > 0 {
		family = strings.TrimSpace(familyID[0])
	}
	if family == "" && tokenType == "refresh" {
		family = newTokenID()
	}
	if family != "" {
		claims["fid"] = family
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// parseAuthToken verifies the signature, algorithm, issuer, audience, expiry
// and token type for access and refresh tokens, then checks the current user
// state (status/role/password version).
func (h *Handler) parseAuthToken(ctx context.Context, raw string, wantType string, emptyErr, invalidErr error) (jwt.MapClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, emptyErr
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, domain.ErrUnexpectedSigningMethod
			}
			return h.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
	)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, invalidErr
	}
	tokenType, _ := claims["type"].(string)
	if tokenType != wantType {
		return nil, domain.ErrInvalidTokenType
	}
	if err := validateCurrentAuthClaims(ctx, h.authSvc, claims); err != nil {
		return nil, invalidErr
	}
	return claims, nil
}

func (h *Handler) parseAccessToken(ctx context.Context, raw string) (jwt.MapClaims, error) {
	return h.parseAuthToken(ctx, raw, "access", domain.ErrEmptyToken, domain.ErrInvalidToken)
}

func (h *Handler) parseRefreshToken(ctx context.Context, raw string) (jwt.MapClaims, error) {
	return h.parseAuthToken(ctx, raw, "refresh", domain.ErrEmptyRefreshToken, domain.ErrInvalidRefreshToken)
}

func parseMapInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

// tokenInvalidatedByPasswordChange reports whether a token must be rejected
// because the user's password changed after the token was issued. The check is
// version-based: every issued token carries the password version it was
// created at, and every password change increments the stored version. This
// makes revocation exact — no dependence on timestamp precision or the
// same-second issuance boundary. Tokens issued before version tracking existed
// carry no version and are accepted only when the account has never changed its
// password since tracking began.
func tokenInvalidatedByPasswordChange(user domain.User, claims jwt.MapClaims) bool {
	if user.PasswordVersion <= 0 {
		return false
	}
	issued, ok := toInt64Claim(claims["pvv"])
	if !ok || issued != user.PasswordVersion {
		return true
	}
	return false
}

// tokenIssuedBeforePasswordChange is kept as an alias for call sites that
// reason about legacy tokens; it now delegates to the version-based check.
func tokenIssuedBeforePasswordChange(user domain.User, claims jwt.MapClaims) bool {
	return tokenInvalidatedByPasswordChange(user, claims)
}

func (h *Handler) Me(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.authSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserSelfDTO(user))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		QQ       string `json:"qq"`
		Phone    string `json:"phone"`
		Bio      string `json:"bio"`
		Intro    string `json:"intro"`
		Password string `json:"password"`
		TOTPCode string `json:"totp_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if strings.TrimSpace(payload.Password) != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPasswordChangeRequiresApiV1MePasswordChange.Error()})
		return
	}
	if strings.TrimSpace(payload.Email) != "" || strings.TrimSpace(payload.Phone) != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailPhoneUpdateRequiresApiV1MeSecurity.Error()})
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	currentUser, err := h.authSvc.GetUser(c.Request.Context(), getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	usernameChange := strings.TrimSpace(payload.Username) != "" && strings.TrimSpace(payload.Username) != strings.TrimSpace(currentUser.Username)
	if currentUser.TOTPEnabled && settings.TwoFAEnabled && usernameChange {
		if !h.allowTOTPVerify(c, currentUser.ID) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyAttempts.Error()})
			return
		}
		if err := h.authSvc.VerifyTOTP(c.Request.Context(), currentUser.ID, payload.TOTPCode); err != nil {
			h.recordAuthFailures(c.Request.Context(), h.totpBudgetDims(c, currentUser.ID), time.Now())
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
			return
		}
		h.resetAuthFailures(c.Request.Context(), h.totpBudgetDims(c, currentUser.ID)[0].key)
	}
	user, err := h.authSvc.UpdateProfile(c.Request.Context(), getUserID(c), appshared.UpdateProfileInput{
		Username: payload.Username,
		QQ:       payload.QQ,
		Bio:      payload.Bio,
		Intro:    payload.Intro,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrRealNameRequired) || errors.Is(err, appshared.ErrForbidden) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserSelfDTO(user))
}

func (h *Handler) MePasswordChange(c *gin.Context) {
	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		TOTPCode        string `json:"totp_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	currentUser, err := h.authSvc.GetUser(c.Request.Context(), getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if err := h.authSvc.VerifyPassword(c.Request.Context(), getUserID(c), payload.CurrentPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCurrentPasswordInvalid.Error()})
		return
	}
	if currentUser.TOTPEnabled && settings.TwoFAEnabled {
		if !h.allowTOTPVerify(c, currentUser.ID) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyAttempts.Error()})
			return
		}
		if err := h.authSvc.VerifyTOTP(c.Request.Context(), currentUser.ID, payload.TOTPCode); err != nil {
			h.recordAuthFailures(c.Request.Context(), h.totpBudgetDims(c, currentUser.ID), time.Now())
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
			return
		}
		h.resetAuthFailures(c.Request.Context(), h.totpBudgetDims(c, currentUser.ID)[0].key)
	}
	if err := validatePasswordBySettings(payload.NewPassword, settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.authSvc.UpdateProfile(c.Request.Context(), getUserID(c), appshared.UpdateProfileInput{Password: payload.NewPassword}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RealNameStatus(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabled, provider, actions := h.realNameSvc.GetConfig(c.Request.Context())
	var record *domain.RealNameVerification
	if latest, err := h.realNameSvc.Latest(c.Request.Context(), getUserID(c)); err == nil {
		record = &latest
	}
	verified := false
	if record != nil && record.Status == "verified" {
		verified = true
	}
	resp := gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
		"verified":      verified,
		"verification":  nil,
	}
	if record != nil {
		resp["verification"] = toRealNameVerificationDTO(*record, h.realNameRedirectURL(c.Request.Context(), *record))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RealNameVerify(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		RealName string `json:"real_name"`
		IDNumber string `json:"id_number"`
		Phone    string `json:"phone"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	phone := strings.TrimSpace(payload.Phone)
	if h.authSvc != nil {
		if user, err := h.authSvc.GetUser(c.Request.Context(), getUserID(c)); err == nil {
			if strings.TrimSpace(user.Phone) != "" {
				phone = strings.TrimSpace(user.Phone)
			}
		}
	}
	if h.realNameSvc != nil {
		_, providerKey, _ := h.realNameSvc.GetConfig(c.Request.Context())
		if realNameProviderNeedsMobile(providerKey) && phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
			return
		}
	}
	record, err := h.realNameSvc.VerifyWithInput(c.Request.Context(), getUserID(c), appshared.RealNameVerifyInput{
		RealName:    payload.RealName,
		IDNumber:    payload.IDNumber,
		Phone:       phone,
		CallbackURL: h.defaultRealNameCallbackURL(c),
	})
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, appshared.ErrForbidden):
			status = http.StatusForbidden
		case errors.Is(err, appshared.ErrTooManyAttempts):
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(record, h.realNameRedirectURL(c.Request.Context(), record)))
}

func realNameProviderNeedsMobile(providerKey string) bool {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	// mangzhu three-factor flow requires params.mobile.
	return strings.Contains(key, "mangzhu_realname")
}

func (h *Handler) defaultRealNameCallbackURL(c *gin.Context) string {
	siteURL := strings.TrimSpace(h.getSettingValueByKey(c, "site_url"))
	if siteURL != "" {
		if callback := buildRealNameCallbackURL(siteURL); callback != "" {
			return callback
		}
	}
	if c == nil || c.Request == nil {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		return ""
	}
	return buildRealNameCallbackURL(scheme + "://" + host)
}

func buildRealNameCallbackURL(base string) string {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if strings.TrimSpace(u.Host) == "" {
		return ""
	}
	path := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(path, "/console/realname") {
		path += "/console/realname"
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
