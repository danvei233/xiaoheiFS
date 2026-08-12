# final-fix.md — 集成候选终修复

**日期**: 2026-08-12
**集成候选 HEAD（修复前）**: `3de8234df8524ec2d2406a95bae28e1181d6a9ba`
**最终复核 HEAD**: `cdfe978e8fabcfc1035bc8c9cfe759740d27e60c`
**主要修复批次**: `20e4132b985d1c56d34adad153f6d27fe28868bd`
**基线**: `6c5cdef5`（PR #50 集成后）

本文件逐 finding 记录 12 份独立终审报告的处置：真实 finding 已修复并补回归测试，误报以精确代码/测试证据驳回。跨报告重复项只修一次。所有测试缓存位于 `/opt/tmp`。

---

## 一、跨报告重复项去重后的修复清单

### 1. P2-4 漏网端点：`POST /api/v1/orders/:id/refresh` 回传 `access_info`（auth F1 / finance F2 / provision F2 / plugins F-8 / ops F-1 / release F-1 / web_security F2?）

- **修复**: `backend/internal/adapter/http/handlers_site_orders_wallet.go` OrderRefresh 改用 `toUserVPSInstanceDTOsWithLifecycle`（P2-4 用户面 DTO，`AccessInfo=nil`）。
- **回归测试**: `backend/internal/adapter/http/p24_credential_leak_test.go` 新增 `TestP2_4_OrderRefreshDoesNotLeakAccessInfo`（先复现旧代码泄露 → 修复后通过，验证测试有效）。
- **证据**: 六份报告同指一处；修复前响应含 `"access_info":{"panel_password":"PANEL-SECRET",...}`。

### 2. legacy SQLite `CreateInstance` 恒失败（provision F1，P1 新回归）

- **根因**: `ensureVPSOrderItemUniqueIndex` 对 legacy SQLite（MySQL-dump）建**部分**唯一索引 `WHERE order_item_id > 0`，而 `CreateInstance` 用 `ON CONFLICT (order_item_id) DO NOTHING` —— SQLite 要求冲突目标精确匹配唯一约束，部分索引不被匹配，语句报 `ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`。
- **修复**: `backend/internal/adapter/repo/core/migrate_gorm.go` `ensureVPSOrderItemUniqueIndex`：legacy SQLite 也建**全量**唯一索引（`normalizeVPSOrderItemLinks` 已先置零/去重，NULL 不冲突），并把已存在的旧部分索引重建为全量索引（`sqliteIndexIsPartial` 探测）。
- **回归测试**: `backend/internal/adapter/repo/core/legacy_vps_create_test.go` `TestLegacySQLiteCreateInstanceIsIdempotentByOrderItem`、`TestEnsureVPSOrderItemUniqueIndex_RebuildsLegacyPartialIndex`。先复现（旧代码首插即败）→ 修复后通过。

### 3. CancelOrder vs 支付回调竞态：真金白银滞留（finance F1，P1）

- **根因**: `CancelOrder` 先对订单做非事务 CAS，随后**非 CAS** 遍历把 pending 支付单整行改写为 rejected；若回调在窗口内先置 approved，取消后钱已确认但订单已取消、无自动退款。
- **修复**: 新增 repo 方法 `CancelOrderAtomic`（`gorm_repo_order.go`）：同一事务内做订单 CAS + `NOT EXISTS(approved external payment)` 守卫 + 仅把仍 pending 的支付单置 rejected。`CancelOrder` 服务改用该方法；已 approved 外部支付单存在时拒绝取消（走退款流）。端口 `CancelOrderAtomic` 加入 `OrderRepository`，两个 fake 同步。
- **回归测试**: `backend/internal/app/payment/doublepay_regression_test.go` `TestPaymentService_ConcurrentCancelAndCallbackDoesNotStrandMoney`（10 轮并发，`-race` 通过）；`m014_fix_test.go` 的并发审批测试改用新路径后仍绿。

### 4. 管理端仪表盘 `Overview.Revenue` 口径与营收分析不一致（finance F3，P2）

