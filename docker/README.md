# Docker Build

本目录提供 Debian、Alpine 两种镜像，以及 Debian + MySQL 的 Compose 部署配置。两个镜像都会从源码构建后端、前端静态资源和 manifest 明确支持当前平台的内置插件。

## 本地构建

```bash
docker build --platform linux/amd64 -f docker/Dockerfile -t xiaohei:debian .
docker build --platform linux/amd64 -f docker/Dockerfile.alpine -t xiaohei:alpine .
```

## Compose 启动

```bash
cat > .env <<'EOF'
MYSQL_ROOT_PASSWORD=replace-with-a-strong-root-password
MYSQL_PASSWORD=replace-with-a-strong-app-password
EOF
docker compose -f docker/docker-compose.yml up -d --build
docker compose -f docker/docker-compose.yml logs -f xiaohei
```

应用默认监听 `http://localhost:8080`。MySQL 只在 Compose 内部网络提供服务，不发布到宿主机；两个数据库密码必须通过 `.env` 显式设置。

镜像内的 Web 进程与插件子进程以非 root 用户 `xiaohei` 运行：入口脚本在启动时把运行时目录交给该用户并降权后再启动服务。健康检查（容器外由 Docker 守护进程以 root 触发）不受影响。若通过 `docker run` 自定义挂载目录，请确保目录属主与镜像内的 `xiaohei`（UID 10001）一致，或设置 `APP_USER` 覆盖目标用户名。

首次安装写入 MySQL 后，后端会正常退出一次以重新加载安装配置。`restart: unless-stopped` 会自动启动新进程，无需手工重启容器。

## 持久化

Compose 使用以下命名卷：

- `app-config`：`app.config.yaml` 与 `install.lock`
- `app-uploads`：上传文件
- `app-plugins`：内置及后续安装的插件
- `app-data`：应用数据目录与 `init.lock`
- `mysql-data`：MySQL 数据

入口脚本把 `/app/app.config.yaml`、`/app/install.lock` 和 `/app/init.lock` 链接到对应持久卷，并通过 `sync-plugins.sh` 同步内置插件。

### 插件同步策略（镜像托管 vs 管理员托管）

`docker/sync-plugins.sh` 区分两类插件，避免管理员手工安装/修改被镜像静默覆盖：

- **镜像托管插件**：镜像把内置插件拷入 `plugins/` 时会在 `config/plugin-bundle-state.tsv`
  记录插件路径与内容 SHA-256。重启时若当前内容与记录一致（未被改动），镜像升级会
  用新 bundle 替换它——因此镜像升级能拿到安全修复。
- **管理员托管插件**：管理员新装、或改动了镜像托管插件的文件后，其内容与记录不一致，
  同步会跳过并把它从镜像托管记录中移除，之后的启动都不会再覆盖它。
- **删除**：管理员删除镜像托管插件后，同步不会“复活”它（尊重删除）。
- **非 root**：同步脚本不要求 root，可在非 root 容器内正常写入插件卷与状态文件。
- **原子与可恢复**：每次拷贝先落临时目录并校验完整，再通过同文件系统两步重命名
  （旧版 → 备份，暂存 → 目标）交换；中途崩溃会保留旧版备份并在下次启动恢复，
  不会出现“删掉旧的、新的还没就位”的窗口。整轮任一托管插件失败即返回非零。
- **Fail-closed 入口**：入口脚本在同步后再执行 `--verify` 基线校验。任何托管插件
  无法应用到镜像安全基线（或基线校验失败）时容器拒绝启动——不会带着旧版、
  可能含漏洞的托管插件继续运行。管理员托管插件不受影响（本就跳过、不覆盖）。

如需手动干预，可直接编辑 `plugins/`。删除 `config/plugin-bundle-state.tsv`
的效果是：下次启动时，内容与镜像内置 bundle **完全一致**的插件会被重新记录为
镜像托管（之后镜像升级仍能覆盖它们）；被管理员改过/自装的插件内容不同，会保持
管理员托管，不会被镜像覆盖，也不再被记录。若想彻底恢复镜像内置插件（连被改过的
也重置），需删除 `plugins/` 下对应插件目录与状态文件后重启。

## 健康检查

镜像与 Compose 会先请求 `GET /api/v1/install/status`。安装前该接口可直接证明安装器存活；安装后还会请求读取数据库的 `GET /api/v1/site/settings`，数据库不可用时容器会进入 unhealthy。Compose 对 MySQL 使用 `mysqladmin ping`（不传密码，只探测存活）；MySQL 凭据错误会表现为 `xiaohei` 崩溃循环而不是 MySQL 不健康。

## CI 发布范围

`.github/workflows/docker-build.yml` 构建 Debian 与 Alpine 镜像并推送到 Docker Hub。当前只发布 `linux/amd64`：内置插件 manifest 尚未声明 `linux/arm64`，在完成真实的平台验证前不会发布缺少可运行插件的 ARM64 镜像。

注意：插件 manifest 声明了 `darwin_amd64` / `darwin_arm64`，但发布流水线（`release-build.yml` 的 `build-unix` matrix 与 `build-windows` job）只构建 `linux/amd64` 与 `windows/amd64` 的插件二进制。因此 macOS 用户安装发布包后，对应插件会因缺少 `bin/darwin_*/plugin` 而不可用。在补齐 darwin 构建之前，这属于"有声明、无产物"。
