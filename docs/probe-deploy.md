# 探针部署教程（pingbot）

这份文档按“从零到可用”写，直接照命令执行即可。

## 1. 部署目标
将 `pingbot` 部署到目标主机，让它持续上报主机状态到财务系统后端。

后端必须可访问以下接口：
- `POST /api/v1/probe/enroll`
- `POST /api/v1/probe/auth/token`
- `GET/WS /api/v1/probe/ws`（WebSocket）

## 2. 配置字段说明（必须看）
配置文件是 YAML，核心字段如下：

```yaml
server_url: "https://your-domain.com"
enroll_token: ""
probe_id: 0
probe_secret: ""
hostname_alias: ""
log_file_source: "file:logs"
tls_insecure_skip_verify: false
```

字段含义：
- `server_url`：后端地址，必须可访问，结尾不要带 `/`（程序会自动处理）
- `enroll_token`：一次性注册令牌，用于首次注册探针
- `probe_id`：探针 ID（注册成功后自动写入）
- `probe_secret`：探针密钥（注册成功后自动写入）
- `hostname_alias`：显示名称，不填则用主机名
- `log_file_source`：日志源，默认 `file:logs`
- `tls_insecure_skip_verify`：是否跳过 TLS 校验，生产环境建议 `false`

## 3. 两种接入模式

### 模式 A：首次接入（推荐）
适用：新机器第一次部署。

配置方式：
- `enroll_token` 填写后台生成的 token
- `probe_id = 0`
- `probe_secret = ""`

首次启动后，探针会：
1. 调用 enroll 接口注册
2. 自动把 `probe_id/probe_secret` 写回配置文件
3. 自动清空 `enroll_token`
4. 转入长期运行

### 模式 B：已有凭证直连
适用：迁移或重部署，已有 `probe_id + probe_secret`。

配置方式：
- `probe_id > 0`
- `probe_secret` 非空
- `enroll_token` 可留空

探针会跳过注册，直接鉴权并建立 WS 连接。

## 4. Linux 部署（systemd，推荐）

以下示例按路径：
- 二进制：`/opt/pingbot/pingbot`
- 配置：`/etc/pingbot/config.yaml`

### 4.1 构建二进制
```bash
cd pingbot
go build -trimpath -ldflags="-s -w" -o pingbot ./cmd/pingbot
```

### 4.2 准备目录与配置
```bash
sudo mkdir -p /opt/pingbot /etc/pingbot
sudo cp ./pingbot /opt/pingbot/pingbot
sudo chmod 755 /opt/pingbot/pingbot

# 复制一份配置模板
sudo cp ./config.yaml /etc/pingbot/config.yaml
sudo chmod 600 /etc/pingbot/config.yaml
```

编辑配置：
```bash
sudo vi /etc/pingbot/config.yaml
```

至少确认这几个字段：
- `server_url` 指向你的后端
- 首次部署用 `enroll_token`，并把 `probe_id/probe_secret` 置空

### 4.3 安装 systemd 服务
项目已提供模板：`pingbot/deploy/systemd/pingbot.service`

安装：
```bash
cd pingbot/deploy/systemd
sudo cp pingbot.service /etc/systemd/system/pingbot.service
sudo systemctl daemon-reload
sudo systemctl enable --now pingbot
```

> 也可直接用仓库脚本：
```bash
cd pingbot/deploy/systemd
sudo ./install.sh ../../pingbot ../../config.yaml
```

### 4.4 查看运行状态
```bash
sudo systemctl status pingbot --no-pager
sudo journalctl -u pingbot -f
```

成功日志通常包含：
- `pingbot starting ...`
- `enrolled probe_id=...`（首次）
- `ws connected probe_id=...`
- `running probe_id=...`

## 5. Windows 部署

## 5.1 构建
```powershell
cd pingbot
go build -trimpath -ldflags="-s -w" -o pingbot.exe .\cmd\pingbot
```

## 5.2 配置
建议放到固定路径：
- `C:\ProgramData\pingbot\config.yaml`（这是程序默认路径）

示例：
```powershell
New-Item -ItemType Directory -Force C:\ProgramData\pingbot | Out-Null
Copy-Item .\deploy\windows\pingbot-example.yaml C:\ProgramData\pingbot\config.yaml -Force
```

编辑 `C:\ProgramData\pingbot\config.yaml`，填写 `server_url` 和首次 `enroll_token`。

## 5.3 前台验证运行
```powershell
.\pingbot.exe -config C:\ProgramData\pingbot\config.yaml
```

确认日志出现 `ws connected` 后，再做服务化。

## 5.4 注册为服务（NSSM）
仓库提供脚本：
- `pingbot/deploy/windows/install-nssm.ps1`
- `pingbot/deploy/windows/uninstall-nssm.ps1`

示例（请按脚本参数要求执行）：
```powershell
powershell -ExecutionPolicy Bypass -File .\deploy\windows\install-nssm.ps1
```

## 6. 部署后验收（必须做）
1. 进程常驻：服务状态为 `active (running)` / Windows 服务为 `Running`
2. 日志稳定：无持续重连、无持续鉴权失败
3. 后台可见：探针在管理后台显示在线
4. 数据有效：CPU/内存/磁盘/端口数据有刷新

## 7. 高概率问题与处理

### 7.1 `missing enroll_token`
原因：首次接入但没填 token，且 `probe_id/probe_secret` 为空。
处理：填写 `enroll_token` 或改为已分配凭证模式。

### 7.2 `http 401/403` 鉴权失败
原因：`probe_id/probe_secret` 不匹配或已失效。
处理：在后台重置探针凭证后更新配置。

### 7.3 一直连不上 WS
排查顺序：
1. `server_url` 是否正确
2. 目标端口/防火墙是否放通
3. 反向代理是否支持 WebSocket 升级
4. HTTPS 证书是否可信（不要长期用 `tls_insecure_skip_verify=true`）

### 7.4 配置改了但不生效
原因：服务仍在用旧配置路径。
处理：检查 `ExecStart` 里的 `-config` 参数，重启服务。

## 8. 卸载
Linux：
```bash
cd pingbot/deploy/systemd
sudo ./uninstall.sh
```

该脚本会删除 systemd 服务定义，但保留 `/etc/pingbot` 与 `/opt/pingbot` 数据。
