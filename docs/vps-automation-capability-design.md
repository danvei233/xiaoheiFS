# VPS 自动化能力动态化基础设计

## 目标
- 前端按能力动态隐藏功能页（防火墙、端口映射、快照、备份）。
- 兼容已有静态插件能力声明（manifest.features）。
- 支持实例级动态能力覆盖：可新增能力、可否定能力。

## 当前能力来源
1. 静态能力（基线）
- 来源：自动化插件 manifest 的 `capabilities.automation.features`。
- 作用：作为默认能力集合。

2. 动态能力（实例覆盖）
- 来源：实例 `access_info` 中的能力覆盖数据（由插件在实例信息里回传）。
- 作用：对静态能力进行实例级增删改，适配同插件下不同产品能力差异。

## 动态能力字段规范（access_info）
建议插件在 `access_info` 中提供以下结构：

```json
{
  "capabilities": {
    "automation": {
      "features": ["firewall", "snapshot"],
      "add_features": ["backup"],
      "remove_features": ["port_mapping"],
      "not_supported_reasons": {
        "port_mapping": "当前机型不支持端口映射"
      }
    }
  }
}
```

兼容字段：
- `disabled_features`
- `deny_features`

说明：
- `features`：可选，若存在表示动态给出的完整能力基线。
- `add_features`：在当前集合上新增能力。
- `remove_features` / `disabled_features` / `deny_features`：在当前集合上移除能力。
- `not_supported_reasons`：返回被禁用能力原因，前端可用于提示。

## 能力合并规则
1. 初始化能力集合 = 静态 `manifest.features`。
2. 若动态 `features` 存在且非空：能力集合替换为动态 `features`。
3. 应用 `add_features`。
4. 应用 `remove_features`（含兼容字段）。
5. 输出最终能力集合 + `not_supported_reasons`。

## 前端行为
- 详情页 Tab 显示规则：
  - `firewall` -> 防火墙 Tab
  - `port_mapping` -> 端口映射 Tab
  - `snapshot` -> 快照 Tab
  - `backup` -> 备份 Tab
- 若后端未返回能力字段：保持兼容，默认全部显示（不改变旧行为）。
- 若当前激活 Tab 被隐藏：自动回退到第一个可用 Tab。

## 已落地接口影响
- `GET /api/v1/vps/:id` 增加 `capabilities` 字段。
- 字段示例：

```json
{
  "capabilities": {
    "automation": {
      "features": ["catalog_sync", "lifecycle", "snapshot"],
      "not_supported_reasons": {
        "port_mapping": "Not supported by upstream"
      }
    }
  }
}
```

## 后续建议
- 在插件开发文档中补充 `access_info.capabilities` 回传约定。
- 为动态能力合并逻辑补充单元测试（静态-only、动态覆盖、冲突字段、空值容错）。

## 魔方系统对齐冲突矩阵（基础版）
1. 商品模型映射
- 对齐项：两边都可抽象为 `商品类型 -> 地区 -> 线路 -> 套餐`。
- 冲突点：上游字段名和层级深度可能不一致（如 area/region、line/node）。
- 方案：统一走内部模型（GoodsType/Region/PlanGroup/Package），插件负责字段映射。

2. 生命周期动作
- 对齐项：开通、销毁、续费、开关机、重装、重置密码、面板登录、信息获取均可抽象为标准能力。
- 冲突点：部分产品不支持端口映射/快照/备份/升降配/退款。
- 方案：按实例能力动态否定，不支持即隐藏前端入口并在后端拒绝执行。

3. 升降配与退款开关归属
- 目标：开关从“全局设置”下沉到“商品/实例能力”。
- 当前实现：支持通过实例动态能力控制 `resize/refund`；全局开关保留为默认值（兜底）。
- 后续可选：增加“套餐级能力策略”管理页，覆盖默认值。

4. 库存同步任务范式
- 结论：已是范式任务（`integration_inventory_sync`），可配置启停与调度策略。
- 建议：生产默认关闭，按商品类型逐步启用，并结合同步日志观测。

## 套餐级开关落地（已实现）
- 配置存储：`setting.key = package_capabilities_json`。
- 数据结构示例：
```json
{
  "1001": { "resize_enabled": true, "refund_enabled": false },
  "1002": { "resize_enabled": false }
}
```
- 管理接口：
  - `GET /admin/api/v1/packages/:id/capabilities`
  - `PATCH /admin/api/v1/packages/:id/capabilities`
- 生效优先级：
  1. 套餐级开关（若配置）
  2. 全局默认开关（`resize_enabled` / `refund_enabled`）
  3. 实例动态能力覆盖（可继续否定/新增）

## 轻舟插件配置动态候选（已实现）
- 目的：商品类型绑定自动化实例后，为插件配置中的 `product_type_id/line_id/package_id` 提供自动候选列表，减少手填错误。
- 后端接口：
  - `GET /admin/api/v1/goods-types/:id/automation-options`
- 返回数据：
  - `line_items`：上游线路列表
  - `product_type_items`：默认复用线路列表作为“产品类型候选”
  - `package_items`：按线路展开的上游套餐列表
- 前端行为：
  - 商品类型页加载插件 schema 后，若存在上述字段，则自动注入 `enum + enumNames` 下拉选项。
  - 拉取失败时保持手填，不阻断保存流程（兼容静态配置）。

## 魔方 OpenAPI 插件（进行中）
- 新增独立插件：`automation/mofang_openapi`（与轻舟/代理插件分离）。
- 已实现：
  - JWT 鉴权（`POST /v1/login_api`，自动处理 405 重新登录）。
  - 基础生命周期：查询实例、开机/关机/重启、重置密码、重装系统、续费、提交停用。
  - 面板能力：通过 `module/vnc` 获取访问 URL。
  - 商品发现：通过 `GET /v1/products` 输出目录层级与套餐基础信息。
  - 开通流程（基础版）：`cart/products + cart/checkout + hosts轮询` 自动发现新实例 ID。
- 当前按能力禁用（前后端统一隐藏）：
  - 端口映射、备份、快照、防火墙（魔方 OpenAPI 无统一标准接口）。
- 待补强：
  - 套餐规格字段精确映射（CPU/内存/磁盘/带宽）与多配置项产品的创建参数编排。
  - `module/status/charts` 的监控指标标准化映射。
