# 更新清单签名契约（Update Manifest Signing Contract）

> 适用对象：`app/xiaoheifs_userapp`（用户端）与 `app/xiaoheifs_app`（管理端）的
> 应用自更新通道。两端共享同一契约，只有内嵌公钥表不同。

## 1. 目标

客户端「检查更新」返回的任何待安装清单，都必须是**真实发布端签名**的，且签名覆盖
**规范化且无歧义**的字段与制品 hash。否则客户端一律拒绝（fail closed），不会展示
更新、更不会安装。

这替代了早期「清单自带 SHA-256 即视为可信」的做法：SHA-256 只保证下载字节未被
篡改，不能证明清单本身来自发布端。现在真实性由 Ed25519 签名保证，`sha256` 是
被签名字段之一——发布端对「url + sha256 + 版本 + 时效」负责签名，客户端在验签
通过后才信任这些字段。

**下载完整性模型（如实声明，P2-13 受控下载）**：客户端「立即更新」走**受控下载**
——`consentAndLaunchUpdate` 先用 `downloadAndVerifyApk` 把 APK 下载到本地，按
清单 `sha256` **常量时间比对实际字节**，校验通过才把**已验证的本地文件**交给系统
安装器，且只有安装器成功交接后才记录「已同意安装」。因此：
- 清单声明的 `sha256` 约束**实际安装字节**：hash mismatch / 缺 sha256 / 下载失败
  一律 fail-closed（不启动安装、不写同意记录）；
- 下载通道由 TLS 保护（`apkUrl` 必须 https 且与更新服务器同源）；
- Android 经 FileProvider（`xiaohei/update` method channel）把已验证文件以
  `content://` 交给系统安装器，绝不直接暴露 `file://` 给第三方 intent；
- `openUpdateLink`（外部浏览器分发）仍作为底层 helper 保留，但**不再被
  「立即更新」调用**——外部浏览器无法约束安装字节，因此不能作为默认更新路径。

**签名覆盖全部影响更新决策的字段**（`hasUpdate`/`keyEpoch`/`latestVersion`/
`latestVersionCode`/`forceUpdate`/`apkUrl`/`sha256`/`changelog`/`issuedAt`）：
攻击者无法在不破坏签名的情况下改变「是否提示更新、下载地址、强制更新、版本、
时效」中的任何一项。

## 1.5 发布端托管（让签名更新通道真正可用）

客户端用 `GET {serverBaseUrl}{checkPath}?pkg=<包名>&versionCode=<当前版本>`
轮询更新（`serverBaseUrl` 与 `checkPath` 见 `update_config.dart`，默认
`https://pkg.example.com/api/v1/update/check`）。要让发布版 APK 真正收到更新，
发布者必须把**真实发布端签名的清单**托管在该路径下，并把真实 URL/公钥注入构建：

- **托管方式（三选一）**：
  1. **静态托管**：`script/update-sign` 输出签名清单 JSON，把该 JSON 作为静态
     文件发布到 `{serverBaseUrl}/api/v1/update/check`（客户端查询串
     `pkg`/`versionCode` 仅用于服务端决策；静态文件忽略查询串即可，清单本身
     已把 `pkg` 与 `hasUpdate` 等全部决策字段纳入签名）。GitHub Pages /
     OSS / 任意静态站均可。
  2. **小型只读服务**：任何返回该 JSON 的 HTTP 服务（Nginx 静态映射、Serverless
     Function 等）。
  3. **主后端端点**：由服务端实现 `/api/v1/update/check`（本仓库 release 端
     契约），按 `pkg`/`versionCode` 返回签名清单。
- **构建注入**：两个 release 工作流（`release-xiaoheifs-app.yml` /
  `release-xiaoheifs-userapp.yml`）的 `Resolve build-time dart-defines` 步骤把
  `UPDATE_SERVER_URL`（清单服务器 base URL）与 `UPDATE_SIGN_PUBLIC_KEY`（epoch 1
  生产公钥 hex）经 `--dart-define` 传给 `flutter build apk --release`。**发布
  （`release` 事件）时若未配置这两项，构建直接失败**（fail-closed），不会产出内嵌
  占位域名/占位公钥的「更新通道死掉的」APK；`workflow_dispatch` 手动构建只告警。
- **验证**：发布后应能看到签名清单被托管，且 `UPDATE_SIGN_PUBLIC_KEY` 与清单
  验签成功（`script/manifest-verifier-test` 的 golden vector 可离线复验）。

## 2. 清单 JSON 格式

`GET {serverBaseUrl}{checkPath}?pkg=...&versionCode=...` 返回：

```json
{
  "hasUpdate": true,
  "keyEpoch": 1,
  "latestVersion": "1.2.3",
  "latestVersionCode": 4,
  "forceUpdate": false,
  "apkUrl": "https://pkg.example.com/app-v1.2.3.apk",
  "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "changelog": "bug fixes",
  "issuedAt": 1754952800,
  "algorithm": "ed25519",
  "signature": "<base64>"
}
```