- **根因**: `Overview` 只按订单状态计营收，不检查是否已有 approved 支付单；`collectRevenueDataByDimension` 要求 `approvedCount>0`。
- **修复**: `backend/internal/app/report/service.go` `Overview` 增加 `revenueRecognized`（正订单须有 approved 支付单；负退款单保留）。
- **回归测试**: `service_test.go` 更新 `TestOverviewRevenueUsesOrderStatusScope`/`TestOverviewRevenueExcludesRejectedAndDraft`（approved 无支付单不计营收）。

### 5. 前端模拟登录可被刷新「转正」为真实登录（auth F2 / plugins F-3 / web_perf F1，P1-21 绕过）

- **修复**: `frontend/src/stores/auth.ts` `applyImpersonatedToken` 清 `user_refresh_token` + bump user epoch；`frontend/src/utils/auth/refresh.ts` 用户 realm 在 `sessionKind==='impersonated'` 时拒绝刷新。
- **回归测试**: `refreshSingleflight.regression.test.mjs`、`authSessionKind.regression.test.mjs` 新增用例。

### 6. 前端 401 刷新-重试无限循环（database #3 / plugins F-4，P1-21 残余）

- **修复**: `frontend/src/services/http.ts` 重放前给请求打 `_retried` 标记；再 401 时跳过刷新直接登出。
- **回归测试**: `httpRealm401.regression.test.mjs`。

### 7. 管理端 MFA 解锁把 access token 当 refresh token 存（web_perf F2 / release F-2，P1-21）

- **修复**: `frontend/src/stores/adminAuth.ts` `completeMfa(accessToken, refreshToken)` 用 refresh token 写 refresh 槽；`views/admin/auth/login/index.vue` 传入 `response.data?.refresh_token`。
- **回归测试**: `authSessionKind.regression.test.mjs` 断言两个 token 槽位正确。

### 8. VPS 详情页在 `:id` 切换时闪显上一台 VPS 凭据（clients F3，P2）

- **修复**: `frontend/src/views/console/vps/detail.vue` `watch(() => route.params.id)` 清空 `credentials`/`showOsPassword`/`showPanelPassword` 并重新拉取。
- **回归测试**: store 层 `vpsSessionRace` 场景 3 覆盖；SFC 无法单测，用 lint/typecheck/build 验证。

### 9. P2-4 移除 `AccessInfo` 后用户 VPS 列表 IP 列恒显 "-"（web_perf F5，P2 跨层回归）

- **修复**: 新增 `frontend/src/utils/vpsAccess.ts` `resolveVpsIp` 优先读 `row.ip`，兜底 `access_info`；`normalizeRecord` 改用。

### 10. lightboat 自动化插件 http_trace 明文持久化 VPS 凭据（plugins F-1，P1，P0-13 残余）

- **修复**: `backend/internal/adapter/plugins/automation/client_logging.go` `sanitizePayload` 增加 form-urlencoded body 打码（`scrubFormBody`，`url.ParseQuery` + `isSensitiveKey`）；`plugin-demo/pluginv1/automation_lightboat/http_client.go` `emitLog` 落体前打码。
- **回归测试**: `client_log_test.go`（host 侧）+ `automation_lightboat/sanitize_test.go`。

### 11. Docker 镜像构建因 P2-65 签名门禁未接入密钥而断裂（plugins F-2 / clients F1，P1 发布阻断）

- **修复**: `docker/Dockerfile` + `Dockerfile.alpine` backend-builder 加 BuildKit secret mount（`--mount=type=secret,id=plugin_sign_key`）；`.github/workflows/docker-build.yml` 非 PR 注入 `secrets: plugin_sign_key`，PR 传 `PLUGIN_ALLOW_UNSIGNED=1`；`script/build-plugins.sh` 加 `--allow-unsigned` 逃生门（仅 CI PR）；`build-linux.sh` 无 key 时给出清晰指引。
- **回归测试**: `script/test-release-plugin-signing.sh` 20 项（含无 key 拒绝、`--allow-unsigned` 通过）。

### 12. 公开支付回调无限流 DoS 放大器（finance F6 / plugins F-5 / closure F3/F4，P1-4/P2-7）

