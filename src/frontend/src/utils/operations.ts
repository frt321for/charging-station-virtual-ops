import type {
  ChargerNode,
  CommandPathType,
  ConnectorNode,
  OperationsSnapshot,
  SessionDetail,
  SessionStatus,
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

export function formatNumber(value: number, digits = 1): string {
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: digits,
    minimumFractionDigits: digits,
  }).format(value)
}

export function formatCompactTime(value?: string): string {
  if (!value) return '--'
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
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
