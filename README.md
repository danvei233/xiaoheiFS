# xiaoheiFS (小黑云财务)

一个自托管财务系统，包含用户端、管理员端、插件机制与探针能力。

## 1. 开发阶段
- 当前阶段：`Alpha`
- 现状：核心功能链路可跑通，但仍有明显的稳定性和安全性缺口。
- 建议：仅用于测试、内测或低风险试运行环境。

## 2. 系统功能（当前可用）

### 2.1 用户侧
- 注册、登录、找回密码（基础实现）
- 商品浏览、购物车、下单、订单查看
- VPS 相关管理页（按后端能力启用）
- 钱包相关页（充值/记录等）
- 工单与通知相关页面

### 2.2 管理后台
- 用户与订单管理
- 套餐、计费周期、商品类型配置
- 站点与系统参数配置
- 审计、调试、计划任务等运维页面
- 插件管理、自动化对接页面

### 2.3 插件能力
- 基于 `backend/plugins` 目录管理插件内容
- 支持在后台启用/禁用配置插件
- 可配合“轻舟自动化插件”完成商品与自动化联动

### 2.4 探针能力
- 独立项目：`pingbot`
- 可单独构建与部署
- 工作流支持多平台产物发布

## 3. 仓库目录说明（按项目划分）
- `.github/`：CI/CD 工作流（构建、打包、发布）
- `frontend/`：Web 前端项目（用户端 + 管理端）
- `backend/`：Go 后端主服务（API、业务、插件管理）
- `app/`：Flutter 客户端项目
  - `app/xiaoheifs_app`：管理端 App
  - `app/xiaoheifs_userapp`：用户端 App
- `pingbot/`：探针项目（独立 Go 服务）
- `script/`：本地与 CI 共用构建脚本
- `docs/`：部署与运维文档
- `build/`：本地构建输出目录（ignore）
- `dist/`：发布打包输出目录（ignore）
- `tmp/`：临时目录（ignore）
- `qz-override/`：轻舟相关覆盖资源
- `automation/`：自动化相关模块/资源

## 4. 最小部署方案（你现在就能跑起来）

### 4.1 准备
- 一台 Linux/Windows 服务器
- 一个 MySQL 实例（建议 8.0+）
- 开放端口 `8080`

### 4.2 安装
1. 从 Release 下载最新主系统包（示例命名：`xiaohei-<tag>-linux-amd64.tar.gz` 或 `xiaohei-<tag>-windows-amd64.zip`）。
2. 解压到运行目录，例如 `/opt/xiaohei` 或 `D:\xiaohei`。
3. 直接启动服务（如需 systemd，可自行托管为系统服务）。
4. 打开 `http://<IP>:8080/`，系统会跳转安装页。
5. 填写数据库连接信息并完成安装。
6. 打开 `http://<IP>:8080/admin/login` 登录管理员后台。

### 4.3 初始化业务配置（最少步骤）
1. 进入 `基础设置 -> 插件设置`，启用轻舟自动化插件。
2. 进入商品配置，新增商品类型并执行同步。
3. 新增套餐。
4. 添加计费周期。
5. 用测试账号走一遍下单链路，确认系统可用。

### 4.4 验收检查
- 首页可打开，安装状态正常
- 后台可登录
- 商品类型同步成功
- 套餐和计费周期可正常保存
- 下单流程无 5xx 错误

## 5. 编译教程

### 5.1 一键脚本（推荐）

Linux:
```bash
./script/build-linux.sh
```
输出：`build/linux/`

Windows:
```bat
script\build-win.bat
```
输出：`build/windows/`

### 5.2 手动编译（排障用）
1. 构建前端
```bash
cd frontend
npm ci
npm run build
```

2. 构建后端
```bash
cd backend
go build -o ../build/linux/server ./cmd/server
```

3. 拷贝静态资源
```bash
mkdir -p build/linux/static
cp -a frontend/dist/. build/linux/static/
```

### 5.3 发布打包（CI）
- 触发 `release-build.yml` 后会产出平台包与校验文件。
- 产物会上传到 Release Assets。

## 6. 文档索引
- App 部署教程：`docs/app-deploy.md`
- 探针部署教程：`docs/probe-deploy.md`

## 7. 已知问题（Alpha）
- 支付实名插件流程未完整测试。
- 注册/登录链路安全性与合规性不足，健壮性需要增强。
- 部分边界场景的异常处理与审计闭环仍需完善。

## 8. 近期建议优先级
1. 完整跑通并补齐支付实名插件端到端测试。
2. 重构注册登录安全机制（限流、风控、会话策略、合规策略）。
3. 补全关键链路回归测试（下单、支付、插件同步、退款、工单）。
4. 增加部署自检脚本与健康检查文档。