- **修复**: `backend/internal/adapter/http/router_routes_public.go` 两条 notify 路由挂 per-IP `notifyRateLimit`（600/10min）；慢路径改为按 `payment_trade_no` 直接查询（`GetWalletOrderByPaymentTradeNo`，方言感知 JSON 提取），去掉 200×20 扫描上限。
- **回归测试**: notify 限流中间件 + 既有 `TestWalletPaymentNotify`。

### 13. P0-8 安装器任意 MySQL DSN SSRF（closure F1，P0）

- **修复**: `backend/internal/adapter/http/install.go` `validateInstallMySQLDSN` 拒绝非 tcp/空 host/不可解析 DSN；`InstallDBCheck` 与 `InstallRun` 前置校验；超时维持 3s。
- **回归测试**: `install_dsn_internal_test.go` `TestValidateInstallMySQLDSN`。

### 14. `vps.update` 吸收改价/改凭据/锁定能力（closure F2，P1-20）

- **修复**: `backend/internal/app/adminvps/service.go` 字段级权限门禁（`monthly_price→vps.price`、`access_info→vps.credentials`、`admin_status→vps.admin_status`、`spec→vps.resize`）；新增权限码注册；handler 把 `ErrForbidden` 映射 403。
- **回归测试**: `TestAdminVPSUpdate_SubAdminCannotChangePriceCredentialsStatus`。

### 15. 批量删除非事务/无上限/重复 ID（closure F7，P2-16）

- **修复**: `backend/internal/adapter/http/handlers_admin_catalog.go` `dedupeAndBoundIDs`（去重、拒非正数、上限 200）应用到全部 5 个 bulk-delete。
- **回归测试**: `TestDedupeAndBoundIDs`。

### 16. P1-13 usertier 异步重建无 panic recover（closure F8）

- **修复**: `backend/internal/app/usertier/service.go` 两个异步重建入口加 `defer recover`。
- **回归测试**: `TestRebuildGroupPriceCacheAsync_PanicDoesNotCrash`。

### 17. P2-8 能力策略多实例读改写丢更新（closure F9）

- **修复**: `backend/internal/adapter/repo/core/gorm_repo_settings_plugins.go` `UpdateSettingAtomic`（事务 + FOR UPDATE 行锁）；`admin.Service.UpdateSettingAtomic`；`saveCapabilityPolicy` 改用。
- **回归测试**: 既有 `TestConcurrentSaveCapabilityPolicy_NoLostUpdate` + 新路径。

### 18. P2-55 配置解析失败静默回退默认（closure F10）

- **修复**: `backend/internal/pkg/config/config.go` 存在配置文件但全部解析失败时 `fatalConfigLoad`（默认 `log.Fatalf`，测试可替换 hook）拒绝启动。
- **回归测试**: `TestLoadRefusesUnparseableConfig`（更新原 warn-and-fallback 测试为新语义）。

### 19. P2-12 AdminQQAvatar 未鉴权出站代理（closure F5）

- **修复**: 删除未使用且免鉴权的 `AdminQQAvatar` handler + 路由 + 2FA allowlist 条目（前端零引用）。
- **回归测试**: 删除 `handlers_robot_approve_test.go`（直接调用死 handler 的测试）。

### 20. P2-13 AdminPaymentProviders 原样回传渠道密文（closure F6）

- **修复**: `dto.go` `redactPaymentConfigJSON`（`isPaymentConfigSecretKey` 打码 merchant_key/app_secret/api_key 等）；`app/payment` `ConfigSecretMask` + `MergeConfigMask` + `UpdateProvider` 写回保留旧密文。
- **回归测试**: `TestPaymentProviderDTO_MasksSecrets` + 更新 `TestPaymentProviders_UserResponseWhitelistsPublicFields`（admin 响应含 mask 而非明文）。

### 21. P2-38 Scan readers 生产死代码（closure F18）

- **修复**: 删除 `gorm_repo_scan_readers.go`；仍被测试使用的 `scanUser`/`scanWalletOrder`/`scanner` 移入 `sqlite_repo_scan_wallet_test.go`。
- **回归测试**: 既有 scan 测试。

### 22. P3-3 / P3-17 死代码删除（closure F20/F23）

- **修复**: 删除 `toVPSInstanceDTOs` 与 `RobotApprove`/`RobotReject`。
- **回归测试**: `go build ./...`（无引用残留）。

