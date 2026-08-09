# xiaoheiFS（小黑云财务）

> 一个面向云服务业务的自托管财务与运营系统，包含用户端、管理端、插件扩展和探针能力。

![status](https://img.shields.io/badge/status-alpha-orange)
![go](https://img.shields.io/badge/go-1.25.0-00ADD8?logo=go)
![vue](https://img.shields.io/badge/vue-3.x-42b883?logo=vue.js)
![license](https://img.shields.io/badge/license-GPL--3.0-blue)

## 为什么是 xiaoheiFS

- 一体化：用户下单、订单管理、VPS 生命周期、审计运维在同一套系统内完成
- 可扩展：支付、短信、实名、自动化等能力通过插件接入
- 可落地：提供 Web 前后端、Flutter 客户端、独立探针与 CI 构建流程

## 当前状态

- 阶段：`Alpha`
- 现状：核心链路（注册/登录、商品、下单、后台管理）可跑通
- 建议：适合测试、灰度与低风险试运行环境

## 功能概览

### 用户端

- 注册、登录、找回密码
- 商品浏览、购物车、下单、订单查看
- VPS 管理页（按后端能力启用）
- 钱包、工单、通知相关页面

### 管理端

- 用户与订单管理
- 商品类型、套餐、计费周期配置
- 系统参数、模板、审计日志与运维页面
- 插件管理与自动化对接

### 插件与探针

- 后端插件目录：`backend/plugins`
- 可通过后台启用、禁用和配置插件
- 探针项目：`pingbot`（独立 Go 服务）

## 技术栈

- 后端：Go 1.25.0 + Gin + GORM + validator
- Web：Vue 3 + Vite + Pinia + Element Plus（基于 Art Design Pro 模板），单一 SPA 承载公开站 / 用户控制台 / 管理后台
- 客户端：Flutter（管理员端/用户端）
- 数据库：MySQL / PostgreSQL / SQLite（通过 GORM）
- 插件机制：go-plugin + gRPC + protobuf

## 仓库结构

- `.github/workflows/`：发布与构建流水线
- `backend/`：后端主服务（API、业务、插件管理、内置插件）
- `frontend/`：**唯一** Web 前端。history 模式单页应用，承载公开营销站、登录注册、购物车、用户控制台（`/console`）、安装向导（`/install`）与管理后台（挂在后台配置的 `admin_path` 下）。后端从 `./static` 提供
- `adminweb/`：**已废弃**，正在并入 `frontend/`，迁移完成后整目录删除。不要在它上面开发新功能
- `app/`：Flutter 客户端工程
- `pingbot/`：探针服务
- `docs/`：部署与能力文档
- `docker/`：镜像与 Compose 部署配置
- `script/`：构建脚本

## 快速开始（本地开发）

### 前置环境

- Go `1.25.0`
- Node.js `20.19+` 或 `22.12+`（Vite 7 的要求；CI 与 Docker 使用 22）
- npm
- MySQL（推荐）或 SQLite（开发可用）

### 1) 启动后端

```bash
cd backend
go run ./cmd/server
```

说明：后端支持 `app.config.yaml` / `app.config.yml` / `app.config.json` 配置加载（详见 `backend/README.md`）。

### 2) 启动 Web 前端（frontend）

```bash
cd frontend
npm i
npm run dev
```

默认开发代理：
- `/api` -> `http://localhost:8080`
- `/admin/api` -> `http://localhost:8080`
- `/sdk` -> `http://localhost:8080`

说明：`script/build-linux.sh`、`script/build-win.bat` 和 `.github/workflows/release-build.yml` 当前默认构建的是 `frontend/` 静态资源。

### 3) 完成初始化安装

1. 访问 `http://localhost:8080/` 进入安装页
2. 填写数据库连接并初始化管理员账号
3. 访问 `http://localhost:8080/admin/login` 登录后台（`admin` 是默认后台路径，可在后台设置里修改）

## 构建与发布

### 一键构建

Linux:

```bash
./script/build-linux.sh
```

Windows:

```bat
script\build-win.bat
```

输出目录：
- Linux：`build/linux/`
- Windows：`build/windows/`

说明：一键构建打包 `backend + frontend`，不包含 Flutter App。`frontend/dist` 会被复制成运行目录下的 `static/`。

### Docker 快速启动（后端）

```bash
cat > .env <<'EOF'
MYSQL_ROOT_PASSWORD=replace-with-a-strong-root-password
MYSQL_PASSWORD=replace-with-a-strong-app-password
EOF

# 构建并启动后端、前端静态资源、内置插件和 MySQL
docker compose -f docker/docker-compose.yml up -d --build

# 查看服务日志
docker compose -f docker/docker-compose.yml logs -f xiaohei
```

首次安装完成后服务会退出一次以加载新配置，Compose 的重启策略会自动恢复服务。

更多说明见：`docker/README.md`

### CI 工作流

- `release-build.yml`：主系统构建与发布
- `release-pingbot-probe.yml`：探针发布
- `docker-build.yml`：Debian/Alpine 容器镜像构建与发布
- `release-xiaoheifs-app.yml`：管理端 App 发布
- `release-xiaoheifs-userapp.yml`：用户端 App 发布

## 子项目入口

- 后端说明：`backend/README.md`
- 前端说明：`frontend/README.md`
- 前端规范：`docs/frontend/adp-conventions.md`
- 前端改造方案：`docs/frontend/adp-migration-plan.md`
- 探针部署：`docs/probe-deploy.md`
- App 部署：`docs/app-deploy.md`
- 自动化插件开发：`docs/automation-plugin-development.md`

## 已知限制

- 仍有部分边界场景的异常处理和审计闭环待完善
- 处于 Alpha 阶段，不建议直接承载高风险生产流量

## 特别鸣谢

- duncai：支付实名接口与关键设计灵感
- kaqi：安全测试与漏洞修复建议
- kingbatsoft：安全测试与漏洞反馈
- luochen：测试与设计建议
- xiaohei：测试与设计建议，项目命名灵感来源
- Caius：实名、财务统计与审计能力设计灵感
- Pika：OpenIDC 系统开发与自动化插件接入支持
- xmccln：App 亮色模式设计灵感
- danvei233：项目维护者

排名不分先后。
