# 园区充电桩虚拟运营中台基础设施说明

## 强制约束

- 项目目标是完整平台，不按 MVP 裁剪文档中已明确要求的功能。
- 基础设施先行：先完成云端 PostgreSQL/TimescaleDB、Valkey、NATS JetStream，再开发本地后端和前端。
- 本地开发机是 Windows，本地脚本只使用 PowerShell `.ps1`。
- 本地只跑后端和前端进程，基础设施服务部署在云端。
- 本地后端和前端通过 SSH 隧道连接云端基础设施。
- SSH 入口固定为 `ssh aliyun2738`。
- 后端接口开发阶段可以先用 `curl` 验证 JSON 响应。
- 前端搭建后，所有页面验收必须使用 MCP 浏览器真实操作，不允许用脚本测试替代 UI 验证。
- 项目目录 `02-charging-station-virtual-ops/` 使用独立 Git 仓库。
- 每个关键阶段完成后必须提交一次，提交信息使用约定式提交规范。

## 云端服务器

- 连接命令：`ssh aliyun2738`
- 系统：Ubuntu 24.04.2 LTS
- Docker：已安装
- Docker Compose：已安装
- 规划目录：`/opt/charging-ops`

## 云端服务

云端服务只绑定服务器本机 `127.0.0.1`，不直接公网暴露。

| 服务 | 云端监听 | 本地隧道端口 | 用途 |
| --- | --- | --- | --- |
| PostgreSQL/TimescaleDB | `127.0.0.1:5432` | `localhost:15432` | 事务数据、时序数据 |
| Valkey | `127.0.0.1:6379` | `localhost:16379` | 实时状态、命令超时、短锁 |
| NATS JetStream | `127.0.0.1:4222` | `localhost:14222` | 事件流 |
| NATS Monitor | `127.0.0.1:8222` | `localhost:18222` | NATS 监控接口 |

## 本地开发端口

| 服务 | 本地端口 |
| --- | --- |
| 后端 API | `localhost:8080` |
| 前端 Vite | `localhost:5173` |

## 本地脚本

- `scripts/start-tunnels.ps1`：启动 SSH 隧道。
- `scripts/stop-tunnels.ps1`：停止 SSH 隧道。
- `scripts/tunnels.ps1 status|start|stop|restart`：管理 SSH 隧道完整生命周期。
- `scripts/cloud-infra.ps1 status|start|stop|restart|logs`：管理云端基础设施容器生命周期。
- `scripts/check-ports.ps1`：检查本地开发端口和隧道端口。
- `scripts/check-cloud-infra.ps1`：查看云端容器和服务健康状态。
- `scripts/backend.ps1 status|start|stop|restart`：管理本地 Go 后端生命周期。
- `scripts/frontend.ps1 status|start|stop|restart`：管理本地 React 前端生命周期。
- `scripts/dev-backend.ps1`：`backend.ps1 start` 的兼容入口。
- `scripts/dev-frontend.ps1`：`frontend.ps1 start` 的兼容入口。

## 提交规范

项目完成关键阶段后必须提交，提交前需要完成对应验证。

提交格式：

```text
type(scope): description
```

常用类型：

- `chore`: 基础设施、脚手架、依赖、配置。
- `feat`: 新功能。
- `fix`: 缺陷修复。
- `docs`: 文档。
- `test`: 测试。
- `refactor`: 不改变行为的重构。

阶段示例：

- `chore(infra): add cloud infrastructure and tunnel lifecycle scripts`
- `feat(api): add charger simulator and gateway events`
- `feat(ui): add station operations dashboard`
- `test(regression): add full browser regression checklist`

## LLM 配置

AI 助手使用 OpenAI-compatible API 形态接入。

- Base URL: `https://api-inference.modelscope.cn/v1`
- Model: `deepseek-ai/DeepSeek-V3.2`
- API key 环境变量名：`MODELSCOPE_API_KEY`

真实 API key 是密钥，不允许提交到仓库、不允许写入普通文档、不允许在命令输出中回显。

## 浏览器全量回归要求

前端存在后，功能完成标准必须包括真实浏览器验证：

- 每个可见导航项都要点击。
- 每个按钮都要点击。
- 每个输入框都要输入有效值和必要的异常值。
- 每个下拉、开关、分页、筛选、表格操作、详情入口、弹窗操作都要覆盖。
- 需要检查网络请求、响应状态、请求体、响应体和前端状态变化。
- 需要截图检查布局、遮挡、溢出、加载态、空态和错误态。
- 需要验证鉴权、角色权限、对象级站点权限、越权访问、审计记录。
- 后端 `curl`/自动化接口测试只能作为辅助，不能替代 UI 验收。
