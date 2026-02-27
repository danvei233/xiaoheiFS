package http

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
	"xiaoheiplay/internal/domain"
)

var (
	adminPathRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	reservedPaths  = map[string]bool{
		"login":       true,
		"admin":       true,
		"api":         true,
		"install":     true,
		"console":     true,
		"register":    true,
		"assets":      true,
		"uploads":     true,
		"static":      true,
		"public":      true,
		"user":        true,
		"users":       true,
		"auth":        true,
		"logout":      true,
		"profile":     true,
		"settings":    true,
		"dashboard":   true,
		"home":        true,
		"index":       true,
		"help":        true,
		"docs":        true,
		"products":    true,
		"about":       true,
		"contact":     true,
		"support":     true,
		"forgot":      true,
		"reset":       true,
		"verify":      true,
		"callback":    true,
		"oauth":       true,
		"download":    true,
		"downloads":   true,
		"file":        true,
		"files":       true,
		"image":       true,
		"images":      true,
		"video":       true,
		"videos":      true,
		"media":       true,
		"css":         true,
		"js":          true,
		"javascript":  true,
		"favicon":     true,
		"robots":      true,
		"sitemap":     true,
		"manifest":    true,
		"service":     true,
		"worker":      true,
		"sw":          true,
		"health":      true,
		"ping":        true,
		"status":      true,
		"metrics":     true,
		"debug":       true,
		"test":        true,
		"demo":        true,
		"example":     true,
		"sample":      true,
		"tmp":         true,
		"temp":        true,
		"cache":       true,
		"backup":      true,
		"config":      true,
		"system":      true,
		"root":        true,
		"administrator": true,
		"webmaster":   true,
		"moderator":   true,
		"superuser":   true,
		"sysadmin":    true,
	}
)

// ValidateAdminPath 校验管理端路径
func ValidateAdminPath(path string) error {
	path = strings.TrimSpace(path)
	
	// 允许为空（使用默认路径 /admin）
	if path == "" {
		return nil
	}
	
	// 检查是否只包含字母和数字
	if !adminPathRegex.MatchString(path) {
		return domain.ErrAdminPathInvalid
	}
	
	// 检查是否在黑名单中
	if reservedPaths[strings.ToLower(path)] {
		return domain.ErrAdminPathReserved
	}
	
	return nil
}

// GenerateRandomAdminPath 生成随机管理端路径
func GenerateRandomAdminPath() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 12
	
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// 降级方案：使用base64编码
		fallback := make([]byte, 9)
		rand.Read(fallback)
		encoded := base64.RawURLEncoding.EncodeToString(fallback)
		// 只保留字母和数字
		result := ""
		for _, c := range encoded {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				result += string(c)
			}
		}
		if len(result) >= length {
			return result[:length]
		}
		return result + "admin"
	}
	
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[int(b[i])%len(charset)]
	}
	
	return string(result)
}

// GetAdminPathFromSettings 从设置中获取管理端路径
func GetAdminPathFromSettings(settingsSvc SettingsService) string {
	if settingsSvc == nil {
		return ""
	}
	
	settings, err := settingsSvc.List(nil)
	if err != nil {
		return ""
	}
	
	for _, s := range settings {
		if s.Key == "admin_path" {
			path := strings.TrimSpace(s.ValueJSON)
			// 移除引号（如果是JSON字符串）
			path = strings.Trim(path, `"`)
			if ValidateAdminPath(path) == nil {
				return path
			}
			return ""
		}
	}
	
	return ""
}