### 23. P3-42 钱包概览只取最近 500 条（auth F4 / finance F4 / provision F4 / database #1 / ops F-2 / web_security F2 / release F-4）

- **修复**: `frontend/src/views/console/billing/index.vue` `loadOverview` 分页累加（有界 20 页）代替本地 latest-500 快照。
- **回归测试**: billing 行为 + 既有 wallet 分页测试。

### 24. P3-43 用户 VPS 列表全量拉取 + 本地分页（auth F3 / finance F5 / provision F5 / database #2 / ops F-3 / web_security F3 / release F-5 / tests F5）

- **修复**: 后端 `VPSList` 支持 `limit/offset/status` 并返回 `total`（`ListInstancesByUserFiltered`）；前端 `listVps`/`fetchVpsTable` 上送参数并消费 `total`（无 total 时降级本地）。
- **回归测试**: `TestP2_4_UserVPSResponsesDoNotLeakAccessInfo`（回归）+ 前端 vps 分页路径。

### 25. P3-45 custom_html 公开页从不渲染（closure F24）

- **修复**: `frontend/src/views/public/layout/index.vue` 渲染 `custom_html` 区块（按页面 key + footer 作用域）。

### 26. P3-47 analytics_code 未校验（closure F25）

- **修复**: 服务端 `validateAndNormalizeSettingValue` 拒绝 `<script`/`<iframe`/`onerror=` 内容；前端确认 textarea-only、无 v-html。

### 27. P3-54 文档引用已删除的 adminweb（closure F26）

- **修复**: `AGENTS.md` 更新当前状态段（adminweb 已删除、迁移完成）；迁移计划文档仅保留历史引用。

### 28. P2-61 archtest HTTP 边界缺 gorm/SQL/db/plugins（tests F3 / closure F17）

- **修复**: `backend/internal/archtest/dependency_test.go` 边界表补 `adapter/plugins`、`pkg/db`、`gorm.io/`、`database/sql`。
- **回归测试**: `go test ./internal/archtest -run TestHTTPAdapterDependencyBoundary`。

### 29. P2-60 测试 fake 默认过于乐观（closure F19 / plugins F-13 / tests F6）

- **修复**: `FakeAutomationClient.Strict` 模式（未配置即失败/不伪造 HostID）。
- **回归测试**: `TestFakeAutomationClient_StrictFailsClosed`。

### 30. P3-10 工单 ID 存在性 oracle（closure F15）

- **修复**: `TicketDetail`/`TicketMessageCreate`/`TicketClose` 对他人工单返回 404。
- **回归测试**: 更新 `TestHandlers_TicketAndNotifications`/`TestUserFlow_E2E`。

### 31. P3-27 / P3-28 紧急续费窗口与过期时间（closure F13/F14）

- **修复**: `loadEmergencyRenewPolicy` clamp `window<=0→1`、`emergencyRenewInWindow` 对 `<=0` 返回 false、设置校验拒绝非正数；`UpdateExpireAt` 拒过去时间；`SetAdminStatus` 联动用户态 locked。
- **回归测试**: `TestEmergencyRenewInWindow_NonPositiveWindowNeverAllowed`、`TestAdminVPSUpdateExpireAt_RejectsPastTime` 等。

### 32. P3-2 绑定错误与校验错误不区分（closure F22）

- **修复**: `domain.ErrBindFailed`/`ErrValidationFailed` + `classifyBindError` + `writeBodyError` 区分响应。
- **回归测试**: `TestBindJSON_DistinguishesBindFromValidationErrors`。

### 33. P3-4 JSON 设置键清单双份漂移（closure F21）

- **修复**: `app/shared/settings_keys.go` 单一来源 `JSONSettingKeys`，handler 与 repo 共用。
- **回归测试**: `TestIsLikelyJSONSettingKey_MatchesSharedSource` / `TestNormalizedListSettingKeys_MatchesSharedSource`。

### 34. P2-1 设置 JSON 校验死代码（closure F16）

- **修复**: `validateAndNormalizeSettingValue` default 分支对 `isLikelyJSONSettingKey` 键调用 `validateSettingJSONValue`。
- **回归测试**: `TestValidateAndNormalizeSettingValue_RejectsMalformedJSONForJSONKeys`。