字段类型必须严格匹配（客户端按此契约校验，类型不符即拒绝）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `hasUpdate` | boolean | 是否提示更新，**参与签名** |
| `keyEpoch` | integer | 密钥代次（正整数），**参与签名**；客户端按它选择公钥 |
| `latestVersion` | string | 版本名，参与签名 |
| `latestVersionCode` | integer | 版本号，参与签名，客户端用它做防降级 |
| `forceUpdate` | boolean | 是否强制更新，参与签名 |
| `apkUrl` | string | https 下载地址，参与签名 |
| `sha256` | string | 制品 SHA-256（**小写** hex，64 字符），参与签名 |
| `changelog` | string | 更新说明（可为空串），参与签名 |
| `issuedAt` | integer | 签发 Unix 秒，参与签名，客户端用它做防重放 |
| `algorithm` | string | **必须为 `ed25519`**；其他算法一律拒绝 |
| `signature` | string | 对规范化载荷的 base64 Ed25519 签名 |

`hasUpdate=false` 时客户端直接返回无更新，不要求签名。**只要 `hasUpdate=true`，
必须携带有效签名，否则该更新不会被提供。** 注意：`hasUpdate` 参与签名，因此对
已签名清单翻转 `hasUpdate` 会破坏签名，客户端在验签**之前**不会信任 `hasUpdate`
的值（见 §5.11）。

## 3. 规范化载荷（Canonical Payload）

签名覆盖的字节串由固定顺序、转义后的字段行拼接而成（行分隔符为 LF `\n`）：

```
pkg=<escaped packageName>
hasUpdate=<1|0>
epoch=<keyEpoch>
version=<escaped latestVersion>
versionCode=<latestVersionCode>
force=<1|0>
url=<escaped apkUrl>
sha256=<escaped sha256>
notes=<escaped changelog>
issuedAt=<issuedAt>
```

