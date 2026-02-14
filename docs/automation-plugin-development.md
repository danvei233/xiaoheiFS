# 自动化系统对接插件开发文档

> 适用仓库：`xiaohei`  
> 适用读者：后端插件开发者、交付实施、运维联调人员  
> 当前阶段：`Alpha`（接口稳定性以当前仓库代码为准）

## 1. 文档目标与适用范围

本文档用于指导你从 0 开发并接入一个 `automation` 插件，目标是实现：

1. 能被后端插件系统加载并通过握手。
2. 能在后台完成安装、实例化、启用、配置、同步。
3. 能支持商品类型绑定后，驱动实例生命周期操作（开关机、重装、续费等）。
4. 能支持可选能力（端口映射/备份/快照/防火墙）且未实现能力可安全降级。

本文件不覆盖：

1. Flutter 客户端开发（见 `app/` 项目）。
2. 前端页面开发（见 `frontend/` 项目）。
3. 第三方云厂商 API 业务规则细节（由插件自行实现）。

---

## 2. 系统架构与调用链

### 2.1 Host / Plugin 边界

Host（主系统）与 Plugin（插件进程）通过 Hashicorp `go-plugin` + gRPC 通讯：

1. 握手参数在 `backend/pkg/pluginsdk/handshake.go`：
   1. `ProtocolVersion = 1`
   2. `MagicCookieKey = XIAOHEI_PLUGIN`
   3. `MagicCookieValue = xiaoheiplay`
2. Host 必须先拿到 `core` 客户端，再按 `manifest` 声明挂载 `automation` 客户端。
3. 插件需要独立进程运行，入口由 `manifest.json` 的 `binaries` 指定。

关键实现锚点：

1. 运行时启动：`backend/internal/adapter/plugins/runtime.go`
2. 插件管理：`backend/internal/adapter/plugins/manager.go`
3. 插件安装：`backend/internal/adapter/plugins/install.go`

### 2.2 业务调用链（端到端）

以“商品类型已绑定 automation 插件实例”为前提，调用链如下：

1. 管理后台触发商品同步或用户发起 VPS 操作。
2. Usecase 层请求自动化客户端。
3. `Resolver` 从商品类型读取 `automation_plugin_id + automation_instance_id`，创建插件客户端：
   1. `backend/internal/adapter/automation/resolver.go`
4. `PluginInstanceClient` 将业务请求转换为 `plugin.v1.AutomationService` RPC：
   1. `backend/internal/adapter/automation/plugin_client.go`
5. `Manager.GetAutomationClient(...)` 确保对应插件实例已启动：
   1. `backend/internal/adapter/plugins/manager.go`
6. `Runtime.Start(...)` 进行握手、`GetManifest`、`Init`、心跳：
   1. `backend/internal/adapter/plugins/runtime.go`
7. 插件完成对第三方自动化系统 API 调用并回传结果。

### 2.3 管理端相关 API 路径

见 `backend/internal/adapter/http/router.go`，核心路径：

1. 插件安装管理：
   1. `GET /admin/api/v1/plugins`
   2. `POST /admin/api/v1/plugins/install`
   3. `POST /admin/api/v1/plugins/:category/:plugin_id/instances`
   4. `POST /admin/api/v1/plugins/:category/:plugin_id/:instance_id/enable`
   5. `PUT /admin/api/v1/plugins/:category/:plugin_id/:instance_id/config`
2. 自动化集成：
   1. `GET /admin/api/v1/integrations/automation`
   2. `PATCH /admin/api/v1/integrations/automation`
   3. `POST /admin/api/v1/integrations/automation/sync`
   4. `GET /admin/api/v1/integrations/automation/sync-logs`
3. 商品类型同步：
   1. `POST /admin/api/v1/goods-types/:id/sync-automation`

---

## 3. 插件协议与生命周期

### 3.1 协议定义文件

1. `backend/plugin/v1/core.proto`
2. `backend/plugin/v1/manifest.proto`
3. `backend/plugin/v1/types.proto`
4. `backend/plugin/v1/automation.proto`

### 3.2 `core` 服务（必选）

插件必须实现 `CoreService`：

1. `GetManifest(Empty) returns (Manifest)`
2. `GetConfigSchema(Empty) returns (ConfigSchema)`
3. `ValidateConfig(ValidateConfigRequest) returns (ValidateConfigResponse)`
4. `Init(InitRequest) returns (InitResponse)`
5. `ReloadConfig(ReloadConfigRequest) returns (ReloadConfigResponse)`
6. `Health(HealthCheckRequest) returns (HealthCheckResponse)`

