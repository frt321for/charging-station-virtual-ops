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

## Phase 1 核心接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/health` | 本地后端到云端 PostgreSQL、Valkey、NATS 联通检查 |
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
