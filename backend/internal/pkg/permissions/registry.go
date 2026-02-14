package permissions

import (
	"xiaoheiplay/internal/domain"
)

var (
	definitions []domain.PermissionDefinition
)

func Register(code, name, category string, sortOrder int) {
	RegisterWithParentFriendlyName(code, name, "", category, "", sortOrder)
}

func RegisterWithFriendlyName(code, name, friendlyName, category string, sortOrder int) {
	RegisterWithParentFriendlyName(code, name, friendlyName, category, "", sortOrder)
}

func RegisterWithParent(code, name, category, parentCode string, sortOrder int) {
	RegisterWithParentFriendlyName(code, name, "", category, parentCode, sortOrder)
}

func RegisterWithParentFriendlyName(code, name, friendlyName, category, parentCode string, sortOrder int) {
	if friendlyName == "" {
		friendlyName = name
	}
	definitions = append(definitions, domain.PermissionDefinition{
		Code:         code,
		Name:         name,
		FriendlyName: friendlyName,
		Category:     category,
		ParentCode:   parentCode,
		SortOrder:    sortOrder,
	})
}

func GetDefinitions() []domain.PermissionDefinition {
	return definitions
}

func SetDefinitions(defs []domain.PermissionDefinition) {
	definitions = defs
}

func InitRegistry() {
	definitions = []domain.PermissionDefinition{}

	Register("user.view", "查看用户详情", "用户管理", 1)
	Register("user.list", "查看用户列表", "用户管理", 2)
	Register("user.create", "创建用户", "用户管理", 3)
	Register("user.update", "更新用户", "用户管理", 4)
	Register("user.delete", "删除用户", "用户管理", 5)
	Register("user.reset_password", "重置用户密码", "用户管理", 6)

	Register("order.view", "查看订单详情", "订单管理", 1)
	Register("order.list", "查看订单列表", "订单管理", 2)
	Register("order.approve", "批准订单", "订单管理", 3)
	Register("order.reject", "驳回订单", "订单管理", 4)
	Register("order.delete", "删除订单", "订单管理", 5)

	Register("vps.view", "查看VPS详情", "VPS管理", 1)
	Register("vps.list", "查看VPS列表", "VPS管理", 2)
	Register("vps.create", "创建VPS", "VPS管理", 3)
	Register("vps.update", "更新VPS", "VPS管理", 4)
	Register("vps.delete", "删除VPS", "VPS管理", 5)
	Register("vps.resize", "调整VPS配置", "VPS管理", 6)
	Register("vps.renew", "续费VPS", "VPS管理", 7)
	Register("vps.admin_status", "设置VPS管理员状态", "VPS管理", 8)

	Register("settings.view", "查看系统设置", "系统设置", 1)
	Register("settings.update", "更新系统设置", "系统设置", 2)
	Register("plugin.upload", "上传支付插件", "系统", 3)
	Register("server.status", "查看服务器状态", "系统", 4)

	Register("admin.view", "查看管理员详情", "管理员管理", 1)
	Register("admin.list", "查看管理员列表", "管理员管理", 2)
	Register("admin.create", "创建管理员", "管理员管理", 3)
	Register("admin.update", "更新管理员", "管理员管理", 4)
	Register("admin.delete", "删除管理员", "管理员管理", 5)

	Register("audit_log.view", "查看审计日志", "审计日志", 1)

	Register("api_key.view", "查看API密钥详情", "API密钥", 1)
	Register("api_key.list", "查看API密钥列表", "API密钥", 2)
	Register("api_key.create", "创建API密钥", "API密钥", 3)
	Register("api_key.update", "更新API密钥", "API密钥", 4)
	Register("api_key.delete", "删除API密钥", "API密钥", 5)

	Register("email_template.view", "查看邮件模板详情", "邮件模板", 1)
	Register("email_template.list", "查看邮件模板列表", "邮件模板", 2)
	Register("email_template.update", "更新邮件模板", "邮件模板", 3)
	Register("email_template.delete", "删除邮件模板", "邮件模板", 4)
	Register("sms.view", "查看短信配置", "短信配置", 1)
	Register("sms.update", "更新短信配置", "短信配置", 2)
	Register("sms.test", "测试短信发送", "短信配置", 3)
	Register("sms_template.view", "查看短信模板详情", "短信模板", 1)
	Register("sms_template.list", "查看短信模板列表", "短信模板", 2)
	Register("sms_template.update", "更新短信模板", "短信模板", 3)
	Register("sms_template.delete", "删除短信模板", "短信模板", 4)

	Register("product.view", "查看产品详情", "产品管理", 1)
	Register("product.list", "查看产品列表", "产品管理", 2)
	Register("product.create", "创建产品", "产品管理", 3)
	Register("product.update", "更新产品", "产品管理", 4)
	Register("product.delete", "删除产品", "产品管理", 5)

	Register("region.view", "查看区域详情", "区域管理", 1)
	Register("region.list", "查看区域列表", "区域管理", 2)
	Register("region.create", "创建区域", "区域管理", 3)
	Register("region.update", "更新区域", "区域管理", 4)
	Register("region.delete", "删除区域", "区域管理", 5)

	Register("billing_cycle.view", "查看计费周期详情", "计费周期", 1)
	Register("billing_cycle.list", "查看计费周期列表", "计费周期", 2)
	Register("billing_cycle.create", "创建计费周期", "计费周期", 3)
	Register("billing_cycle.update", "更新计费周期", "计费周期", 4)
	Register("billing_cycle.delete", "删除计费周期", "计费周期", 5)

	Register("system_image.view", "查看系统镜像详情", "系统镜像", 1)
	Register("system_image.list", "查看系统镜像列表", "系统镜像", 2)
	Register("system_image.create", "创建系统镜像", "系统镜像", 3)
	Register("system_image.update", "更新系统镜像", "系统镜像", 4)
	Register("system_image.delete", "删除系统镜像", "系统镜像", 5)

	Register("permission_group.view", "查看权限组详情", "权限组", 1)
	Register("permission_group.list", "查看权限组列表", "权限组", 2)
	Register("permission_group.create", "创建权限组", "权限组", 3)
	Register("permission_group.update", "更新权限组", "权限组", 4)
	Register("permission_group.delete", "删除权限组", "权限组", 5)
}

func GetCategories() []string {
	cats := make(map[string]bool)
	for _, d := range definitions {
		cats[d.Category] = true
	}
	result := make([]string, 0, len(cats))
	for cat := range cats {
		result = append(result, cat)
	}
	return result
}

func GetByCategory(category string) []domain.PermissionDefinition {
	var result []domain.PermissionDefinition
	for _, d := range definitions {
		if d.Category == category {
			result = append(result, d)
		}
	}
	return result
}
