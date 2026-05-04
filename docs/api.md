# 后端 API 说明

## 响应格式

所有接口统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": "2026-05-04T00:00:00Z",
  "traceId": "trace-id"
}
```

## 认证与权限

除 `/api/v1/health`、`/api/v1/meta`、`/api/v1/auth/login` 和虚拟桩网关上报接口外，运营后台接口都需要登录。登录后前端可以使用响应里的 `token` 发送：

```http
Authorization: Bearer <token>
```

后端同时写入 `HttpOnly` 会话 Cookie，便于同源浏览器请求。

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/login` | public | 用户登录，返回 token、用户、角色、权限和站点边界 |
| `GET` | `/api/v1/auth/me` | login | 查询当前登录用户 |
| `POST` | `/api/v1/auth/logout` | login | 注销当前会话 |
| `GET` | `/api/v1/users` | `users:manage` | 查询用户、角色、权限和站点授权 |
| `GET` | `/api/v1/roles` | `users:manage` | 查询角色及权限清单 |
| `GET` | `/api/v1/audit-logs` | `audit:read` | 查询审计日志 |
| `GET` | `/api/v1/config/sites/{siteId}` | `config:manage` | 查询站点配置快照 |
| `PATCH` | `/api/v1/config/sites/{siteId}` | `config:manage` | 更新站点、容量、负载边界和 SLA |
| `PATCH` | `/api/v1/config/areas/{areaId}` | `config:manage` | 更新区域配置 |
| `PATCH` | `/api/v1/config/charger-groups/{groupId}` | `config:manage` | 更新桩组配置 |
| `PATCH` | `/api/v1/config/chargers/{chargerId}` | `config:manage` | 更新桩机配置 |
| `PATCH` | `/api/v1/config/connectors/{connectorId}` | `config:manage` | 更新枪口配置 |
| `PATCH` | `/api/v1/config/load-policies/{policyId}` | `config:manage` | 更新负载策略 |
| `PATCH` | `/api/v1/config/reservation-rules/{ruleId}` | `config:manage` | 更新预约规则 |
| `PATCH` | `/api/v1/config/queue-rules/{ruleId}` | `config:manage` | 更新排队规则 |
| `POST` | `/api/v1/config/pricing-policies` | `config:manage` | 新建分时价格版本 |

未登录返回 HTTP `401`、业务码 `20001`。权限不足返回 HTTP `403`、业务码 `20002`。

本地演示迁移会创建以下账号，初始密码均为 `Password2026!`；真实部署前必须修改或删除这些演示账号。

| 用户名 | 角色 | 边界 |
| --- | --- | --- |
| `ops.admin` | 运营人员 | 全站点 |
| `station.manager` | 站长 | `HQ-CAMPUS` |
| `maintenance.tech` | 运维人员 | `HQ-CAMPUS` |
| `finance.reviewer` | 财务人员 | `HQ-CAMPUS` |
| `support.agent` | 客服人员 | `HQ-CAMPUS` |
| `ai.assistant` | AI 运维助手 | `HQ-CAMPUS` |

核心权限边界：

| 角色 | 允许 | 禁止 |
| --- | --- | --- |
| 站长 | 站点、会话、远程命令、负载控制、维保读取、审计读取 | 用户角色管理 |
| 运维人员 | 维保工单处理、站点/会话读取 | 财务核查、账单修正、远程控制 |
| 财务人员 | 账单生成、核查、修正、确认、导出 | 原始读数修改、远程控制、维保状态流转 |
| 客服人员 | 站点、会话、账单、维保读取 | 远程控制、财务修正、工单流转 |
| AI 运维助手 | 只读解释、摘要、报告草稿 | 所有写操作 |

