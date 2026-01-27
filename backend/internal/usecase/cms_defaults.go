package usecase

import (
	"context"
	"encoding/json"
	"strings"

	"xiaoheiplay/internal/domain"
)

type defaultCMSBlock struct {
	Page      string
	Type      string
	Title     string
	Subtitle  string
	Lang      string
	SortOrder int
	Visible   bool
	Content   map[string]any
}

func defaultCMSBlocks(page, lang string) []defaultCMSBlock {
	if lang == "" {
		lang = "zh-CN"
	}
	switch strings.ToLower(page) {
	case "home":
		return []defaultCMSBlock{
			{Page: "home", Type: "hero", Title: "Home Hero", Lang: lang, SortOrder: 1, Visible: true, Content: map[string]any{}},
			{Page: "home", Type: "features", Title: "Home Features", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{Page: "home", Type: "products", Title: "Home Products", Lang: lang, SortOrder: 3, Visible: true, Content: map[string]any{}},
			{Page: "home", Type: "cta", Title: "Home CTA", Lang: lang, SortOrder: 4, Visible: true, Content: map[string]any{}},
		}
	case "products":
		return []defaultCMSBlock{
			{Page: "products", Type: "hero", Title: "Products Hero", Lang: lang, SortOrder: 1, Visible: true, Content: map[string]any{}},
			{Page: "products", Type: "calculator", Title: "Products Calculator", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{Page: "products", Type: "pricing", Title: "Products Pricing", Lang: lang, SortOrder: 3, Visible: true, Content: map[string]any{}},
			{Page: "products", Type: "comparison", Title: "Products Comparison", Lang: lang, SortOrder: 4, Visible: true, Content: map[string]any{}},
			{Page: "products", Type: "cta", Title: "Products CTA", Lang: lang, SortOrder: 5, Visible: true, Content: map[string]any{}},
		}
	case "docs":
		return []defaultCMSBlock{
			{
				Page:      "docs",
				Type:      "hero",
				Title:     "Docs Hero",
				Lang:      lang,
				SortOrder: 1,
				Visible:   true,
				Content: map[string]any{
					"title":    "文档中心",
					"subtitle": "官方文档与最佳实践",
				},
			},
			{Page: "docs", Type: "posts", Title: "Docs Posts", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{
				Page:      "docs",
				Type:      "resources",
				Title:     "Docs Resources",
				Lang:      lang,
				SortOrder: 3,
				Visible:   true,
				Content: map[string]any{
					"title": "相关资源",
					"items": []map[string]any{
						{"icon_key": "book", "title": "API 文档", "description": "完整的 API 参考手册和示例代码", "url": "#"},
						{"icon_key": "video", "title": "视频教程", "description": "手把手教您使用各项功能", "url": "#"},
						{"icon_key": "code", "title": "代码示例", "description": "常用场景的代码片段和最佳实践", "url": "#"},
						{"icon_key": "chat", "title": "社区支持", "description": "加入讨论，获取帮助与经验分享", "url": "#"},
					},
				},
			},
		}
	case "announcements":
		return []defaultCMSBlock{
			{
				Page:      "announcements",
				Type:      "hero",
				Title:     "Announcements Hero",
				Lang:      lang,
				SortOrder: 1,
				Visible:   true,
				Content: map[string]any{
					"title":    "最新公告",
					"subtitle": "产品动态与重要通知",
				},
			},
			{Page: "announcements", Type: "posts", Title: "Announcements Posts", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{Page: "announcements", Type: "resources", Title: "Announcements Resources", Lang: lang, SortOrder: 3, Visible: true, Content: map[string]any{}},
		}
	case "activities":
		return []defaultCMSBlock{
			{
				Page:      "activities",
				Type:      "hero",
				Title:     "Activities Hero",
				Lang:      lang,
				SortOrder: 1,
				Visible:   true,
				Content: map[string]any{
					"title":    "活动中心",
					"subtitle": "限时活动与优惠计划",
				},
			},
			{Page: "activities", Type: "posts", Title: "Activities Posts", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{Page: "activities", Type: "resources", Title: "Activities Resources", Lang: lang, SortOrder: 3, Visible: true, Content: map[string]any{}},
		}
	case "tutorials":
		return []defaultCMSBlock{
			{
				Page:      "tutorials",
				Type:      "hero",
				Title:     "Tutorials Hero",
				Lang:      lang,
				SortOrder: 1,
				Visible:   true,
				Content: map[string]any{
					"title":    "教程学院",
					"subtitle": "从入门到进阶的学习路径",
				},
			},
			{Page: "tutorials", Type: "posts", Title: "Tutorials Posts", Lang: lang, SortOrder: 2, Visible: true, Content: map[string]any{}},
			{Page: "tutorials", Type: "resources", Title: "Tutorials Resources", Lang: lang, SortOrder: 3, Visible: true, Content: map[string]any{}},
		}
	case "footer":
		return []defaultCMSBlock{
			{Page: "footer", Type: "footer", Title: "Footer", Lang: lang, SortOrder: 1, Visible: true, Content: map[string]any{}},
		}
	case "help":
		return []defaultCMSBlock{
			{
				Page:      "help",
				Type:      "help_hero",
				Title:     "Help Hero",
				Lang:      lang,
				SortOrder: 1,
				Visible:   true,
				Content: map[string]any{
					"badge":              "帮助中心",
					"title_main":         "我们能为您",
					"title_gradient":     "做些什么？",
					"subtitle":           "快速找到您需要的答案，或联系我们的专业团队获取支持",
					"search_placeholder": "搜索问题、关键词...",
					"quick_stats": []map[string]any{
						{"value": "100+", "label": "常见问题"},
						{"value": "24/7", "label": "在线支持"},
						{"value": "<5m", "label": "平均响应"},
						{"value": "99.9%", "label": "满意度"},
					},
				},
			},
			{
				Page:      "help",
				Type:      "help_actions",
				Title:     "Help Actions",
				Lang:      lang,
				SortOrder: 2,
				Visible:   true,
				Content: map[string]any{
					"cards": []map[string]any{
						{"key": "docs", "title": "文档中心", "description": "详细的产品文档和使用指南", "url": "/docs"},
						{"key": "tickets", "title": "提交工单", "description": "获取一对一的技术支持", "url": "/console/tickets", "guest_url": "/auth/login"},
						{"key": "announcements", "title": "最新公告", "description": "系统更新与重要通知", "url": "/announcements"},
						{"key": "contact", "title": "邮件支持", "description": "support@example.com", "url": "mailto:support@example.com"},
					},
				},
			},
			{
				Page:      "help",
				Type:      "help_faq",
				Title:     "Help FAQ",
				Lang:      lang,
				SortOrder: 3,
				Visible:   true,
				Content: map[string]any{
					"title":    "常见问题",
					"subtitle": "快速找到您关心的问题答案",
					"categories": []map[string]any{
						{"key": "all", "label": "全部"},
						{"key": "account", "label": "账号相关"},
						{"key": "payment", "label": "支付问题"},
						{"key": "vps", "label": "VPS使用"},
						{"key": "billing", "label": "账单退款"},
					},
					"faqs": []map[string]any{
						{"category": "account", "question": "如何注册账号？", "answer": "点击页面右上角的\"注册\"按钮，填写用户名、邮箱和密码即可完成注册。注册后需要验证邮箱才能使用全部功能。"},
						{"category": "account", "question": "忘记密码怎么办？", "answer": "点击登录页面的\"忘记密码\"链接，输入您的注册邮箱，我们会发送密码重置链接到您的邮箱。"},
						{"category": "account", "question": "如何修改个人资料？", "answer": "登录后进入控制台，点击右上角的用户头像，选择\"个人资料\"，即可修改您的基本信息、联系方式等。"},
						{"category": "account", "question": "如何开启二次验证？", "answer": "在控制台的\"安全设置\"中，可以开启两步验证功能，支持验证器应用（如Google Authenticator）提高账户安全性。"},
						{"category": "payment", "question": "支持哪些支付方式？", "answer": "我们支持支付宝、微信支付、银行卡等多种支付方式。企业用户还可以申请对公转账和发票服务。"},
						{"category": "payment", "question": "支付失败怎么办？", "answer": "如果支付失败，请先检查账户余额是否充足。如问题仍未解决，请联系客服并提供订单号，我们会协助您处理。"},
						{"category": "payment", "question": "可以申请发票吗？", "answer": "可以。企业用户可以在控制台的\"发票管理\"中申请开具增值税专用发票或普通发票。个人用户可申请电子发票。"},
						{"category": "payment", "question": "充值有优惠吗？", "answer": "我们不定期会推出充值优惠活动，请关注我们的公告页面或订阅邮件通知获取最新优惠信息。"},
						{"category": "vps", "question": "VPS多久可以开通？", "answer": "订单支付成功后，系统会自动开通VPS，通常在1-5分钟内完成。开通成功后您会收到邮件通知。"},
						{"category": "vps", "question": "如何远程连接VPS？", "answer": "Windows系统使用远程桌面连接，Linux系统使用SSH工具。控制台会显示您的IP地址和初始密码，请在首次登录后及时修改密码。"},
						{"category": "vps", "question": "VPS可以升级配置吗？", "answer": "可以。在控制台的VPS管理页面，选择\"升级配置\"，选择更高配置的套餐并支付差价即可。升级过程不会影响您的数据。"},
						{"category": "vps", "question": "如何重装系统？", "answer": "在控制台选择您要重装的VPS，点击\"重装系统\"，选择所需的系统镜像并确认。重装会清空系统盘数据，请提前备份。"},
						{"category": "vps", "question": "VPS可以做什么？", "answer": "您可以使用VPS搭建网站、运行应用程序、部署游戏服务器、搭建开发测试环境等。但请遵守我们的服务条款，禁止用于非法用途。"},
						{"category": "vps", "question": "带宽是如何计算的？", "answer": "我们提供的带宽是指峰值带宽，您可以随时使用达到该峰值。流量方面，不同套餐有不同配额，超出后可购买额外流量包。"},
						{"category": "billing", "question": "如何查看我的账单？", "answer": "登录控制台后，进入\"账单管理\"可以查看所有历史订单、消费记录和账单详情。支持按时间范围筛选和导出账单。"},
						{"category": "billing", "question": "支持自动续费吗？", "answer": "支持。您可以在VPS管理页面开启\"自动续费\"功能，系统会在到期前自动从余额扣款续费。请确保账户余额充足。"},
						{"category": "billing", "question": "退款政策是什么？", "answer": "我们提供7天无理由退款服务。新用户在首次购买后的7天内，如对服务不满意，可以申请全额退款（已使用流量按标准扣除费用）。"},
						{"category": "billing", "question": "VPS到期会怎样？", "answer": "VPS到期后会被停用，数据保留15天。期间您可以续费恢复服务。超过15天未续费，服务器将被回收，数据将无法恢复。"},
					},
				},
			},
			{
				Page:      "help",
				Type:      "help_contact",
				Title:     "Help Contact",
				Lang:      lang,
				SortOrder: 4,
				Visible:   true,
				Content: map[string]any{
					"title":       "还有问题？",
					"description": "我们的专业支持团队随时准备为您提供帮助",
					"channels": []map[string]any{
						{"key": "chat", "title": "在线客服", "subtitle": "工作日 9:00 - 18:00"},
						{"key": "mail", "title": "邮件支持", "subtitle": "24小时内回复"},
						{"key": "tickets", "title": "工单系统", "subtitle": "技术问题优先处理"},
					},
					"cta_title":       "立即开始使用",
					"cta_desc":        "注册账号，享受专业的云服务",
					"cta_button_text": "免费注册",
					"cta_url":         "/auth/register",
				},
			},
		}
	default:
		return nil
	}
}

func (s *CMSService) ensureDefaultBlocks(ctx context.Context, page, lang string, existing []domain.CMSBlock) ([]domain.CMSBlock, error) {
	defaults := defaultCMSBlocks(page, lang)
	if len(defaults) == 0 {
		return existing, nil
	}
	// Index defaults by type for quick lookup.
	defaultByType := make(map[string]defaultCMSBlock, len(defaults))
	for _, def := range defaults {
		defaultByType[def.Type] = def
	}

	isEffectivelyEmptyJSON := func(raw string) bool {
		if strings.TrimSpace(raw) == "" {
			return true
		}
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			// If it's not valid JSON, treat as empty so we can repair it.
			return true
		}
		switch t := v.(type) {
		case nil:
			return true
		case map[string]any:
			return len(t) == 0
		case []any:
			return len(t) == 0
		default:
			return false
		}
	}

	changed := false
	existingTypes := make(map[string]bool, len(existing))
	for _, item := range existing {
		existingTypes[item.Type] = true

		// If the block exists but its content is empty/invalid, backfill from defaults.
		if def, ok := defaultByType[item.Type]; ok && len(def.Content) > 0 && isEffectivelyEmptyJSON(item.ContentJSON) {
			contentJSON, _ := json.Marshal(def.Content)
			item.ContentJSON = string(contentJSON)
			if item.Title == "" {
				item.Title = def.Title
			}
			if item.Subtitle == "" {
				item.Subtitle = def.Subtitle
			}
			if item.SortOrder == 0 {
				item.SortOrder = def.SortOrder
			}
			// Respect user toggles: don't override Visible if it was explicitly set false.
			if item.Visible {
				item.Visible = def.Visible
			}
			if err := s.blocks.UpdateCMSBlock(ctx, item); err != nil {
				return existing, err
			}
			changed = true
		}
	}

	// Create any missing default blocks.
	for _, def := range defaults {
		if existingTypes[def.Type] {
			continue
		}
		contentJSON, _ := json.Marshal(def.Content)
		block := domain.CMSBlock{
			Page:        def.Page,
			Type:        def.Type,
			Title:       def.Title,
			Subtitle:    def.Subtitle,
			ContentJSON: string(contentJSON),
			CustomHTML:  "",
			Lang:        def.Lang,
			Visible:     def.Visible,
			SortOrder:   def.SortOrder,
		}
		if err := s.blocks.CreateCMSBlock(ctx, &block); err != nil {
			return existing, err
		}
		existing = append(existing, block)
		changed = true
	}
	if !changed {
		return existing, nil
	}
	return s.blocks.ListCMSBlocks(ctx, page, lang, true)
}
