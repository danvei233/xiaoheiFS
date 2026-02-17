package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) PasswordResetOptions(c *gin.Context) {
	var payload struct {
		Account string `json:"account"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.PasswordResetEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "password reset disabled"})
		return
	}
	user, err := h.findUserByAccount(c, payload.Account)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	channels := make([]string, 0, 2)
	if hasChannel(settings.PasswordResetChannels, "email") && strings.TrimSpace(user.Email) != "" {
		channels = append(channels, "email")
	}
	if hasChannel(settings.PasswordResetChannels, "sms") && strings.TrimSpace(user.Phone) != "" {
		channels = append(channels, "sms")
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":      user.ID,
		"account":      user.Username,
		"channels":     channels,
		"masked_email": maskEmail(user.Email),
		"masked_phone": maskPhone(user.Phone),
		"has_email":    strings.TrimSpace(user.Email) != "",
		"has_phone":    strings.TrimSpace(user.Phone) != "",
		"sms_requires_phone_full": strings.TrimSpace(payload.Account) != "" &&
			strings.TrimSpace(payload.Account) != strings.TrimSpace(user.Phone),
	})
}

func (h *Handler) PasswordResetSendCode(c *gin.Context) {
	var payload struct {
		Account   string `json:"account"`
		Channel   string `json:"channel"`
		PhoneFull string `json:"phone_full"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.PasswordResetEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "password reset disabled"})
		return
	}
	user, err := h.findUserByAccount(c, payload.Account)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	channel := strings.ToLower(strings.TrimSpace(payload.Channel))
	if !hasChannel(settings.PasswordResetChannels, channel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel not enabled"})
		return
	}
	receiver := ""
	switch channel {
	case "email":
		receiver = strings.TrimSpace(user.Email)
		if receiver == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email not bound"})
			return
		}
	case "sms":
		receiver = strings.TrimSpace(user.Phone)
		if receiver == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone not bound"})
			return
		}
		account := strings.TrimSpace(payload.Account)
		phoneFull := strings.TrimSpace(payload.PhoneFull)
		if phoneFull == "" && account != receiver {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone_full required"})
			return
		}
		if phoneFull != "" && phoneFull != receiver {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone mismatch"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel"})
		return
	}
	receiverKey := receiver
	if channel == "email" {
		receiverKey = strings.ToLower(receiverKey)
	}
	if !resetCodeLimiter.Allow("password_reset_send:ip:"+strings.TrimSpace(c.ClientIP()), 10, 10*time.Minute) ||
		!resetCodeLimiter.Allow("password_reset_send:"+channel+":"+receiverKey, 3, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	length := settings.SMSCodeLength
	complexity := settings.SMSCodeComplexity
	if channel == "email" {
		length = settings.EmailCodeLength
		complexity = settings.EmailCodeComplexity
	}
	code, err := h.authSvc.CreateVerificationCodeWithPolicy(c, channel, receiver, "password_reset", settings.PasswordResetVerifyTTL, length, complexity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if channel == "email" {
		if err := h.sendSecurityMessage(c, []string{"email"}, "password_reset_verify_code", user, map[string]string{
			"code":  code,
			"email": user.Email,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := h.sendSecurityMessage(c, []string{"sms"}, "password_reset_verify_code", user, map[string]string{
			"code":  code,
			"phone": user.Phone,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) PasswordResetVerifyCode(c *gin.Context) {
	var payload struct {
		Account string `json:"account"`
		Channel string `json:"channel"`
		Code    string `json:"code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.PasswordResetEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "password reset disabled"})
		return
	}
	accountKey := strings.ToLower(strings.TrimSpace(payload.Account))
	channelKey := strings.ToLower(strings.TrimSpace(payload.Channel))
	if !resetVerifyLimiter.Allow("password_reset_verify:ip:"+strings.TrimSpace(c.ClientIP()), 20, 10*time.Minute) ||
		!resetVerifyLimiter.Allow("password_reset_verify:"+channelKey+":"+accountKey, 8, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	user, err := h.findUserByAccount(c, payload.Account)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	channel := strings.ToLower(strings.TrimSpace(payload.Channel))
	receiver := ""
	if channel == "email" {
		receiver = strings.TrimSpace(user.Email)
	} else if channel == "sms" {
		receiver = strings.TrimSpace(user.Phone)
	}
	if receiver == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "receiver not found"})
		return
	}
	if err := h.authSvc.VerifyVerificationCode(c, channel, receiver, "password_reset", payload.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification code"})
		return
	}
	svc := h.securityTicketSvc
	if svc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	_ = svc.DeleteExpired(c)
	token := randomToken(32)
	ticket := &domain.PasswordResetTicket{
		UserID:    user.ID,
		Channel:   channel,
		Receiver:  receiver,
		Token:     token,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := svc.Create(c, ticket.UserID, ticket.Channel, ticket.Receiver, ticket.Token, ticket.ExpiresAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reset_ticket": token, "expires_in": 900})
}

func (h *Handler) PasswordResetConfirm(c *gin.Context) {
	var payload struct {
		ResetTicket string `json:"reset_ticket"`
		NewPassword string `json:"new_password"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.PasswordResetEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "password reset disabled"})
		return
	}
	svc := h.securityTicketSvc
	if svc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	ticket, err := svc.Get(c, strings.TrimSpace(payload.ResetTicket))
	if err != nil || ticket.Used || time.Now().After(ticket.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reset ticket"})
		return
	}
	if err := validatePasswordBySettings(payload.NewPassword, h.loadAuthSettings(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.authSvc.UpdateProfile(c, ticket.UserID, appshared.UpdateProfileInput{Password: payload.NewPassword}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = svc.MarkUsed(c, ticket.ID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) MeSecurityContacts(c *gin.Context) {
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"email_bound":    strings.TrimSpace(user.Email) != "",
		"phone_bound":    strings.TrimSpace(user.Phone) != "",
		"email_masked":   maskEmail(user.Email),
		"phone_masked":   maskPhone(user.Phone),
		"totp_enabled":   user.TOTPEnabled,
		"security_level": map[string]any{"totp_enabled": user.TOTPEnabled},
	})
}

func (h *Handler) MeSecurityEmailSendCode(c *gin.Context) {
	h.meSecurityContactSendCode(c, "email")
}

func (h *Handler) MeSecurityPhoneSendCode(c *gin.Context) {
	h.meSecurityContactSendCode(c, "phone")
}

func (h *Handler) MeSecurityEmailVerify2FA(c *gin.Context) {
	h.meSecurityContactVerify2FA(c, "email")
}

func (h *Handler) MeSecurityPhoneVerify2FA(c *gin.Context) {
	h.meSecurityContactVerify2FA(c, "phone")
}

func (h *Handler) MeSecurityEmailConfirm(c *gin.Context) {
	h.meSecurityContactConfirm(c, "email")
}

func (h *Handler) MeSecurityPhoneConfirm(c *gin.Context) {
	h.meSecurityContactConfirm(c, "phone")
}

func (h *Handler) MeTwoFAStatus(c *gin.Context) {
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": user.TOTPEnabled})
}

func (h *Handler) MeTwoFASetup(c *gin.Context) {
	var payload struct {
		Password    string `json:"password"`
		CurrentCode string `json:"current_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.TwoFAEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "2fa disabled"})
		return
	}
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.TOTPEnabled && !settings.TwoFARebindEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "2fa rebind disabled"})
		return
	}
	if !user.TOTPEnabled && !settings.TwoFABindEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "2fa bind disabled"})
		return
	}
	secret, otpURL, err := h.authSvc.SetupTOTP(c, getUserID(c), payload.Password, payload.CurrentCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "otpauth_url": otpURL})
}

func (h *Handler) MeTwoFAConfirm(c *gin.Context) {
	var payload struct {
		Code string `json:"code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.authSvc.ConfirmTOTP(c, getUserID(c), payload.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func contactSecurityTicketChannel(kind string) string {
	if strings.ToLower(strings.TrimSpace(kind)) == "phone" {
		return "contact_bind_phone"
	}
	return "contact_bind_email"
}

func (h *Handler) verifyContactSecurityTicket(ctx *gin.Context, userID int64, kind string, rawToken string) (domain.PasswordResetTicket, error) {
	svc := h.securityTicketSvc
	if svc == nil {
		return domain.PasswordResetTicket{}, appshared.ErrNotSupported
	}
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return domain.PasswordResetTicket{}, domain.ErrSecurityTicketRequired
	}
	ticket, err := svc.Get(ctx, token)
	if err != nil {
		return domain.PasswordResetTicket{}, domain.ErrSecurityTicketInvalid
	}
	if ticket.Used || time.Now().After(ticket.ExpiresAt) {
		return domain.PasswordResetTicket{}, domain.ErrSecurityTicketInvalid
	}
	if ticket.UserID != userID {
		return domain.PasswordResetTicket{}, domain.ErrSecurityTicketInvalid
	}
	if strings.TrimSpace(ticket.Channel) != contactSecurityTicketChannel(kind) {
		return domain.PasswordResetTicket{}, domain.ErrSecurityTicketInvalid
	}
	return ticket, nil
}

func (h *Handler) findUserByAccount(ctx *gin.Context, account string) (domain.User, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if looksLikePhone(account) {
		if user, err := h.authSvc.GetUserByPhone(ctx, account); err == nil {
			return user, nil
		}
	}
	return h.authSvc.GetUserByUsernameOrEmail(ctx, account)
}

func looksLikePhone(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 6 {
		return false
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '+' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
}

func (h *Handler) meSecurityContactSendCode(c *gin.Context, kind string) {
	var payload struct {
		Value           string `json:"value"`
		CurrentPassword string `json:"current_password"`
		SecurityTicket  string `json:"security_ticket"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	isRebind := false
	if kind == "email" {
		if !settings.EmailBindEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "email bind disabled"})
			return
		}
		isRebind = strings.TrimSpace(user.Email) != ""
	} else {
		if !settings.PhoneBindEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "phone bind disabled"})
			return
		}
		isRebind = strings.TrimSpace(user.Phone) != ""
	}
	require2FA := user.TOTPEnabled && settings.TwoFAEnabled
	value := strings.TrimSpace(payload.Value)
	if value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value required"})
		return
	}
	if require2FA {
		svc := h.securityTicketSvc
		if svc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
			return
		}
		if _, err := h.verifyContactSecurityTicket(c, user.ID, kind, payload.SecurityTicket); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		if isRebind && settings.RebindRequirePasswordWhenNo2FA {
			if err := h.authSvc.VerifyPassword(c, user.ID, payload.CurrentPassword); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
				return
			}
		}
		if !isRebind && settings.BindRequirePasswordWhenNo2FA {
			if err := h.authSvc.VerifyPassword(c, user.ID, payload.CurrentPassword); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
				return
			}
		}
	}
	valueKey := value
	if kind == "email" {
		valueKey = strings.ToLower(valueKey)
	}
	if !contactCodeLimiter.Allow(fmt.Sprintf("contact_bind_send:user:%d:%s", user.ID, kind), 3, 10*time.Minute) ||
		!contactCodeLimiter.Allow("contact_bind_send:"+kind+":"+valueKey, 5, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	channel := "email"
	purpose := "bind_email"
	if kind == "phone" {
		channel = "sms"
		purpose = "bind_phone"
	}
	length := settings.SMSCodeLength
	complexity := settings.SMSCodeComplexity
	if channel == "email" {
		length = settings.EmailCodeLength
		complexity = settings.EmailCodeComplexity
	}
	code, err := h.authSvc.CreateVerificationCodeWithPolicy(c, channel, value, purpose, settings.ContactBindVerifyTTL, length, complexity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	shadow := user
	if kind == "email" {
		shadow.Email = value
	} else {
		shadow.Phone = value
	}
	templateName := "email_bind_verify_code"
	if kind == "phone" {
		templateName = "phone_bind_verify_code"
	}
	if err := h.sendSecurityMessage(c, []string{channel}, templateName, shadow, map[string]string{
		"code":  code,
		"email": value,
		"phone": value,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) meSecurityContactConfirm(c *gin.Context, kind string) {
	var payload struct {
		Value          string `json:"value"`
		Code           string `json:"code"`
		SecurityTicket string `json:"security_ticket"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	value := strings.TrimSpace(payload.Value)
	if value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value required"})
		return
	}
	settings := h.loadAuthSettings(c)
	require2FA := user.TOTPEnabled && settings.TwoFAEnabled
	var ticketSvc = h.securityTicketSvc
	var ticket domain.PasswordResetTicket
	if require2FA {
		svc := h.securityTicketSvc
		if svc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
			return
		}
		t, err := h.verifyContactSecurityTicket(c, user.ID, kind, payload.SecurityTicket)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ticket = t
	}
	if !contactVerifyLimiter.Allow(fmt.Sprintf("contact_bind_verify:user:%d:%s", user.ID, kind), 10, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	channel := "email"
	purpose := "bind_email"
	if kind == "phone" {
		channel = "sms"
		purpose = "bind_phone"
	}
	if err := h.authSvc.VerifyVerificationCode(c, channel, value, purpose, payload.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification code"})
		return
	}
	if kind == "email" {
		if exists, err := h.authSvc.GetUserByUsernameOrEmail(c, value); err == nil && exists.ID != user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
			return
		}
		user.Email = value
	} else {
		if exists, err := h.authSvc.GetUserByPhone(c, value); err == nil && exists.ID != user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone already exists"})
			return
		}
		user.Phone = value
	}
	if err := h.authSvc.UpdateUser(c, user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if require2FA && ticketSvc != nil {
		_ = ticketSvc.MarkUsed(c, ticket.ID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) meSecurityContactVerify2FA(c *gin.Context, kind string) {
	var payload struct {
		TOTPCode string `json:"totp_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	user, err := h.authSvc.GetUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !(user.TOTPEnabled && settings.TwoFAEnabled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2fa not enabled"})
		return
	}
	if err := h.authSvc.VerifyTOTP(c, user.ID, payload.TOTPCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 2fa code"})
		return
	}
	svc := h.securityTicketSvc
	if svc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	_ = svc.DeleteExpired(c)
	token := randomToken(9)
	ticket := &domain.PasswordResetTicket{
		UserID:    user.ID,
		Channel:   contactSecurityTicketChannel(kind),
		Receiver:  "-",
		Token:     token,
		ExpiresAt: time.Now().Add(20 * time.Minute),
	}
	if err := svc.Create(c, ticket.UserID, ticket.Channel, ticket.Receiver, ticket.Token, ticket.ExpiresAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"security_ticket": token, "expires_in": 1200})
}