### 35. P3-57 迁移每 boot DROP+CREATE（closure F27）

- **修复**: `ensureMySQLNonUniqueIndex` 幂等（方言感知探测 + 可移植 DropIndex）。
- **回归测试**: `TestEnsureMySQLNonUniqueIndex_IsIdempotentOnSecondRun`。

### 36. P3-60 repairTimestampNulls 全表扫描探测（database #7）

- **修复**: `repairOneTimestampNull` 用 `SELECT 1 ... LIMIT 1` 存在性探测代替 `COUNT(*)`。
- **回归测试**: `TestRepairTimestampNullsProbeShortCircuits`。

### 37. 优惠券注释声称不存在的唯一索引（finance F7 / database #5）

- **修复**: 新增 `ensureCouponRedemptionPerUserUniqueIndex`（SQLite/PG 部分唯一索引 `(coupon_id,user_id) WHERE status IN ('applied','confirmed')`，先归一化历史重复）；更正 `gorm_repo_coupon.go` 注释。
- **回归测试**: `TestEnsureCouponRedemptionPerUserUniqueIndex_CreatesGateAndDedupes`。

### 38. payment_demo V2 参考实现丢弃 query/body（clients F4，P3）

- **修复**: `plugin-demo/payment_demo/main.go` `rawToParams` 合并 headers + query + form body（对齐 `notify_params.go`）。
- **回归测试**: `main_test.go` `VerifyNotifyV2_QueryAndFormBody` + net/rpc round-trip。

### 39. legacy net/rpc 非 Linux fail-closed 守卫死代码（auth 备注 / plugins F-9 / ops F-4 / web_security F4 / clients F5 / release F-3 / tests F2）

- **修复**: `backend/internal/adapter/payment/registry.go` 增加 `legacyExecSupported = runtime.GOOS == "linux"`，非 Linux 返回 nil handle → `snapFile==nil` 分支跳过该插件而非整批失败；更正注释。
- **回归测试**: `TestRegistry_LegacyPluginSkippedOnNonLinux`。

### 40. 前端 npm lint 3 处 prettier 错误（tests F1，门禁损坏）

- **修复**: `authSessionKind`/`cart`/`vpsSessionRace` regression 测试文件 3 处逗号后补空格。
- **回归测试**: `npm run lint` 0 errors。

### 41. .gitignore 未覆盖私钥/keystore（clients F6，P3）

- **修复**: `.gitignore` 补 `*.key/*.pem/*.jks/*.p12/*.pfx/*.crt/*.keystore`。
- **回归测试**: `git check-ignore`。

### 42. 演示插件 HTTP 客户端默认 12s < 宿主 120s 预算（auth F5，P3）

- **修复**: 三个 demo 插件默认 120s、schema `timeout_sec` default/maximum=120。
- **回归测试**: 各插件 `main_test.go` 断言 `>=120`。

### 43. admin Flutter 启动不检测过期 token（clients F7，P3）

- **修复**: `app/xiaoheifs_app/lib/app_state.dart` `load()` 比较 `tokenExpiresAt`，过期即清会话。
- **回归测试**: `test/app_state_expired_token_test.dart`（本机无 Flutter SDK，未能运行；CI `flutter-test` 验收）。

### 44. QZ 面板/VNC 默认 http（clients F8，P3）

- **修复**: `qz-override/.../CloudPolicy.php` 默认 https；公网明文端点返回空串。
- **回归测试**: `php qz-override/tests/CloudPolicyTest.php` 141 项。

### 45. 终修复后的独立复核残余（P2-16 / P3-42 / P3-43 / 迁移保护）

- **P2-16**: 五类 catalog 批量删除改为仓储事务内原子执行；任一引用冲突会回滚此前删除。回归 `TestHandlers_AdminRegionBulkDeleteRollsBackOnConflict`。
- **P3-42**: 新增 `/api/v1/wallet/orders/summary`，由 SQL 聚合本月充值/提现、全部待处理与总数；回归覆盖 10,001 条以上历史，前端不再翻页拉全表。
- **P3-43**: 用户 VPS 的 status/keyword/region/expire_days 全部在 COUNT 与分页前服务端过滤；回归覆盖匹配项原本位于下一页、跨用户隔离及前后端状态枚举。
- **迁移数据保护**: 运行中 VPS 的重复 order_item 链接不再被静默解绑，迁移记录冲突并 fail-closed；重复手机号也在改 schema 前返回可操作错误。
- **安装器 DSN**: P0-8 校验追加 loopback/private/link-local/unspecified 地址拒绝测试。