生命周期时序（简化）：

1. Host 启动插件进程。
2. Host 调 `GetManifest` 并校验和 `manifest.json` 一致。
3. Host 调 `Init(instance_id, config_json)`。
4. Host 每 10s 调 `Health`（见 `runtime.go`）。
5. 配置更新时，Host 调 `ReloadConfig`。

### 3.3 `manifest` 一致性要求

Host 会比对以下字段（不一致直接启动失败）：

1. `plugin_id`
2. `name`
3. `version`
4. capability presence（是否声明 sms/payment/kyc/automation）
5. automation features 及 not_supported_reasons

参考：`validateManifestConsistency` in `backend/internal/adapter/plugins/runtime.go`

---

## 4. `automation` RPC 接口逐项说明

定义来源：`backend/plugin/v1/automation.proto`

### 4.1 目录同步能力

| RPC | 用途 | 关键请求字段 | 关键响应字段 | 幂等建议 |
|---|---|---|---|---|
| `ListAreas` | 拉取地域列表 | 无 | `items[].id/name/state` | 幂等 |
| `ListLines` | 拉取线路列表 | 无 | `items[].id/name/area_id/state` | 幂等 |
| `ListPackages` | 拉取套餐规格 | `line_id` | `items[].cpu/memory_gb/disk_gb/monthly_price` | 幂等 |
| `ListImages` | 拉取镜像 | `line_id` | `items[].id/name/type` | 幂等 |

### 4.2 实例生命周期能力

| RPC | 用途 | 关键请求字段 | 返回 | 失败语义 |
|---|---|---|---|---|
| `CreateInstance` | 创建实例 | `line_id` + 资源参数 | `instance_id` | 业务错误建议返回可读 message |
| `GetInstance` | 查询实例详情 | `instance_id` | `AutomationInstance` | 找不到实例返回 error |
| `ListInstancesSimple` | 简易实例搜索 | `search_tag` | `items[].id/name/ip` | 幂等 |
| `Start` | 开机 | `instance_id` | `Empty` | 非成功状态写入 `Empty.status/msg` |
| `Shutdown` | 关机 | `instance_id` | `Empty` | 同上 |
| `Reboot` | 重启 | `instance_id` | `Empty` | 同上 |
| `Rebuild` | 重装系统 | `instance_id/image_id/password` | `Empty` | 同上 |
| `ResetPassword` | 重置系统密码 | `instance_id/password` | `Empty` | 同上 |
| `ElasticUpdate` | 弹性变更 | `instance_id` + optional 资源字段 | `Empty` | 未传字段不应覆盖 |
| `Lock` | 锁定 | `instance_id` | `Empty` | 同上 |
| `Unlock` | 解锁 | `instance_id` | `Empty` | 同上 |
| `Renew` | 续费更新到期 | `instance_id/next_due_at_unix` | `Empty` | 幂等处理重复续期 |
| `Destroy` | 销毁实例 | `instance_id` | `Empty` | 建议幂等（已删除视为成功） |
| `GetPanelURL` | 获取面板链接 | `instance_name/panel_password` | `url` | URL 不落日志明文密码 |
| `GetVNCURL` | 获取 VNC 链接 | `instance_id` | `url` | 同上 |
| `GetMonitor` | 获取监控原始数据 | `instance_id` | `raw_json` | 返回合法 JSON 字符串 |

### 4.3 可选能力（未实现可返回 Unimplemented）

1. 端口映射：
   1. `ListPortMappings`
   2. `AddPortMapping`
   3. `DeletePortMapping`
   4. `FindPortCandidates`
2. 备份：
   1. `ListBackups`
   2. `CreateBackup`
   3. `DeleteBackup`
   4. `RestoreBackup`
3. 快照：
   1. `ListSnapshots`
   2. `CreateSnapshot`
   3. `DeleteSnapshot`
   4. `RestoreSnapshot`
4. 防火墙：
   1. `ListFirewallRules`
   2. `AddFirewallRule`
   3. `DeleteFirewallRule`

`PluginInstanceClient` 会把 gRPC `Unimplemented` 映射为业务 `ErrNotSupported`，见 `backend/internal/adapter/automation/plugin_client.go`。

---

## 5. 最小插件实现指南（Go）

### 5.1 目录结构（建议）

