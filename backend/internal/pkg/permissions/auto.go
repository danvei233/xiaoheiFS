package permissions

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/domain"
)

type moduleMeta struct {
	Display   string
	SortOrder int
}

var moduleMapping = map[string]moduleMeta{
	"user":             {Display: "用户管理", SortOrder: 1},
	"order":            {Display: "订单管理", SortOrder: 2},
	"vps":              {Display: "VPS管理", SortOrder: 3},
	"region":           {Display: "地区管理", SortOrder: 4},
	"plan_group":       {Display: "线路管理", SortOrder: 5},
	"line":             {Display: "线路管理", SortOrder: 5},
	"package":          {Display: "套餐管理", SortOrder: 6},
	"system_image":     {Display: "系统镜像", SortOrder: 7},
	"billing_cycle":    {Display: "计费周期", SortOrder: 8},
	"settings":         {Display: "系统设置", SortOrder: 9},
	"debug":            {Display: "Debug", SortOrder: 9},
	"automation":       {Display: "自动化平台", SortOrder: 10},
	"robot":            {Display: "机器人配置", SortOrder: 11},
	"smtp":             {Display: "SMTP配置", SortOrder: 12},
	"api_key":          {Display: "API密钥", SortOrder: 13},
	"email_template":   {Display: "邮件模板", SortOrder: 14},
	"admin":            {Display: "管理员管理", SortOrder: 15},
	"permission_group": {Display: "权限组", SortOrder: 16},
	"permission":       {Display: "权限配置", SortOrder: 17},
	"audit_log":        {Display: "审计日志", SortOrder: 18},
	"dashboard":        {Display: "数据面板", SortOrder: 19},
	"profile":          {Display: "个人中心", SortOrder: 20},
	"cms_category":     {Display: "内容分类", SortOrder: 21},
	"cms_post":         {Display: "内容管理", SortOrder: 22},
	"cms_block":        {Display: "页面模块", SortOrder: 23},
	"upload":           {Display: "资源上传", SortOrder: 24},
	"tickets":          {Display: "工单管理", SortOrder: 25},
	"goods_type":       {Display: "Goods Type", SortOrder: 6},
	"plugin":           {Display: "Plugin", SortOrder: 26},
	"probe":            {Display: "探针监控", SortOrder: 27},
}

var actionFriendlyName = map[string]string{
	"list":              "列表",
	"view":              "详情",
	"create":            "创建",
	"update":            "更新",
	"delete":            "删除",
	"bulk_delete":       "批量删除",
	"approve":           "审核通过",
	"reject":            "驳回",
	"mark_paid":         "标记已收款",
	"retry":             "重试",
	"lock":              "锁定",
	"unlock":            "解锁",
	"resize":            "变更配置",
	"status":            "修改状态",
	"update_status":     "更新状态",
	"update_expire":     "修改到期时间",
	"reset_password":    "重置密码",
	"emergency_renew":   "紧急续费",
	"refresh":           "刷新",
	"sync":              "同步",
	"test":              "测试",
	"set_system_images": "设置镜像",
	"change_password":   "修改密码",
	"overview":          "概览",
	"revenue":           "收入统计",
	"vps_status":        "VPS状态分布",
	"tree":              "权限树",
}

var actionSortOrder = map[string]int{
	"list":              1,
	"view":              2,
	"create":            3,
	"update":            4,
	"delete":            5,
	"bulk_delete":       6,
	"approve":           7,
	"reject":            8,
	"mark_paid":         9,
	"retry":             10,
	"lock":              11,
	"unlock":            12,
	"resize":            13,
	"status":            14,
	"update_status":     15,
	"update_expire":     16,
	"reset_password":    17,
	"emergency_renew":   18,
	"refresh":           19,
	"sync":              20,
	"test":              21,
	"set_system_images": 22,
	"change_password":   23,
	"overview":          24,
	"revenue":           25,
	"vps_status":        26,
	"tree":              27,
}

func BuildFromRoutes(routes []gin.RouteInfo) []domain.PermissionDefinition {
	permMap := make(map[string]domain.PermissionDefinition)
	moduleMap := make(map[string]domain.PermissionDefinition)

	for _, route := range routes {
		module, action, ok := inferPermission(route.Method, route.Path)
		if !ok {
			continue
		}
		meta := moduleMapping[module]
		moduleName := meta.Display
		if moduleName == "" {
			moduleName = module
		}
		if _, exists := moduleMap[module]; !exists {
			moduleMap[module] = domain.PermissionDefinition{
				Code:         module,
				Name:         moduleName,
				FriendlyName: moduleName,
				Category:     module,
				ParentCode:   "",
				SortOrder:    meta.SortOrder,
			}
		}

		actionName := actionFriendlyName[action]
		if actionName == "" {
			actionName = action
		}
		code := module + "." + action
		permMap[code] = domain.PermissionDefinition{
			Code:         code,
			Name:         actionName,
			FriendlyName: actionName,
			Category:     module,
			ParentCode:   module,
			SortOrder:    actionSortOrder[action],
		}
	}

	defs := make([]domain.PermissionDefinition, 0, len(moduleMap)+len(permMap))
	for _, def := range moduleMap {
		defs = append(defs, def)
	}
	for _, def := range permMap {
		defs = append(defs, def)
	}

	sort.Slice(defs, func(i, j int) bool {
		if defs[i].Category != defs[j].Category {
			return defs[i].Category < defs[j].Category
		}
		if defs[i].SortOrder != defs[j].SortOrder {
			return defs[i].SortOrder < defs[j].SortOrder
		}
		return defs[i].Code < defs[j].Code
	})

	return defs
}

