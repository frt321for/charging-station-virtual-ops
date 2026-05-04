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

`siteId` 支持站点 UUID 或站点编码。`sessionId` 支持会话 UUID 或会话编号。

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