```text
my-automation-plugin/
  manifest.json
  checksums.json
  signature.sig
  schemas/
    config.schema.json
    config.ui.json
  bin/
    windows_amd64/plugin.exe
    linux_amd64/plugin
    darwin_amd64/plugin
    darwin_arm64/plugin
```

### 5.2 `manifest.json` 最小示例

```json
{
  "plugin_id": "my_automation",
  "name": "My Automation Plugin",
  "version": "0.1.0",
  "description": "Automation plugin example",
  "binaries": {
    "windows_amd64": "bin/windows_amd64/plugin.exe",
    "linux_amd64": "bin/linux_amd64/plugin",
    "darwin_amd64": "bin/darwin_amd64/plugin",
    "darwin_arm64": "bin/darwin_arm64/plugin"
  },
  "capabilities": {
    "automation": {
      "features": ["catalog_sync", "lifecycle"]
    }
  }
}
```

### 5.3 最小可运行代码骨架

参考实现：`backend/plugin-demo/pluginv1/automation_lightboat/main.go`

```go
package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type cfg struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	conf cfg
}

func (s *coreServer) GetManifest(context.Context, *pluginv1.Empty) (*pluginv1.Manifest, error) {
	return &pluginv1.Manifest{
		PluginId: "my_automation",
		Name:     "My Automation Plugin",
		Version:  "0.1.0",
		Automation: &pluginv1.AutomationCapability{
			Features: []pluginv1.AutomationFeature{
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_LIFECYCLE,
			},
		},
	}, nil
}

func (s *coreServer) GetConfigSchema(context.Context, *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	return &pluginv1.ConfigSchema{
		JsonSchema: `{"type":"object","properties":{"base_url":{"type":"string"},"api_key":{"type":"string","format":"password"}},"required":["base_url","api_key"]}`,
		UiSchema:   `{"api_key":{"ui:widget":"password"}}`,
	}, nil
}

func (s *coreServer) ValidateConfig(_ context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	var c cfg
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &c); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(c.APIKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "base_url/api_key required"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(_ context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	if strings.TrimSpace(req.GetConfigJson()) == "" {
		return &pluginv1.InitResponse{Ok: false, Error: "empty config"}, nil
	}
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(_ context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	var c cfg
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &c); err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: "invalid config"}, nil
	}
	s.conf = c
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(context.Context, *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
		Message:    "ok",
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

type autoServer struct {
	pluginv1.UnimplementedAutomationServiceServer
}

func (a *autoServer) ListAreas(context.Context, *pluginv1.Empty) (*pluginv1.ListAreasResponse, error) {
	return &pluginv1.ListAreasResponse{Items: []*pluginv1.AutomationArea{}}, nil
}

func main() {
	core := &coreServer{}
	auto := &autoServer{}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:       &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyAutomation: &pluginsdk.AutomationGRPCPlugin{Impl: auto},
	})
}
```

### 5.4 必须实现与建议实现

必须实现：

1. `CoreService` 全量方法。
2. `automation` 的目录同步 + 生命周期基础方法（至少满足你的业务路径）。
3. `manifest.json` 与 `GetManifest` 完全一致。

建议实现：

1. `ReloadConfig` 热更新。
2. 结构化错误透传（不要只返回 `"failed"`）。
3. 可选能力按需实现，未实现返回 `Unimplemented`。

---

## 6. 配置模型与敏感字段处理

### 6.1 Schema/UI Schema 规范

1. `GetConfigSchema` 需返回 `json_schema` + `ui_schema`。
2. 密钥字段建议：
   1. JSON Schema 使用 `"format": "password"`。
   2. UI Schema 使用 `"ui:widget": "password"`。
3. 建议在 UI help 明确“留空表示不修改”。

参考：

1. `backend/plugin-demo/pluginv1/automation_lightboat/main.go`
2. `backend/plugins/automation/lightboat/schemas/config.schema.json`
3. `backend/plugins/automation/lightboat/schemas/config.ui.json`

### 6.2 Host 对敏感字段的处理规则

`Manager` 会对非 automation 类插件进行敏感字段合并和脱敏。automation 当前走“每商品类型绑定”的配置路径，但你仍应遵循同样策略：

1. 日志中不输出 `api_key/secret/token` 明文。
2. 返回配置时应做脱敏展示。
3. 配置更新时支持“空值不覆盖旧密钥”。

`PluginInstanceClient` 已对日志 payload 做敏感键掩码（`***`），见：
`backend/internal/adapter/automation/plugin_client.go`

