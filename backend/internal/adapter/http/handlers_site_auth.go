package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"image/png"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
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
	settings := h.loadAuthSettings(c)
	if settings.CaptchaProvider == "geetest" {
		c.JSON(http.StatusOK, gin.H{
			"captcha_provider": "geetest",
			"captcha_id":       settings.GeeTestCaptchaID,
			"api_server":       settings.GeeTestAPIServer,
		})
		return
	}
	captcha, code, err := h.authSvc.CreateCaptchaWithPolicy(c, 5*time.Minute, settings.CaptchaLength, settings.CaptchaComplexity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrCaptchaError.Error()})
		return
	}
	img := renderCaptcha(code)
	var buf strings.Builder
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
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
		Username      string `json:"username"`
		Email         string `json:"email"`
		QQ            string `json:"qq"`
		Phone         string `json:"phone"`
		Password      string `json:"password"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	settings := h.loadAuthSettings(c)
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
			if err := h.authSvc.VerifyVerificationCode(c, "email", strings.TrimSpace(payload.Email), "register", code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidVerificationCode.Error()})
				return
			}
		case "sms":
			if strings.TrimSpace(payload.Phone) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
				return
			}
			if err := h.authSvc.VerifyVerificationCode(c, "sms", strings.TrimSpace(payload.Phone), "register", code); err != nil {
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
	user, err := h.authSvc.Register(c, appshared.RegisterInput{
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
		if err == appshared.ErrRealNameRequired || err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
}

func (h *Handler) Login(c *gin.Context) {
	var payload struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaCode   string `json:"captcha_code"`
		LotNumber     string `json:"lot_number"`
		CaptchaOutput string `json:"captcha_output"`
		PassToken     string `json:"pass_token"`
		GenTime       string `json:"gen_time"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	settings := h.loadAuthSettings(c)
	if settings.LoginRateLimitEnabled {
		key := "login:" + strings.ToLower(strings.TrimSpace(payload.Username)) + ":" + strings.TrimSpace(c.ClientIP())
		if !loginLimiter.Allow(key, settings.LoginRateLimitMax, settings.LoginRateLimitWindow) {
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
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidCredentials.Error()})
		return
	}
	h.postLoginSecurityHook(c, user, settings)
	accessToken, err := h.signAuthToken(user.ID, string(user.Role), 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthToken(user.ID, string(user.Role), 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
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
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AuthSettings(c *gin.Context) {
	settings := h.loadAuthSettings(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	settings := h.loadAuthSettings(c)
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
		if !registerCodeLimiter.Allow("register_code:email:"+strings.ToLower(emailVal), 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
			return
		}
		code, err := h.authSvc.CreateVerificationCodeWithPolicy(c, "email", emailVal, "register", settings.RegisterVerifyTTL, settings.EmailCodeLength, settings.EmailCodeComplexity)
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
		if err := h.emailSender.Send(c, emailVal, subject, body); err != nil {
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
		if !registerCodeLimiter.Allow("register_code:sms:"+phoneVal, 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
			return
		}
		code, err := h.authSvc.CreateVerificationCodeWithPolicy(c, "sms", phoneVal, "register", settings.RegisterVerifyTTL, settings.SMSCodeLength, settings.SMSCodeComplexity)
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
		cctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()
		if _, err := h.smsSender.Send(cctx, settings.RegisterSMSPluginID, settings.RegisterSMSInstanceID, msg); err != nil {
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
		_ = h.authSvc.UpdateLoginSecurity(c, user.ID, ip, user.LastLoginCity, user.LastLoginTZ, time.Now())
		return
	}
	city, tz := h.resolveGeoByIP(c, ip, settings.GeoIPMMDBPath)
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
	_ = h.authSvc.UpdateLoginSecurity(c, user.ID, ip, city, tz, loginTime)
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
				lastErr = fmt.Errorf("email template %s not configured", templateName)
				continue
			}
			if h.emailSender == nil {
				lastErr = domain.ErrEmailSenderNotConfigured
				continue
			}
			if err := h.emailSender.Send(c, emailAddr, subject, body); err != nil {
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
				lastErr = fmt.Errorf("sms template %s not configured", templateName)
				continue
			}
			msgVars := map[string]string{}
			for k, v := range vars {
				msgVars[k] = v
			}
			_, err := h.smsSender.Send(c, pluginID, instanceID, appshared.SMSMessage{
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	claims, err := h.parseRefreshToken(payload.RefreshToken)
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
	if role == "" {
		role = "user"
	}
	if h.authSvc != nil {
		user, err := h.authSvc.GetUser(c, userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
		if tokenIssuedBeforePasswordChange(user, claims) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
	}
	accessToken, err := h.signAuthToken(userID, role, 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	newRefreshToken, err := h.signAuthToken(userID, role, 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) signAuthToken(userID int64, role string, ttl time.Duration, tokenType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"type":    tokenType,
		"iat":     float64(time.Now().UnixNano()) / 1e9,
		"exp":     time.Now().Add(ttl).Unix(),
	})
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) signAuthTokenWithMFA(userID int64, role string, ttl time.Duration, tokenType string, mfa int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"type":    tokenType,
		"mfa":     mfa,
		"iat":     float64(time.Now().UnixNano()) / 1e9,
		"exp":     time.Now().Add(ttl).Unix(),
	})
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) parseAccessToken(ctx context.Context, raw string) (jwt.MapClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, domain.ErrEmptyToken
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnexpectedSigningMethod
		}
		return h.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, domain.ErrInvalidToken
	}
	tokenType, _ := claims["type"].(string)
	if tokenType != "" && tokenType != "access" {
		return nil, domain.ErrInvalidTokenType
	}
	if h.authSvc != nil {
		userID, ok := parseMapInt64(claims["user_id"])
		if ok && userID > 0 {
			user, userErr := h.authSvc.GetUser(ctx, userID)
			if userErr != nil {
				return nil, domain.ErrInvalidToken
			}
			if tokenIssuedBeforePasswordChange(user, claims) {
				return nil, domain.ErrInvalidToken
			}
		}
	}
	return claims, nil
}

func (h *Handler) parseRefreshToken(raw string) (jwt.MapClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, domain.ErrEmptyRefreshToken
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnexpectedSigningMethod
		}
		return h.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, domain.ErrInvalidRefreshToken
	}
	tokenType, _ := claims["type"].(string)
	// Backward compatible: allow legacy tokens without type.
	if tokenType != "" && tokenType != "refresh" {
		return nil, domain.ErrInvalidTokenType
	}
	return claims, nil
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

func tokenIssuedBeforePasswordChange(user domain.User, claims jwt.MapClaims) bool {
	if user.Role == domain.UserRoleAdmin {
		return false
	}
	if user.PasswordChangedAt == nil || user.PasswordChangedAt.IsZero() {
		return false
	}
	iatSeconds, ok := toFloat64Claim(claims["iat"])
	if !ok || iatSeconds <= 0 {
		// No iat -> treat as legacy token, invalidate after password change.
		return true
	}
	changedAt := float64(user.PasswordChangedAt.UnixNano()) / 1e9
	return iatSeconds <= changedAt
}

func (h *Handler) Me(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.authSvc.GetUser(c, userID)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
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
	settings := h.loadAuthSettings(c)
	currentUser, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	usernameChange := strings.TrimSpace(payload.Username) != "" && strings.TrimSpace(payload.Username) != strings.TrimSpace(currentUser.Username)
	if currentUser.TOTPEnabled && settings.TwoFAEnabled && usernameChange {
		if err := h.authSvc.VerifyTOTP(c, currentUser.ID, payload.TOTPCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
			return
		}
	}
	user, err := h.authSvc.UpdateProfile(c, getUserID(c), appshared.UpdateProfileInput{
		Username: payload.Username,
		QQ:       payload.QQ,
		Bio:      payload.Bio,
		Intro:    payload.Intro,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrRealNameRequired || err == appshared.ErrForbidden {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	settings := h.loadAuthSettings(c)
	currentUser, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if err := h.authSvc.VerifyPassword(c, getUserID(c), payload.CurrentPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCurrentPasswordInvalid.Error()})
		return
	}
	if currentUser.TOTPEnabled && settings.TwoFAEnabled {
		if err := h.authSvc.VerifyTOTP(c, currentUser.ID, payload.TOTPCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
			return
		}
	}
	if err := validatePasswordBySettings(payload.NewPassword, settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.authSvc.UpdateProfile(c, getUserID(c), appshared.UpdateProfileInput{Password: payload.NewPassword}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RealNameStatus(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabled, provider, actions := h.realnameSvc.GetConfig(c)
	var record *domain.RealNameVerification
	if latest, err := h.realnameSvc.Latest(c, getUserID(c)); err == nil {
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
		resp["verification"] = toRealNameVerificationDTO(*record)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RealNameVerify(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		RealName string `json:"real_name"`
		IDNumber string `json:"id_number"`
		Phone    string `json:"phone"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	phone := strings.TrimSpace(payload.Phone)
	if h.authSvc != nil {
		if user, err := h.authSvc.GetUser(c, getUserID(c)); err == nil {
			if strings.TrimSpace(user.Phone) != "" {
				phone = strings.TrimSpace(user.Phone)
			}
		}
	}
	record, err := h.realnameSvc.VerifyWithInput(c, getUserID(c), appshared.RealNameVerifyInput{
		RealName:    payload.RealName,
		IDNumber:    payload.IDNumber,
		Phone:       phone,
		CallbackURL: h.defaultRealNameCallbackURL(c),
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(record))
}

func (h *Handler) defaultRealNameCallbackURL(c *gin.Context) string {
	siteURL := strings.TrimSpace(h.getSettingValueByKey(c, "site_url"))
	if siteURL != "" {
		if strings.HasPrefix(siteURL, "http://") || strings.HasPrefix(siteURL, "https://") {
			return strings.TrimRight(siteURL, "/")
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
	return strings.TrimRight(scheme+"://"+host, "/")
}