## Phase 1 核心接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/health` | 本地后端到云端 PostgreSQL、Valkey、NATS 联通检查 |
| `GET` | `/api/v1/meta` | 平台模块元信息 |
| `GET` | `/api/v1/sites` | 查询站点运营摘要 |
| `GET` | `/api/v1/sites/{siteId}/topology` | 查询站点、区域、桩组、桩机、枪口拓扑 |
| `POST` | `/api/v1/reservations` | 创建预约会话，枪口进入预约状态 |
| `GET` | `/api/v1/sessions` | 查询最近充电会话 |
| `GET` | `/api/v1/sessions/{sessionId}` | 查询会话详情、事件时间线、电表采样 |
| `POST` | `/api/v1/sessions/{sessionId}/transition` | 按状态机推进会话状态并写入事件 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/start` | 下发远程启动命令并写命令记录 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/stop` | 下发远程停止命令并写命令记录 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/pause` | 下发远程暂停命令并写命令记录 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/resume` | 下发远程恢复命令并写命令记录 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/reset` | 下发远程复位命令并写命令记录 |
| `POST` | `/api/v1/sessions/{sessionId}/commands/limit-power` | 下发限功率命令并写命令记录 |
| `POST` | `/api/v1/gateway/chargers/register` | 虚拟桩注册 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/heartbeat` | 虚拟桩心跳 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/status` | 虚拟桩与枪口状态上报 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/meter-values` | 虚拟桩电表读数上报 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/alarms` | 虚拟桩故障告警上报 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/command-receipts` | 虚拟桩命令回执 |
| `POST` | `/api/v1/gateway/chargers/{chargerCode}/offline` | 虚拟桩离线 |
| `GET` | `/api/v1/simulator/status` | 查询浏览器控制的虚拟桩模拟器状态 |
| `POST` | `/api/v1/simulator/once` | 运行一次虚拟桩闭环场景 |
| `POST` | `/api/v1/simulator/start` | 启动持续心跳/告警模拟 |
| `POST` | `/api/v1/simulator/stop` | 停止持续模拟 |

`siteId` 支持站点 UUID 或站点编码。`sessionId` 支持会话 UUID 或会话编号。

## Phase 2 计费、排队与负载控制接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/pricing-policies?siteId=HQ-CAMPUS` | 查询站点定价策略及分时价格版本 |
| `GET` | `/api/v1/billing-drafts?siteId=HQ-CAMPUS` | 查询最近账单草稿 |
| `POST` | `/api/v1/sessions/{sessionId}/billing-draft` | 根据会话读数、时长和当前定价策略生成账单草稿 |
| `GET` | `/api/v1/reconciliation-exceptions?siteId=HQ-CAMPUS` | 查询能量、时长、金额、停止原因异常核查队列 |
| `GET` | `/api/v1/sites/{siteId}/load-control` | 查询站点负载策略、当前负载、队列和控制记录 |
| `POST` | `/api/v1/load-control/records` | 写入人工负载控制记录 |

## Phase 3 故障维保与财务复核接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/maintenance?siteId=HQ-CAMPUS` | 查询故障队列、工单列表和 SLA 汇总 |
| `POST` | `/api/v1/faults/{faultId}/work-order` | 从故障创建或关联维保工单 |
| `GET` | `/api/v1/work-orders/{workOrderId}/events` | 查询工单流转事件 |
| `POST` | `/api/v1/work-orders/{workOrderId}/transition` | 推进工单状态并写审计记录 |
| `POST` | `/api/v1/reconciliation-exceptions/{exceptionId}/review` | 更新核查异常状态 |
| `POST` | `/api/v1/reconciliation-exceptions/{exceptionId}/corrections` | 写入账单修正记录，不改写原始读数 |
| `POST` | `/api/v1/billing-drafts/{billId}/confirm` | 确认账单并关闭关联核查异常 |
| `GET` | `/api/v1/reconciliation-exceptions/export?siteId=HQ-CAMPUS&generatedBy=finance-reviewer` | 导出核查异常 CSV 包并写审计 |

## Phase 4 AI 运维助手接口

