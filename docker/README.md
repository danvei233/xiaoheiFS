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

首次安装写入 MySQL 后，后端会正常退出一次以重新加载安装配置。`restart: unless-stopped` 会自动启动新进程，无需手工重启容器。

## 持久化

Compose 使用以下命名卷：

- `app-config`：`app.config.yaml` 与 `install.lock`
- `app-uploads`：上传文件
- `app-plugins`：内置及后续安装的插件
- `app-data`：应用数据目录与 `init.lock`
- `mysql-data`：MySQL 数据

入口脚本把 `/app/app.config.yaml`、`/app/install.lock` 和 `/app/init.lock` 链接到对应持久卷，并在每次启动时把当前镜像的内置插件同步进插件卷。第三方插件目录不会被删除。

## 健康检查

镜像与 Compose 会先请求 `GET /api/v1/install/status`。安装前该接口可直接证明安装器存活；安装后还会请求读取数据库的 `GET /api/v1/site/settings`，数据库不可用时容器会进入 unhealthy。

## CI 发布范围

`.github/workflows/docker-build.yml` 构建 Debian 与 Alpine 镜像并推送到 Docker Hub。当前只发布 `linux/amd64`：内置插件 manifest 尚未声明 `linux/arm64`，在完成真实的平台验证前不会发布缺少可运行插件的 ARM64 镜像。