### 6.3 开源仓库隐私基线（必须）

1. 禁止提交真实 `API Key`、AccessKey、私有域名、生产服务器地址。
2. 示例统一使用占位符：
   1. `https://api.example.com`
   2. `YOUR_API_KEY`
3. 错误日志避免包含完整请求签名字符串。

---

## 7. 插件打包、签名与安装

### 7.1 编译插件二进制

示例（Linux）：

```bash
cd backend
go build -o ../tmp/plugin-build/plugin ./plugin-demo/pluginv1/automation_lightboat
```

将产物放入对应平台目录，例如：
`bin/linux_amd64/plugin`

### 7.2 生成签名文件

工具：`backend/cmd/tools/pluginsign/main.go`

命令：

```bash
cd backend
go run ./cmd/tools/pluginsign -dir ../tmp/my_automation_plugin
```

输出：

1. `checksums.json`
2. `signature.sig`

如需指定固定私钥（CI）：

```bash
go run ./cmd/tools/pluginsign -dir ../tmp/my_automation_plugin -ed25519-priv "<BASE64_PRIVATE_KEY>"
```

### 7.3 打包格式

安装器支持：

1. `.zip`
2. `.tar.gz` / `.tgz`

安装校验规则（见 `backend/internal/adapter/plugins/install.go`）：

1. 包内必须且只能有一个 `manifest.json`。
2. 插件目录必须能解析出 `<category>/<plugin_id>` 或 `plugins/<category>/<plugin_id>`。
3. `manifest.plugin_id` 必须与目录名一致。
4. 当前平台必须存在可执行文件。

### 7.4 后台安装流程

1. `POST /admin/api/v1/plugins/install` 上传包。
2. 成功后在插件列表可见。
3. 创建实例：`POST /admin/api/v1/plugins/:category/:plugin_id/instances`
4. 更新实例配置：`PUT /admin/api/v1/plugins/:category/:plugin_id/:instance_id/config`
5. 启用实例：`POST /admin/api/v1/plugins/:category/:plugin_id/:instance_id/enable`

---

## 8. 平台侧绑定与业务启用流程

### 8.1 标准启用步骤

1. 安装并启用 automation 插件实例。
2. 在“自动化集成”页确认可用实例。
3. 为商品类型绑定 `automation_plugin_id` 与 `automation_instance_id`。
4. 执行商品同步（地域/线路/镜像/套餐）。
5. 创建套餐、计费周期，完成业务上架。

### 8.2 同步操作建议

1. 先 `ListAreas`/`ListLines`，再 `ListPackages`/`ListImages`。
2. 同步失败后查看：
   1. `GET /admin/api/v1/integrations/automation/sync-logs`
3. 修复配置后再重试，不要在错误配置下连续重试。

### 8.3 常见冲突：无可写 automation 实例

症状：

1. 自动化配置接口返回冲突或提示不可写实例。
2. 页面可能引导跳转 `"/admin/catalog"`。

处理顺序：

1. 确认至少有一个 automation 插件实例已创建。
2. 确认实例已启用且配置合法。
3. 确认商品类型已绑定该实例。

---

## 9. 联调与验收清单

### 9.1 静态一致性校验

1. `manifest.json` 与 `GetManifest` 一致。
2. `capabilities.automation.features` 与实际实现一致。
3. `binaries` 覆盖目标发布平台。

### 9.2 启动与健康校验

1. 插件可被 Host 启动，不报握手错误。
2. `Init` 返回 `ok=true`。
3. `Health` 周期性可达。

### 9.3 业务路径校验（最小）

1. 目录同步成功：
   1. `ListAreas`
   2. `ListLines`
   3. `ListPackages`
   4. `ListImages`
2. 生命周期至少通过三项：
   1. `CreateInstance`
   2. `Start` 或 `Shutdown`
   3. `Destroy`

### 9.4 可选能力校验

1. 已实现能力：功能可执行、字段完整。
2. 未实现能力：返回 `Unimplemented`，系统展示“not supported”而非 500 崩溃。

---

## 10. 常见错误与排障手册

### 10.1 握手失败

现象：

1. 启用插件时报连接失败或握手失败。

定位点：

1. `backend/pkg/pluginsdk/handshake.go`
2. 插件 `pluginsdk.Serve(...)` 是否加载同一握手配置。

处理动作：

1. 对齐 `ProtocolVersion` 与 MagicCookie。
2. 确认插件进程可执行权限与路径正确。

