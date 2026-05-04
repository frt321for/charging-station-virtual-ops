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
| `GET` | `/api/v1/sessions` | 查询最近充电会话 |
| `GET` | `/api/v1/sessions/{sessionId}` | 查询会话详情、事件时间线、电表采样 |
| `POST` | `/api/v1/sessions/{sessionId}/transition` | 按状态机推进会话状态并写入事件 |

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