func InferPermissionCode(method, path string) (string, bool) {
	module, action, ok := inferPermission(method, path)
	if !ok {
		return "", false
	}
	return module + "." + action, true
}

func inferPermission(method, path string) (string, string, bool) {
	if !strings.HasPrefix(path, "/admin/api/v1/") {
		return "", "", false
	}
	if strings.HasPrefix(path, "/admin/api/v1/auth") {
		return "", "", false
	}

	relative := strings.TrimPrefix(path, "/admin/api/v1/")
	if relative == "" {
		return "", "", false
	}
	segments := strings.Split(relative, "/")
	if len(segments) == 0 || segments[0] == "" {
		return "", "", false
	}

	module := moduleFromSegments(segments)
	if module == "" {
		return "", "", false
	}

	action, ok := actionFromSegments(method, segments)
	if !ok {
		return "", "", false
	}

	return module, action, true
}

func moduleFromSegments(segments []string) string {
	switch segments[0] {
	case "users":
		return "user"
	case "admins":
		return "admin"
	case "orders":
		return "order"
	case "cms":
		if len(segments) > 1 {
			switch segments[1] {
			case "categories":
				return "cms_category"
			case "posts":
				return "cms_post"
			case "blocks":
				return "cms_block"
			}
		}
		return "cms"
	case "plan-groups":
		return "plan_group"
	case "goods-types":
		return "goods_type"
	case "lines":
		return "line"
	case "system-images":
		return "system_image"
	case "billing-cycles":
		return "billing_cycle"
	case "api-keys":
		return "api_key"
	case "email-templates":
		return "email_template"
	case "permission-groups":
		return "permission_group"
	case "permissions":
		return "permission"
	case "audit-logs":
		return "audit_log"
	case "uploads":
		return "upload"
	case "payments":
		return "payment"
	case "plugins":
		return "plugin"
	case "server":
		return "server"
	case "wallet":
		if len(segments) > 1 && segments[1] == "orders" {
			return "wallet_order"
		}
		return "wallet"
	case "wallets":
		return "wallet"
	case "integrations":
		if len(segments) > 1 {
			return strings.ReplaceAll(segments[1], "-", "_")
		}
		return "integration"
	case "probes":
		return "probe"
	default:
		return strings.ReplaceAll(segments[0], "-", "_")
	}
}