### 10.2 `manifest mismatch`

现象：

1. 启动时报 `manifest mismatch: ...`

定位点：

1. `validateManifestConsistency` in `backend/internal/adapter/plugins/runtime.go`

处理动作：

1. 对齐 `plugin_id/name/version`。
2. 对齐 automation features 列表与原因字段。

### 10.3 配置校验失败

现象：

1. 更新配置后报 `missing required config` 或 `invalid config`。

定位点：

1. `ValidateConfig` 插件实现。
2. `GetConfigSchema` required 字段。

处理动作：

1. 先修 schema，再修配置 payload。
2. 对 secret 字段使用占位值验证流程，不要提交真实凭据。

### 10.4 RPC 超时/上游 API 错误

现象：

1. 同步或实例操作超时。
2. 返回上游 4xx/5xx 错误。

定位点：

1. 插件 HTTP client 超时与重试参数。
2. 自动化日志表（由 `PluginInstanceClient.logRPC` 记录）。

处理动作：

1. 降低并发，拉长 timeout，控制重试仅作用于幂等请求。
2. 解析上游错误 message 并透传可读信息。

### 10.5 可选能力未实现

现象：

1. 某功能页面提示不支持。

期望行为：

1. 插件返回 `Unimplemented`。
2. Host 映射为 `ErrNotSupported`，业务侧优雅提示，不应 panic。

---

## 11. 安全与开源发布注意事项

1. 不在仓库提交：
   1. 生产 API Key
   2. 私有域名与内网地址
   3. 私钥/签名密钥
2. 配置模板必须使用占位符。
3. 日志默认脱敏，必要时关闭 debug 日志。
4. 发布前确认：
   1. `checksums.json` 已更新
   2. `signature.sig` 与当前内容匹配
   3. 插件包内仅包含必要二进制与 schema/manifest

---

## 12. 附录

### 12.1 `manifest.features` 字段映射

| manifest feature 字符串 | proto 枚举 |
|---|---|
| `catalog_sync` | `AUTOMATION_FEATURE_CATALOG_SYNC` |
| `lifecycle` | `AUTOMATION_FEATURE_LIFECYCLE` |
| `port_mapping` | `AUTOMATION_FEATURE_PORT_MAPPING` |
| `backup` | `AUTOMATION_FEATURE_BACKUP` |
| `snapshot` | `AUTOMATION_FEATURE_SNAPSHOT` |
| `firewall` | `AUTOMATION_FEATURE_FIREWALL` |

### 12.2 返回码/错误消息建议

1. 成功操作优先返回 `Empty{status:"success",msg:"ok"}`。
2. 业务失败返回可读错误（可包含上游 msg，但不要含敏感字段）。
3. 未实现能力返回 gRPC `Unimplemented`。

### 12.3 发布检查表（Checklist）

1. 编译：
   1. 各平台二进制可执行。
2. 协议：
   1. `core` 与 `automation` 服务注册正确。
3. 配置：
   1. `GetConfigSchema/ValidateConfig` 一致。
4. 安装：
   1. 打包结构正确，只有一个 `manifest.json`。
5. 签名：
   1. 重新生成 `checksums.json + signature.sig`。
6. 联调：
   1. 目录同步成功。
   2. `CreateInstance` + `Start/Shutdown` + `Destroy` 通过。
7. 安全：
   1. 仓库无真实密钥/私有地址。

---

## 参考代码与文件索引

1. 协议：
   1. `backend/plugin/v1/core.proto`
   2. `backend/plugin/v1/manifest.proto`
   3. `backend/plugin/v1/types.proto`
   4. `backend/plugin/v1/automation.proto`
2. 插件 SDK：
   1. `backend/pkg/pluginsdk/handshake.go`
   2. `backend/pkg/pluginsdk/serve.go`
3. 插件管理：
   1. `backend/internal/adapter/plugins/runtime.go`
   2. `backend/internal/adapter/plugins/manager.go`
   3. `backend/internal/adapter/plugins/install.go`
4. 自动化适配：
   1. `backend/internal/adapter/automation/resolver.go`
   2. `backend/internal/adapter/automation/plugin_client.go`
5. API 路由：
   1. `backend/internal/adapter/http/router.go`
6. 示例实现：
   1. `backend/plugin-demo/pluginv1/automation_lightboat/main.go`
   2. `backend/plugins/automation/lightboat/manifest.json`