---

## 二、误报驳回（false-positive 处置）

以下为各报告中的误报/已由既有修复覆盖项，未重复改动：

| 项 | 驳回理由（精确证据） |
|---|---|
| P2-51「非 Linux path-based fallback」注释 | 代码实际仅 fd-bind exec，无路径回退；安全属性成立（F-9 已改为显式平台门）。 |
| 管理侧 `AccessInfo`（web_security F1） | 管理端 VPS 管理属预期保留凭据能力；`vps.list` 只授予可信操作员；与数据库报告「管理员管理凭据，属预期」一致。已通过 P1-20 字段级门禁收紧改价/改凭据能力。 |
| P2-13「仅主管理员可达不构成越权」 | web_security 2.11 已文档化；本批次仍做了密文打码（见修复 #20），双保险。 |

---

## 三、实测命令与结果

所有命令在 `/opt/tmp/xiaohei-fix-all-20260811/worktrees/integration`（Go 在 `backend/` 下）执行，缓存 `GOCACHE=/opt/tmp/xiaohei-fix-all-20260811/cache/final-fix/gocache`、`GOPROXY=off`。

```bash
# 后端
go build ./...                                          # PASS
go vet ./...                                            # 0 告警
go test ./... -count=1 -timeout 1800s                   # 全包 PASS（cdfe978e 复跑）
go test -race ./internal/app/payment/ ./internal/app/order/ \
  ./internal/app/walletorder/ ./internal/adapter/payment/ \
  ./internal/adapter/plugins/payment/ ./internal/adapter/plugins/automation/ \
  -count=1                                              # PASS
go test ./internal/archtest/ -count=1                   # PASS
go test ./internal/adapter/repo/core/ -count=1          # PASS（含 PG 多方言）

# 前端（frontend/ 下）
npm run lint                                            # 0 errors
npm run typecheck                                       # PASS
npm run build                                           # PASS
node --test src/**/__tests__/*.mjs                      # 32/32 PASS（当前 glob）

# 客户端 / 发布 / QZ / Dart / Python
bash script/test-release-plugin-signing.sh              # 20/20 PASS
bash script/test-sync-plugins.sh                        # 34/34 PASS
php qz-override/tests/CloudPolicyTest.php               # 141/141 PASS
python3 script/check-qz-contracts.py                    # PASS
python3 script/check-flutter-contracts.py               # PASS
python3 script/check-dockerignore-contract.py           # PASS
node script/release-smoke.mjs                           # PASS
```

## 四、台账与校验

- `/opt/tmp/xiaohei-fix-all-20260811/reports/issue51-ledger.tsv` 已更新为 193 行最终台账，状态仅 `fixed`（190）/ `external`（3）。
- 校验：`node /opt/tmp/xiaohei-fix-all-20260811/validate-issue51-ledger.mjs` → **`issue51 ledger valid: 193 items`**。
- 3 项 external：P0-10（pingbot probe_secret 历史提交）、P2-63（pingbot root/secret 部署加固 + 历史泄露）、P2-85（docs 历史真实凭据）——均为仓库外动作（凭据轮换/git 历史重写/系统用户配置），代码侧缓解已完成（凭据从仓库移除、部署脚本非 root 降权、docs 脱敏）。

## 五、环境受限项（如实声明）

- **MySQL 三方言**：本机无 MySQL 客户端/凭据；迁移逻辑与 SQLite/PG 共享入口，PG 多方言实测通过；MySQL 生成列/部分唯一索引改写在 CI `backend-go` 验收。
- **Docker 容器级断言**：本机 docker daemon 未运行；`test-sync-plugins.sh`/`test-release-plugin-signing.sh` 覆盖沙箱逻辑。
- **Flutter `flutter test`**：本机仅 Dart SDK；`check-flutter-contracts.py` PASS，`app_state_expired_token_test.dart` 由 CI `flutter-test` 运行。
