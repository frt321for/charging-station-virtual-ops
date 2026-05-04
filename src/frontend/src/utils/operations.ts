import type {
  ChargerNode,
  CommandPathType,
  ConnectorNode,
  FaultSeverity,
  FaultStatus,
  OperationsSnapshot,
  LoadActionType,
  LoadRecordStatus,
  ReconciliationException,
  SessionDetail,
  SessionStatus,
  WorkOrderStatus,
} from '../types/operations'

export const sessionStatusLabels: Record<SessionStatus, string> = {
  reserved: '已预约',
  waiting_arrival: '等待到场',
  plugged_in: '已插枪',
  starting: '启动中',
  charging: '充电中',
  paused: '暂停中',
  stopping: '结束中',
  pending_billing: '待计费',
  billed: '已计费',
  pending_review: '待核查',
  cancelled: '已取消',
}

export const loadActionLabels: Record<LoadActionType, string> = {
  limit_power: '限功率',
  pause: '暂停',
  resume: '恢复',
  queue: '排队',
  reject: '拒绝',
  promote: '晋级',
  release: '释放',
}

export const loadRecordStatusLabels: Record<LoadRecordStatus, string> = {
  recommended: '建议',
  sent: '已下发',
  applied: '已生效',
  rejected: '已拒绝',
}

export const billingStatusLabels = {
  draft: '草稿',
  confirmed: '已确认',
  pending_review: '待核查',
} as const

export const exceptionSeverityLabels = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重',
} as const

export const reconciliationStatusLabels: Record<ReconciliationException['status'], string> = {
  open: '待处理',
  reviewing: '复核中',
  resolved: '已解决',
}

export const faultStatusLabels: Record<FaultStatus, string> = {
  open: '待派单',
  linked_work_order: '已派单',
  resolved: '已恢复',
}

export const workOrderStatusLabels: Record<WorkOrderStatus, string> = {
  open: '待分派',
  assigned: '已分派',
  accepted: '已接单',
  arrived: '已到场',
  handling: '处理中',
  retest: '复测中',
  recovered: '已恢复',
  closed: '已关闭',
  cancelled: '已取消',
}

export function formatNumber(value: number, digits = 1): string {
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: digits,
    minimumFractionDigits: digits,
  }).format(value)
}

export function formatMoney(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    currency: 'CNY',
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
    style: 'currency',
  }).format(value)
}

export function formatCompactTime(value?: string): string {
  if (!value) return '--'
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

export function formatMinuteRange(startMinute: number, endMinute: number): string {
  return `${formatMinute(startMinute)}-${formatMinute(endMinute)}`
}

function formatMinute(value: number): string {
  const hour = Math.floor(value / 60)
  const minute = value % 60
  return `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`
}

export function latestMeterPower(detail?: SessionDetail): number {
  const latest = detail?.meterValues.at(-1)
  return latest?.powerKw ?? 0
}

export function sessionEnergy(detail?: SessionDetail): number | undefined {
  if (!detail) return undefined
  const latest = detail.meterValues.at(-1)
  if (!latest || detail.meterStartKwh === undefined) return undefined
  return Math.max(0, latest.meterKwh - detail.meterStartKwh)
}

export function currentLoadKw(snapshot?: OperationsSnapshot): number {
  if (!snapshot) return 0
  return snapshot.sessionDetails
    .filter((detail) => detail.status === 'charging' || detail.status === 'starting')
    .reduce((total, detail) => total + latestMeterPower(detail), 0)
}

export function flattenChargers(snapshot?: OperationsSnapshot): ChargerNode[] {
  if (!snapshot) return []
  return snapshot.topology.areas.flatMap((area) =>
    area.groups.flatMap((group) => group.chargers),
  )
}

export function connectorStatusTone(status: string): 'ok' | 'charge' | 'warn' | 'danger' | 'idle' {
  if (status === 'available') return 'ok'
  if (status === 'charging' || status === 'plugged') return 'charge'
  if (status === 'reserved') return 'warn'
  if (status === 'faulted' || status === 'offline' || status === 'unavailable') return 'danger'
  return 'idle'
}

export function sessionStatusTone(status: SessionStatus): 'ok' | 'charge' | 'warn' | 'danger' | 'idle' {
  if (status === 'charging' || status === 'starting') return 'charge'
  if (status === 'pending_billing' || status === 'reserved' || status === 'waiting_arrival') return 'warn'
  if (status === 'pending_review' || status === 'cancelled') return 'danger'
  if (status === 'billed') return 'ok'
  return 'idle'
}

export function severityTone(severity: FaultSeverity): 'ok' | 'charge' | 'warn' | 'danger' | 'idle' {
  if (severity === 'critical' || severity === 'high') return 'danger'
  if (severity === 'medium') return 'warn'
  return 'idle'
}

export function workOrderStatusTone(status: WorkOrderStatus): 'ok' | 'charge' | 'warn' | 'danger' | 'idle' {
  if (status === 'closed' || status === 'recovered') return 'ok'
  if (status === 'cancelled') return 'danger'
  if (status === 'handling' || status === 'retest') return 'charge'
  if (status === 'open' || status === 'assigned') return 'warn'
  return 'idle'
}

export function canRunCommand(status: SessionStatus | undefined, commandType: CommandPathType): boolean {
  if (!status) return false
  if (commandType === 'start') return status === 'plugged_in'
  if (commandType === 'stop') return status === 'charging' || status === 'paused'
  if (commandType === 'pause') return status === 'charging'
  if (commandType === 'resume') return status === 'paused'
  if (commandType === 'reset') return status !== 'billed' && status !== 'cancelled'
  if (commandType === 'limit-power') return status === 'charging'
  return false
}

export function findConnectorByCode(
  chargers: ChargerNode[],
  connectorCode?: string,
): { charger: ChargerNode; connector: ConnectorNode } | undefined {
  if (!connectorCode) return undefined

  for (const charger of chargers) {
    const connector = charger.connectors.find((item) => item.code === connectorCode)
    if (connector) return { charger, connector }
  }

  return undefined
}