- `packageName` 即客户端查询时上传的 `pkg`（把签名绑定到具体应用，跨应用重放无效）。
- `hasUpdate` 为 `1`/`0`；`force` 为 `1`/`0`。
- `keyEpoch`、`versionCode`、`issuedAt` 为十进制整数。
- 字段值转义规则：`\` → `\\`，LF → `\n`，CR → `\r`，TAB → `\t`，其他字符原样。
  转义保证「key=value 行」编码无歧义。
- **末尾不追加换行**：规范化载荷以 `issuedAt=<秒>` 结尾，无 trailing `\n`。
  Go 签名端（`script/update-sign`）与 Dart 验签端逐字节一致，跨实现由
  「Go 签名 → Dart 验签」往返测试与 golden vector 保证。

### 示例

对 `packageName="com.example.app"`, `hasUpdate=true`, `keyEpoch=1`,
`latestVersion="1.2.3"`, `latestVersionCode=4`, `forceUpdate=false`,
`apkUrl="https://pkg.example.com/app-v1.2.3.apk"`,
`sha256="0123…cdef"`, `changelog="bug fixes"`, `issuedAt=1754952800`：

```
pkg=com.example.app
hasUpdate=1
epoch=1
version=1.2.3
versionCode=4
force=0
url=https://pkg.example.com/app-v1.2.3.apk
sha256=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
notes=bug fixes
issuedAt=1754952800
```

## 4. 签名

- 算法：Ed25519（RFC 8032，纯 ed25519，不带 prehash）。
- 私钥：发布维护者持有（见第 6 节），对上述规范化载荷的 UTF-8 字节签名。
- `signature` = base64（标准 base64，含 padding）编码的 64 字节签名。

## 5. 客户端校验规则（fail-closed，任一不满足即整体拒绝）

1. `algorithm` 必须为 `ed25519`（不支持其他算法，不做降级）。
2. `signature` 必须存在、可 base64 解码、长度 64 字节。
3. `keyEpoch` 必须为正整数，且客户端内嵌该 epoch 的公钥（未知 epoch 即拒绝）。
4. 内嵌公钥必须存在、hex 可解码、长度 32 字节，且**不是全零占位公钥**
   （缺 key 或占位 key 一律拒绝，不允许「未配置=放行」）。
5. 被签名字段必须齐全且类型严格匹配第 2 节（含 `hasUpdate`/`keyEpoch`）。
6. `apkUrl` 必须为 https。
7. `sha256` 必须为小写 64 位 hex。
8. `issuedAt` 必须为正整数。
9. 用对应 epoch 的内嵌公钥验签，验签失败即拒绝。
10. **防降级**：`latestVersionCode` 必须 > 当前安装版本号，否则拒绝（旧版本签名清单
    不能作为更高版本被重放）。
11. **防重放（时间窗）**：`issuedAt` 必须在「当前时间 − 7 天」到「当前时间 + 5 分钟」
    窗内，过期或来自未来的清单均拒绝。
12. **防重放（跨会话持久）**：客户端在**安全存储**（`flutter_secure_storage` /
    Keychain/Keystore）中按 `packageName/keyEpoch` 维护「已接受的最高
    versionCode 与 issuedAt」；任何不比已记录值更新的签名清单（旧签名清单降级/
    重放）一律拒绝。`checkForUpdate` **先验签，再信任 `hasUpdate`**，之后才执行
    该跨会话重放检查，防止已接受过更高版本后旧清单被重放。
    **记录时机（F14 + round-5 交接顺序）**：记录只在用户**真正选择安装**时写入
    ——UI 在用户点击「立即更新」后调用 `UpdateService.consentAndLaunchUpdate`
    （`checkForUpdate` 只「检查可接受」、**不写记录**）。`consentAndLaunchUpdate`
    **先启动外部链接、成功交接后才 `recordAcceptedConsent`**：若启动失败（无浏览
    器/intent 处理者/系统拒绝），**不写记录**——同版本强更在下次启动仍会再次提示，
    不会因一次失败的交接被永久抑制。因此被用户「稍后」关闭或启动失败的非强制更新
    在下次启动仍会再次提示（同一签名清单不会因一次展示/一次失败启动就被永久吞掉）；
    只有已成功交接安装的清单在重放时才被拒绝。启动器可注入（`launcher` 参数），
    测试覆盖普通与强更的成功/失败路径。
13. **密钥轮换显式化**：`keyEpoch` 参与签名。同一构建可内嵌新旧两把公钥（双 key
    过渡），清单声明哪个 epoch 就用哪把验签；发布端用新 epoch 签名即可平滑切换，
    不需要「随手改 dart-define」这种非显式机制。

客户端实现：`app/xiaoheifs_userapp/lib/core/update/manifest_verifier.dart`、
`app/xiaoheifs_userapp/lib/core/update/update_replay_guard.dart` 与
`app/xiaoheifs_app/lib/services/manifest_verifier.dart`、
`app/xiaoheifs_app/lib/services/update_replay_guard.dart`；可执行测试：
`script/manifest-verifier-test/`（纯 Dart，无需 Flutter）。

## 6. 发布端签名与密钥轮换

### 6.1 密钥位置

- **公钥表**：客户端源码 `lib/core/update/update_keys.dart` /
  `lib/services/update_keys.dart`，按 keyEpoch 索引。**epoch 1 的生产公钥由发布
  构建通过 `--dart-define=UPDATE_SIGN_PUBLIC_KEY=<hex>` 注入**；未注入时使用全零
  占位公钥，验签器对占位公钥一律 fail-closed——**不带真实公钥的发布构建不会放行
  任何更新**（「未配置=放行」是禁止的）。
- **私钥**：只存在发布维护者的密钥库 / CI secret / 密钥管理服务中，**绝不允许进入
  仓库或客户端二进制**。泄露即视为安全事件，按 `SECURITY.md` 轮换。

### 6.2 签名工具与私钥输入

`script/update-sign`（Go）只从以下**恰好一个**受保护来源读取私钥，**绝不接受
`-key` 命令行参数**（argv 会泄漏到进程列表、shell history、CI 日志）：

1. 环境 secret：`UPDATE_SIGN_PRIVATE_KEY`；
2. 权限为 0600 的私钥文件：`-key-file <path>`（group/other 可读会直接拒绝）；
3. stdin/fd：`-key-stdin`（可从 secret manager 管道输入）。

工具不打印私钥、不回显任何含密钥的输入；`-pkg`/`-manifest`/`-epoch` 为必填，
`-issued-at` 缺省为当前时间。输出为带 `algorithm` 与 `signature` 的清单 JSON。

### 6.3 轮换（新私钥上线，显式 epoch 双 key 过渡）

1. 生成新密钥对，**先**把新公钥以新 epoch（例如 `keyEpoch=2`）编入客户端公钥表，
   与旧 epoch（`keyEpoch=1`）同时内嵌，发布带**双公钥**的客户端。
2. 发布端用新私钥 + `-epoch 2` 签名，客户端按清单 `keyEpoch=2` 选择新公钥验签。
3. 等待足够旧客户端升级后，在后续发布中移除旧 epoch 的公钥。

## 7. 参考实现与测试

- Go 签名工具与规范载荷单测：`script/update-sign/`
  （`cd script/update-sign && GOWORK=off go test ./...`）。
- Dart 验签器 + 跨会话重放守卫全量反例测试：`script/manifest-verifier-test/`
  （`cd script/manifest-verifier-test && dart pub get && dart test`）。
- 两端共享 golden vector：`canonical_test.go` 的 `TestGoldenVector` 与
  `manifest_verifier_test.dart` 的 `golden vectors` 组钉死同一
  seed/公钥/载荷，保证 Go 签名 → Dart 验签跨实现一致。
