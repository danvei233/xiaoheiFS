# Docker Build

本目录包含用于构建和运行 xiaoheiFS 应用的 Docker 配置。

## Dockerfile 说明

### `Dockerfile` (Debian - 默认)
基于 Debian bookworm-slim，提供全面的系统支持。

**优点：**
- 更广泛的工具和库支持
- 更好的兼容性
- 适合生产环境

**镜像大小：** ~400-500MB

### `Dockerfile.alpine`
基于 Alpine Linux，极小化镜像大小。

**优点：**
- 更小的镜像尺寸 (~150-200MB)
- 快速启动时间
- 适合云环境和边缘计算

**缺点：**
- 某些库可能不可用

## 本地构建

### 构建 Debian 版本
```bash
docker build -f docker/Dockerfile -t xiaohei:debian .
```

### 构建 Alpine 版本
```bash
docker build -f docker/Dockerfile.alpine -t xiaohei:alpine .
```

## 使用 Docker Compose 本地测试

```bash
docker-compose -f docker/docker-compose.yml up
```

这将同时运行两个版本的应用：
- Debian 版本：http://localhost:8080
- Alpine 版本：http://localhost:8081

## 运行容器

### Debian 版本
```bash
docker run -d -p 8080:8080 --name xiaohei-debian xiaohei:debian
```

### Alpine 版本
```bash
docker run -d -p 8080:8080 --name xiaohei-alpine xiaohei:alpine
```

## GitHub Actions 自动构建

提交代码到 `main` 或 `develop` 分支时，GitHub Actions 会自动：
1. 构建两个版本的镜像（Debian 和 Alpine）
2. 推送到 GitHub Container Registry (GHCR)

### 镜像标签

pushed 到 GHCR 的镜像将包含以下标签：

- `ghcr.io/username/xiaoheiFS:latest` (Debian)
- `ghcr.io/username/xiaoheiFS:latest-alpine` (Alpine)
- `ghcr.io/username/xiaoheiFS:v1.0.0` (基于 git 标签)
- `ghcr.io/username/xiaoheiFS:sha-abc123def456` (基于提交 SHA)

### 拉取镜像

```bash
# 拉取 Debian 版本
docker pull ghcr.io/username/xiaoheiFS:latest

# 拉取 Alpine 版本
docker pull ghcr.io/username/xiaoheiFS:latest-alpine
```

## 环境变量

应用支持以下环境变量：

- `LOG_LEVEL`: 日志级别 (默认: info)
- 其他应用级别的环境变量可根据需要添加

## 健康检查

两个镜像都配置了健康检查，会定期检查 `/health` 端点：

- 间隔：30 秒
- 超时：3 秒
- 启动延迟：40 秒
- 失败阈值：3 次

## 多平台支持

GitHub Actions 工作流配置为支持多个平台：
- `linux/amd64` (x86-64)
- `linux/arm64` (ARM64/Apple Silicon)

注意：本地构建默认为当前平台。若要跨平台构建，使用：

```bash
docker buildx build --platform linux/amd64,linux/arm64 -f docker/Dockerfile -t xiaohei:debian .
```
