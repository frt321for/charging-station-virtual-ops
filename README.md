# 园区充电桩虚拟运营中台

面向园区站点的充电运营后台，覆盖虚拟桩接入、会话生命周期、站点负载控制、计费核查、故障维保、权限审计和 AI 运维辅助。

## 功能范围

- 站点、区域、桩组、桩机、枪口拓扑与实时运营摘要。
- 虚拟桩注册、心跳、状态、电表读数、告警、离线和命令回执。
- 预约、插枪、远程启动、充电、暂停、恢复、停止、待计费、待核查和已计费状态流转。
- 站点总功率阈值、桩组限功率、排队策略和控制记录。
- 分时价格策略、账单草稿、对账异常、账单修正、确认和导出。
- 故障识别、工单创建、派单流转、复测关闭和 SLA 汇总。
- 登录鉴权、角色权限、站点对象权限、用户/角色管理和审计日志。
- AI 会话异常解释、工单摘要、拥堵风险、站点日报和站点问答。

## 技术栈

- 后端：Go，PostgreSQL/TimescaleDB，Valkey，NATS JetStream。
- 前端：React，TypeScript，Vite，TanStack Query，ECharts。
- 基础设施：云端 Docker Compose，本地通过 SSH 隧道连接。
- 本地脚本：全部使用 PowerShell `.ps1`。

## 目录结构

```text
docs/                 API 与基础设施文档
infra/cloud/          云端 PostgreSQL、Valkey、NATS 编排
prototypes/           前端原型
scripts/              Windows PowerShell 生命周期脚本
src/backend/          Go 后端 API、迁移、模拟器
src/frontend/         React 运营后台
```

## 本地启动

前置条件：Windows PowerShell、Go、pnpm、可用的 `ssh aliyun2738`、云端 Docker 服务。

```powershell
# 1. 启动或检查云端基础设施
.\scripts\cloud-infra.ps1 status
.\scripts\cloud-infra.ps1 start

# 2. 打开本地到云端的 SSH 隧道
.\scripts\tunnels.ps1 start
.\scripts\check-ports.ps1

# 3. 执行数据库迁移
.\scripts\db.ps1 up

# 4. 启动后端和前端
.\scripts\backend.ps1 start
.\scripts\frontend.ps1 start
```

访问地址：

- 前端：`http://127.0.0.1:5173`
- 后端：`http://127.0.0.1:8080/api/v1`

停止本地进程：

```powershell
.\scripts\frontend.ps1 stop
.\scripts\backend.ps1 stop
.\scripts\tunnels.ps1 stop
```

## 演示账号

初始密码均为 `Password2026!`。真实部署前必须修改或删除这些演示账号。

| 用户名 | 角色 | 站点边界 |
| --- | --- | --- |
| `ops.admin` | 运营人员 | 全站点 |
| `station.manager` | 站长 | `HQ-CAMPUS` |
| `maintenance.tech` | 运维人员 | `HQ-CAMPUS` |
| `finance.reviewer` | 财务人员 | `HQ-CAMPUS` |
| `support.agent` | 客服人员 | `HQ-CAMPUS` |
| `ai.assistant` | AI 运维助手 | `HQ-CAMPUS` |

## AI 配置

AI 接入使用 OpenAI-compatible API。不要把真实密钥提交到仓库。

```text
MODELSCOPE_BASE_URL=https://api-inference.modelscope.cn/v1
MODELSCOPE_MODEL=deepseek-ai/DeepSeek-V3.2
MODELSCOPE_API_KEY=<your-modelscope-api-key>
```

可放入本地忽略文件 `.env.local`，或验证时临时传参：

```powershell
.\scripts\verify-ai.ps1 -ModelScopeApiKey "<your-modelscope-api-key>" -RestartBackend -ExpectProvider modelscope
```

未配置密钥或模型调用失败时，后端会返回 `local-fallback` 的确定性运营摘要，接口仍保持只读。

## 验证命令

```powershell
# 后端单元测试
Set-Location .\src\backend
go test ./...
Set-Location ..\..

# 前端静态检查与构建
Set-Location .\src\frontend
pnpm lint
pnpm build
Set-Location ..\..

# 平台接口闭环
.\scripts\test-flow.ps1

# AI 真实模型验证
.\scripts\verify-ai.ps1 -ExpectProvider modelscope
```

前端功能验收以真实浏览器操作为准，脚本只作为接口和回归辅助。

## 常用脚本

| 脚本 | 用途 |
| --- | --- |
| `scripts\cloud-infra.ps1 status|start|stop|restart|logs` | 云端基础设施容器管理 |
| `scripts\tunnels.ps1 status|start|stop|restart` | SSH 隧道生命周期管理 |
| `scripts\db.ps1 status|up|down` | 数据库迁移管理 |
| `scripts\backend.ps1 status|start|stop|restart` | 本地 Go 后端管理 |
| `scripts\frontend.ps1 status|start|stop|restart` | 本地 React 前端管理 |
| `scripts\simulator.ps1 status|once|start|stop|restart` | 虚拟桩模拟器管理 |
| `scripts\test-flow.ps1` | 平台主流程接口回归 |
| `scripts\verify-ai.ps1` | AI provider 与模型调用验证 |

## 文档

- [基础设施说明](docs/infrastructure.md)
- [后端 API 说明](docs/api.md)

## 提交规范

提交信息采用约定式提交：

```text
type(scope): description
```

常用类型：`feat`、`fix`、`docs`、`test`、`chore`、`refactor`。
