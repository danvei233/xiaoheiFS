# 小黑云控制台（Cloud VPS Console）

一个完整的 VPS 管理平台：官网/产品页 + 用户控制台 + 管理后台。后端负责鉴权、订单/财务、VPS 管理与自动化对接；前端提供可视化控制台与 CMS 内容渲染。

## 功能概览
- 用户端：注册登录、产品目录、购物车/下单、订单与账单、VPS 操作（开机/关机/重装/快照/备份/端口映射）、工单、消息中心、实名认证
- 管理端：用户/订单/VPS 管理、产品目录、支付插件、系统设置、权限组、CMS（区块/文章/导航/上传）、审计日志

## 技术栈
- 后端：Go + Gin + SQLite（Clean Architecture / Ports & Adapters）
- 前端：Vue 3 + Vite + Pinia + Ant Design Vue
- 实时：SSE（订单事件等）

## 快速开始（开发）
### 后端
```bash
cd backend
go run ./cmd/server
```
后端配置与安装相关说明见：`backend/README.md:1`

### 前端
```bash
cd frontend
npm install
npm run dev
```

## 自动化平台（轻舟 / qz）与 `qz-override`
本项目的 VPS 自动化对接走 `backend/internal/adapter/automation`，请求头使用 `apikey`（与轻舟面板一致）。

为了获得更完整/更一致的 API 行为，建议使用本仓库的 `qz-override/` 覆盖（或合并）到轻舟系统中，提供一套与本项目适配的接口。

使用说明见：`backend/docs/qz-override.md:1`

## AI Assistant（多智能体协作）
本仓库允许使用 AI assistant 进行项目开发（规划/实现/验证分工协作）。建议将规则与上下文写入并遵循：
- `CLAUDE.md:1`（开发命令、架构与约定）
- `backend/docs/frontend-readme.md:1`（前端对接与状态映射）
- `backend/docs/openapi.yaml:1`（API 合约源）

## 发布与打包
Release 发布后可自动构建并上传 Windows / Linux 压缩包（GitHub Actions）：
- `/.github/workflows/release-build.yml:1`

## License
GPL-3.0，见 `LICENSE:1`

