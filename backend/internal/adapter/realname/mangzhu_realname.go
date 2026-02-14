package realname

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"xiaoheiplay/internal/usecase"
)

type MangzhuRealNameProvider struct {
	settings usecase.SettingsRepository
}

func NewMangzhuRealNameProvider(settings usecase.SettingsRepository) *MangzhuRealNameProvider {
	return &MangzhuRealNameProvider{settings: settings}
}

func (p *MangzhuRealNameProvider) Key() string {
	return "mangzhu_realname"
}

func (p *MangzhuRealNameProvider) Name() string {
	return "Mangzhu Realname"
}

func (p *MangzhuRealNameProvider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	return p.VerifyWithInput(ctx, usecase.RealNameVerifyInput{
		RealName: realName,
		IDNumber: idNumber,
	})
}

func (p *MangzhuRealNameProvider) VerifyWithInput(ctx context.Context, in usecase.RealNameVerifyInput) (bool, string, error) {
	cfg := p.loadConfig(ctx)
	if strings.TrimSpace(cfg.Key) == "" {
		return false, "mangzhu key not configured", nil
	}
	name := strings.TrimSpace(in.RealName)
	id := strings.TrimSpace(in.IDNumber)
	if name == "" || id == "" {
		return false, "real_name/id_number required", nil
	}
	switch cfg.Mode {
	case "two_factor":
		raw, code, msg, err := p.request(ctx, cfg, "/index/sm_api", map[string]string{
			"key":    cfg.Key,
			"name":   name,
			"idcard": id,
		})
		if err != nil {
			return false, "", err
		}
		return parseTwoFactor(code, msg, raw), parseReason(msg, raw), nil
	case "three_factor":
		phone := strings.TrimSpace(in.Phone)
		if phone == "" {
			return false, "phone required for three_factor", nil
		}
		raw, code, msg, err := p.request(ctx, cfg, "/index/sm3_api", map[string]string{
			"key":    cfg.Key,
			"name":   name,
			"idcard": id,
			"mobile": phone,
		})
		if err != nil {
			return false, "", err
		}
		return code == 200, parseReason(msg, raw), nil
	default:
		startPath := "/index/bd_sm"
		faceProvider := cfg.FaceProvider
		if faceProvider == "wechat" {
			startPath = "/index/wx_sm"
		}
		raw, code, msg, err := p.request(ctx, cfg, startPath, map[string]string{
			"key":    cfg.Key,
			"name":   name,
			"idcard": id,
			"url":    "https://localhost/realname/callback",
		})
		if err != nil {
			return false, "", err
		}
		token, ok := parseFaceStartToken(code, raw)
		if !ok {
			return false, parseReason(msg, raw), nil
		}
		return false, "pending_face:" + faceProvider + ":" + token, nil
	}
}

func (p *MangzhuRealNameProvider) QueryPending(ctx context.Context, token string, provider string) (status string, reason string, err error) {
	cfg := p.loadConfig(ctx)
	if strings.TrimSpace(cfg.Key) == "" {
		return "failed", "mangzhu key not configured", nil
	}
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider != "wechat" && provider != "baidu" {
		provider = cfg.FaceProvider
	}
	queryPath := "/index/bd_cx"
	if provider == "wechat" {
		queryPath = "/index/wx_cx"
	}
	raw, code, msg, err := p.request(ctx, cfg, queryPath, map[string]string{
		"key":   cfg.Key,
		"token": strings.TrimSpace(token),
	})
	if err != nil {
		return "", "", err
	}
	return parseFaceQueryResult(provider, code, msg, raw), parseReason(msg, raw), nil
}

type mangzhuConfig struct {
	BaseURL      string
	Key          string
	Mode         string
	FaceProvider string
	TimeoutSec   int
}

