# qz-override（轻舟面板 API 覆盖）

本项目后端的自动化对接默认以“轻舟（qz）”的接口形态为基准（`apikey` 请求头 + 若干历史遗留参数名兼容），用于：
- 获取产品/线路/镜像等基础数据
- 创建/查询/操作 VPS（开关机、重装、续费、快照/备份、端口映射等）

但不同轻舟版本/插件生态里接口行为容易不一致。为保证后端 `backend/internal/adapter/automation` 能“全面可用”，仓库提供了 `qz-override/`：把关键接口整理成一套更稳定的实现。

## 它是什么
- `qz-override/app/api/controller/Cloud.php`：轻舟 API 控制器（建议覆盖/合并到轻舟项目同路径）
- `qz-override/cloud.yml`：对应接口的 OpenAPI（便于对齐字段与调试）

## 如何接入（推荐做法）
1) 在你的轻舟面板项目里找到对应路径（ThinkPHP 项目结构一般一致）：
   - `app/api/controller/Cloud.php`
2) 用本仓库的 `qz-override/app/api/controller/Cloud.php` 覆盖或合并进去
3) 确认轻舟侧的 `web.apikey`（或同等配置）已设置
   - 该 controller 会校验请求头 `apikey`（除 `panel` 动作外）
4) 在本项目后端配置自动化地址与 key（示例）：
   - `automation.base_url: "https://<your-qz-host>/index.php/api/cloud"`
   - `automation.api_key: "<your-apikey>"`

后端配置支持 `app.config.yaml`（见 `backend/README.md:1`）。

## 后端会怎么调用
后端客户端位于：
- `backend/internal/adapter/automation/client.go:1`

关键点：
- Header：`apikey: <AUTOMATION_API_KEY>`
- BaseURL：拼接诸如 `/create_host`、`/hostinfo` 等端点（因此 BaseURL 需要指向 `.../api/cloud`）

## 常见问题
### 1) 从后端调用返回 apikey 错误
- 确认轻舟侧 `web.apikey` 与后端 `AUTOMATION_API_KEY`（或 YAML 的 `automation.api_key`）一致
- 确认请求确实带了 header `apikey`（后端默认会带）

### 2) BaseURL 填错（404 或接口不生效）
- 正确示例：`https://host/index.php/api/cloud`
- 错误示例：只填到 `https://host/` 或漏掉 `index.php/api/cloud`