func actionFromSegments(method string, segments []string) (string, bool) {
	if segments[0] == "dashboard" {
		if len(segments) > 1 {
			return strings.ReplaceAll(segments[1], "-", "_"), true
		}
		return "", false
	}
	if segments[0] == "profile" {
		switch method {
		case "GET":
			return "view", true
		case "PATCH":
			return "update", true
		case "POST":
			if len(segments) > 1 && segments[1] == "change-password" {
				return "change_password", true
			}
		}
		return "", false
	}
	if segments[0] == "settings" {
		switch method {
		case "GET":
			return "view", true
		case "PATCH":
			return "update", true
		}
		return "", false
	}
	if segments[0] == "debug" {
		if len(segments) > 1 && segments[1] == "status" {
			switch method {
			case "GET":
				return "view", true
			case "PATCH":
				return "update", true
			}
			return "", false
		}
		if len(segments) > 1 && segments[1] == "logs" && method == "GET" {
			return "list", true
		}
		return "", false
	}
	if segments[0] == "integrations" && len(segments) == 2 {
		switch method {
		case "GET":
			return "view", true
		case "PATCH":
			return "update", true
		}
		return "", false
	}
	if segments[0] == "payments" && len(segments) > 1 && segments[1] == "providers" {
		switch method {
		case "GET":
			return "list", true
		case "PATCH":
			return "update", true
		}
		return "", false
	}
	if segments[0] == "plugins" && len(segments) > 2 && segments[2] == "upload" && method == "POST" {
		return "upload", true
	}
	if segments[0] == "plugins" {
		if len(segments) == 1 && method == "GET" {
			return "list", true
		}
		if len(segments) == 2 && segments[1] == "discover" && method == "GET" {
			return "list", true
		}
		if len(segments) == 2 && segments[1] == "install" && method == "POST" {
			return "create", true
		}
		if len(segments) >= 3 && strings.HasPrefix(segments[1], ":") && strings.HasPrefix(segments[2], ":") {
			// /plugins/:category/:plugin_id/import
			if len(segments) == 4 && segments[3] == "import" && method == "POST" {
				return "create", true
			}
			// /plugins/:category/:plugin_id/instances
			if len(segments) == 4 && segments[3] == "instances" && method == "POST" {
				return "create", true
			}
			// /plugins/:category/:plugin_id/files
			if len(segments) == 4 && segments[3] == "files" && method == "DELETE" {
				return "delete", true
			}
			// legacy default-instance endpoints
			if len(segments) == 3 && method == "DELETE" {
				return "delete", true
			}
			if len(segments) == 4 && (segments[3] == "enable" || segments[3] == "disable") && method == "POST" {
				return "update", true
			}
			if len(segments) >= 4 && segments[3] == "config" {
				switch method {
				case "GET":
					return "view", true
				case "PUT", "PATCH":
					return "update", true
				}
			}

			// /plugins/:category/:plugin_id/:instance_id/...
			if len(segments) >= 4 && strings.HasPrefix(segments[3], ":") {
				if len(segments) == 4 && method == "DELETE" {
					return "delete", true
				}
				if len(segments) == 5 && (segments[4] == "enable" || segments[4] == "disable") && method == "POST" {
					return "update", true
				}
				if len(segments) >= 5 && segments[4] == "config" {
					switch method {
					case "GET":
						return "view", true
					case "PUT", "PATCH":
						return "update", true
					}
				}
			}
		}
	}
	if segments[0] == "server" && len(segments) > 1 && segments[1] == "status" && method == "GET" {
		return "status", true
	}
	if segments[0] == "integrations" && len(segments) > 2 {
		action := strings.ReplaceAll(segments[2], "-", "_")
		return action, true
	}
	if segments[0] == "realname" && len(segments) > 1 {
		switch segments[1] {
		case "config":
			switch method {
			case "GET":
				return "view", true
			case "PATCH":
				return "update", true
			}
		case "providers":
			if method == "GET" {
				return "list", true
			}
		case "records":
			if method == "GET" {
				return "list", true
			}
		}
	}
	if segments[0] == "wallet" && len(segments) > 1 && segments[1] == "orders" {
		if len(segments) == 2 && method == "GET" {
			return "list", true
		}
		if len(segments) > 2 && strings.HasPrefix(segments[2], ":") {
			if len(segments) > 3 {
				return strings.ReplaceAll(segments[3], "-", "_"), true
			}
		}
		return "", false
	}
	if segments[0] == "permissions" && len(segments) > 1 {
		if segments[1] == "list" {
			return "list", true
		}
		if segments[1] == "sync" {
			return "sync", true
		}
		switch method {
		case "GET":
			return "view", true
		case "PATCH":
			return "update", true
		}
	}
	if segments[0] == "permissions" && len(segments) == 1 && method == "GET" {
		return "tree", true
	}
	if segments[0] == "audit-logs" && method == "GET" {
		return "view", true
	}
	if len(segments) == 1 {
		switch method {
		case "GET":
			return "list", true
		case "POST":
			return "create", true
		}
		return "", false
	}
	if len(segments) == 2 {
		if segments[1] == "bulk-delete" && method == "POST" {
			return "bulk_delete", true
		}
		if segments[1] == "sync" && method == "POST" {
			return "sync", true
		}
		if segments[1] == "system-images" && method == "POST" {
			return "set_system_images", true
		}
	}
	if segments[0] == "users" && len(segments) > 2 {
		if segments[2] == "status" && method == "PATCH" {
			return "update", true
		}
		if segments[2] == "realname-status" && method == "PATCH" {
			return "update", true
		}
		if segments[2] == "impersonate" && method == "POST" {
			return "update", true
		}
	}
	if segments[0] == "vps" && len(segments) > 2 && segments[2] == "status" && method == "POST" {
		return "admin_status", true
	}
	if len(segments) > 1 && strings.HasPrefix(segments[1], ":") {
		if len(segments) == 2 {
			switch method {
			case "GET":
				return "view", true
			case "PATCH":
				return "update", true
			case "DELETE":
				return "delete", true
			}
			return "", false
		}
		action := strings.ReplaceAll(segments[2], "-", "_")
		switch action {
		case "mark_paid":
			return "mark_paid", true
		case "reset_password":
			return "reset_password", true
		case "emergency_renew":
			return "emergency_renew", true
		case "expire_at":
			return "update_expire", true
		case "system_images":
			return "set_system_images", true
		}
		return action, true
	}
	if len(segments) > 1 && segments[1] == "sync" && method == "POST" {
		return "sync", true
	}
	return "", false
}