func (p *MangzhuRealNameProvider) loadConfig(ctx context.Context) mangzhuConfig {
	cfg := mangzhuConfig{
		BaseURL:      "https://e.mangzhuyun.cn",
		Mode:         "three_factor",
		FaceProvider: "baidu",
		TimeoutSec:   10,
	}
	if p.settings == nil {
		return cfg
	}
	if s, err := p.settings.GetSetting(ctx, "realname_mangzhu_base_url"); err == nil && strings.TrimSpace(s.ValueJSON) != "" {
		cfg.BaseURL = strings.TrimSpace(s.ValueJSON)
	}
	if s, err := p.settings.GetSetting(ctx, "realname_mangzhu_key"); err == nil {
		cfg.Key = strings.TrimSpace(s.ValueJSON)
	}
	if s, err := p.settings.GetSetting(ctx, "realname_mangzhu_auth_mode"); err == nil && strings.TrimSpace(s.ValueJSON) != "" {
		cfg.Mode = strings.TrimSpace(strings.ToLower(s.ValueJSON))
	}
	if s, err := p.settings.GetSetting(ctx, "realname_mangzhu_face_provider"); err == nil && strings.TrimSpace(s.ValueJSON) != "" {
		cfg.FaceProvider = strings.TrimSpace(strings.ToLower(s.ValueJSON))
	}
	if s, err := p.settings.GetSetting(ctx, "realname_mangzhu_timeout_sec"); err == nil && strings.TrimSpace(s.ValueJSON) != "" {
		var v int
		if _, err := fmt.Sscanf(strings.TrimSpace(s.ValueJSON), "%d", &v); err == nil && v > 0 && v <= 60 {
			cfg.TimeoutSec = v
		}
	}
	return cfg
}

func (p *MangzhuRealNameProvider) request(ctx context.Context, cfg mangzhuConfig, endpoint string, params map[string]string) (raw string, code int, msg string, err error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://e.mangzhuyun.cn"
	}
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	form := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			form.Set(k, strings.TrimSpace(v))
		}
	}
	req, _ := http.NewRequestWithContext(cctx, http.MethodPost, base+endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	raw = strings.TrimSpace(string(b))
	var doc map[string]any
	if uerr := json.Unmarshal(b, &doc); uerr == nil {
		if c, ok := doc["code"].(float64); ok {
			code = int(c)
		}
		msg = strings.TrimSpace(anyToString(doc["msg"]))
	}
	return raw, code, msg, nil
}

func parseTwoFactor(code int, msg string, raw string) bool {
	if code != 200 {
		return false
	}
	var doc struct {
		Data struct {
			Result int `json:"result"`
		} `json:"data"`
		Result string `json:"result"`
	}
	_ = json.Unmarshal([]byte(raw), &doc)
	if doc.Data.Result == 1 {
		return true
	}
	return strings.Contains(doc.Result, "通过") || strings.Contains(strings.ToLower(msg), "success")
}

func parseReason(msg, raw string) string {
	var doc map[string]any
	_ = json.Unmarshal([]byte(raw), &doc)
	reason := strings.TrimSpace(anyToString(doc["msg"]))
	if reason != "" {
		return reason
	}
	if data, ok := doc["data"].(map[string]any); ok {
		r := strings.TrimSpace(anyToString(data["message"]))
		if r != "" {
			return r
		}
	}
	return strings.TrimSpace(msg)
}

func parseFaceStartToken(code int, raw string) (string, bool) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return "", false
	}
	if code != 200 {
		return "", false
	}
	token := strings.TrimSpace(anyToString(doc["token"]))
	return token, token != ""
}

func parseFaceQueryResult(provider string, code int, msg string, raw string) string {
	if code != 200 {
		return "failed"
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return "pending"
	}
	sm := -1
	switch v := doc["sm"].(type) {
	case float64:
		sm = int(v)
	case int:
		sm = v
	}
	if provider == "wechat" {
		switch sm {
		case 1:
			return "verified"
		case 3:
			return "failed"
		default:
			return "pending"
		}
	}
	switch sm {
	case 2:
		return "verified"
	case 1:
		return "failed"
	default:
		return "pending"
	}
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

var _ usecase.RealNameProvider = (*MangzhuRealNameProvider)(nil)
var _ usecase.RealNameProviderWithInput = (*MangzhuRealNameProvider)(nil)
var _ usecase.RealNameProviderPendingPoller = (*MangzhuRealNameProvider)(nil)