所有 AI 接口都需要 `ai:read` 权限，只读取站点、会话、工单、账单、核查和事件上下文，不写业务状态，不下发远程命令。后端按登录用户的站点对象权限过滤；AI 运维助手只能访问已授权站点。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/ai/session-explanations` | 生成异常会话解释，包含事件链、命令回执、读数和核查依据 |
| `POST` | `/api/v1/ai/work-order-summaries` | 生成工单摘要，包含故障对象、SLA 节点和流转事件 |
| `POST` | `/api/v1/ai/congestion-risk` | 生成站点拥堵风险说明，基于负载、排队、故障和可用枪口 |
| `POST` | `/api/v1/ai/daily-reports` | 生成站点日报草稿，汇总站点运营、收入电量、工单和核查 |
| `POST` | `/api/v1/ai/station-qa` | 在站点权限范围内回答运营问题 |

## AI 请求示例

会话异常解释：

```json
{
  "sessionId": "CS-20260504000000.000000000"
}
```

工单摘要：

```json
{
  "workOrderId": "WO-SEED-HQ-001"
}
```

拥堵风险：

```json
{
  "siteId": "HQ-CAMPUS",
  "horizonHours": 4
}
```

站点日报：

```json
{
  "siteId": "HQ-CAMPUS",
  "businessDate": "2026-05-04"
}
```

站点问答：

```json
{
  "siteId": "HQ-CAMPUS",
  "question": "当前站点最大的运营风险是什么"
}
```

AI 响应中的 `provider` 为 `modelscope` 时表示调用了配置的 OpenAI 兼容模型；未配置密钥或模型调用失败时，后端会返回 `local-fallback` 的确定性运营摘要，接口仍保持只读。

## 审计查询请求

```text
GET /api/v1/audit-logs?page=1&pageSize=20&entityType=work_order&action=transition
```

`entityType`、`action`、`actorName` 都是可选过滤条件。`entityType` 在审计页作为“对象”关键字使用，会对 `entity_type` 和 `entity_id` 做大小写不敏感的模糊匹配；`action` 精确匹配动作；`actorName` 模糊匹配操作人。审计日志覆盖登录/注销、财务核查、账单修正、账单确认、对账导出、工单创建和工单流转。远程控制目前记录在 `remote_commands`、`session_events`、`charger_events` 中，后续审计页会合并展示命令链路和通用审计日志。

## 配置管理请求

配置快照覆盖站点、区域、桩组、桩机、枪口、价格版本、负载策略、预约规则和排队规则：

```text
GET /api/v1/config/sites/HQ-CAMPUS
```

站点配置更新：

```json
{
  "name": "总部园区充电站",
  "capacityKw": 120,
  "loadLimitKw": 96,
  "slaResponseMinutes": 15,
  "slaRecoveryMinutes": 120,
  "status": "active"
}
```

新建价格版本：

```json
{
  "siteId": "HQ-CAMPUS",
  "code": "HQ-STD",
  "name": "总部园区标准计价 V2",
  "chargerType": "all",
  "status": "active",
  "periods": [
    {
      "label": "平段",
      "startMinute": 0,
      "endMinute": 420,
      "energyPricePerKwh": 0.78,
      "serviceFeePerKwh": 0.22,
      "occupancyFeePerMinute": 0.02
    }
  ]
}
```

## 会话状态流转

当前后端强制校验以下状态流转：

```text
reserved -> waiting_arrival | cancelled
waiting_arrival -> plugged_in | cancelled
plugged_in -> starting | cancelled
starting -> charging | pending_review | cancelled
charging -> paused | stopping
paused -> charging | stopping
stopping -> pending_billing | pending_review
pending_billing -> billed | pending_review
pending_review -> billed | cancelled
```

非法流转返回 `400`，业务错误码 `30011`。

## 状态流转请求

```json
{
  "targetStatus": "paused",
  "eventType": "LoadLimitApplied",
  "source": "load-control",
  "payload": {
    "reason": "site-load-limit",
    "limitKw": 3.5
  }
}
```

`eventType` 默认 `StatusTransitioned`，`source` 默认 `operations`，`payload` 默认 `{}`。

## 预约请求

```json
{
  "connectorCode": "AC-N-001-02",
  "reservationMinutes": 30,
  "requestedBy": "station-manager",
  "payload": {
    "source": "operations"
  }
}
```

## 虚拟桩注册请求

```json
{
  "groupCode": "N-A1",
  "code": "AC-N-002",
  "name": "北区 2 号交流桩",
  "chargerType": "ac",
  "ratedPowerKw": 14,
  "connectorCount": 2,
  "connectorMaxPowerKw": 7,
  "installationLocation": "北区车棚 N-02"
}
```

## 虚拟桩心跳请求

```json
{
  "status": "available",
  "payload": {
    "chargerTime": "2026-05-04T00:00:00Z"
  }
}
```

## 虚拟桩状态请求

```json
{
  "status": "available",
  "sessionNo": "CS-20260504000000.000000000",
  "connectors": [
    {
      "code": "AC-N-002-01",
      "status": "plugged"
    }
  ],
  "payload": {
    "source": "virtual-charger"
  }
}
```

`status=plugged` 且带 `sessionNo` 时，会把预约会话推进到 `plugged_in` 并写入插枪事件。

## 虚拟桩读数请求

```json
{
  "connectorCode": "AC-N-002-01",
  "sessionNo": "CS-20260504000000.000000000",
  "powerKw": 6.4,
  "voltageV": 226,
  "currentA": 28.3,
  "meterKwh": 1250.5,
  "payload": {
    "source": "virtual-charger"
  }
}
```

## 虚拟桩告警请求

```json
{
  "connectorCode": "AC-N-002-01",
  "sessionNo": "CS-20260504000000.000000000",
  "faultCode": "LOW_VOLTAGE",
  "severity": "low",
  "payload": {
    "source": "virtual-charger"
  }
}
```

## 远程命令请求

启动、停止、暂停、恢复、复位请求体：

```json
{
  "requestedBy": "station-manager",
  "payload": {
    "reason": "manual-operation"
  }
}
```

限功率请求体：

```json
{
  "requestedBy": "load-control",
  "targetPowerKw": 3.5,
  "payload": {
    "reason": "site-load-limit"
  }
}
```

命令状态约束：

| 命令 | 会话状态要求 | 下发后会话状态 |
| --- | --- | --- |
| `start` | `plugged_in` | `starting` |
| `stop` | `charging` 或 `paused` | `stopping` |
| `pause` | `charging` | 不立即改变，接受回执后变为 `paused` |
| `resume` | `paused` | 不立即改变，接受回执后变为 `charging` |
| `reset` | 非 `billed`、非 `cancelled` | 不立即改变 |
| `limit-power` | `charging` 且必须提供 `targetPowerKw` | 不立即改变 |

桩机为 `offline` 或 `disabled` 时，远程命令返回 `409`，业务错误码 `40003`。

## 命令回执请求

```json
{
  "commandNo": "RC-20260504000000.000000000",
  "sessionNo": "CS-20260504000000.000000000",
  "commandType": "start",
  "receipt": "accepted",
  "message": "accepted by virtual charger",
  "payload": {
    "chargerTime": "2026-05-04T00:00:00Z"
  }
}
```

命令回执中的 `commandType` 必须与原命令记录一致，否则返回 `404`，业务错误码 `30001`。

## 模拟器控制请求

`/api/v1/simulator/once` 和 `/api/v1/simulator/start` 请求体：

```json
{
  "chargerCount": 5,
  "connectorCount": 1,
  "onlineRate": 0.8,
  "faultRate": 0.05,
  "loadCurve": "commute"
}
```

`loadCurve` 支持 `commute`、`flat`、`random`。`onlineRate` 和 `faultRate` 范围为 `0` 到 `1`。

## 账单草稿生成请求

```json
{
  "generatedBy": "finance-reviewer"
}
```

会话状态必须是 `pending_billing`、`pending_review` 或 `billed`。后端会优先使用会话起止电表读数；如果会话起止读数为空，则使用时序读数的首末值。电量、时长、金额或停止原因异常会同步写入核查队列。

## 负载控制记录请求

```json
{
  "siteId": "HQ-CAMPUS",
  "sessionId": "CS-20260504000000.000000000",
  "actionType": "limit_power",
  "reason": "site-load-limit",
  "beforeLoadKw": 62.4,
  "afterLoadKw": 57.1,
  "targetPowerKw": 3.5,
  "status": "applied",
  "operatorName": "station-manager"
}
```

`actionType` 支持 `limit_power`、`pause`、`resume`、`queue`、`reject`、`promote`、`release`。`status` 支持 `recommended`、`sent`、`applied`、`rejected`。`sessionId` 可为空；为空时记录站点级控制决策。

## 工单创建请求

```json
{
  "assigneeName": "maintenance-shift-a",
  "impactScope": "connector",
  "title": "直流桩 2 号枪不可用",
  "description": "CONNECTOR_UNAVAILABLE",
  "actorName": "maintenance"
}
```

`faultId` 支持故障 UUID 或故障编号。已有工单时接口返回已关联工单。

## 工单流转请求

```json
{
  "targetStatus": "accepted",
  "assigneeName": "maintenance-shift-a",
  "actorName": "maintenance",
  "note": "现场确认",
  "payload": {
    "source": "operations-console"
  }
}
```

工单状态流转约束：

```text
open -> assigned | cancelled
assigned -> accepted | cancelled
accepted -> arrived | cancelled
arrived -> handling | cancelled
handling -> retest | cancelled
retest -> recovered | handling | cancelled
recovered -> closed
```

## 核查复核请求

```json
{
  "status": "reviewing",
  "reviewerName": "finance-reviewer",
  "note": "账单复核"
}
```

`status` 支持 `open`、`reviewing`、`resolved`。

## 账单修正请求

```json
{
  "correctedEnergyKwh": 12.45,
  "correctedDurationMinutes": 86,
  "correctedTotalAmount": 22.8,
  "reason": "人工复核修正",
  "reviewerName": "finance-reviewer"
}
```

修正记录只进入 `billing_corrections` 和审计日志，不更新原始电表读数。

## 账单确认请求

```json
{
  "reviewerName": "finance-reviewer",
  "note": "账单确认"
}
```

确认后账单状态为 `confirmed`，关联核查异常为 `resolved`，会话状态回到 `billed`。
