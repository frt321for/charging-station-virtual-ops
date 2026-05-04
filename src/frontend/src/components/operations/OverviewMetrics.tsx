import type { OperationsSnapshot } from '../../types/operations'
import type { ReactNode } from 'react'
import { currentLoadKw, formatNumber } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface OverviewMetricsProps {
  snapshot?: OperationsSnapshot
  isLoading: boolean
}

export function OverviewMetrics({ snapshot, isLoading }: OverviewMetricsProps) {
  const site = snapshot?.site
  const currentLoad = currentLoadKw(snapshot)
  const loadLimit = site?.loadLimitKw ?? 0
  const loadRatio = loadLimit > 0 ? Math.min(100, (currentLoad / loadLimit) * 100) : 0
  const reviewCount = snapshot?.sessions.filter((session) => session.status === 'pending_review').length ?? 0
  const queueCount =
    snapshot?.sessions.filter((session) => session.status === 'reserved' || session.status === 'waiting_arrival')
      .length ?? 0

  if (isLoading) {
    return (
      <section className="ops-hero" aria-label="运营总览加载中">
        <div className="ops-load-card ops-skeleton" />
        <div className="ops-metric-grid">
          <div className="ops-metric ops-skeleton" />
          <div className="ops-metric ops-skeleton" />
          <div className="ops-metric ops-skeleton" />
          <div className="ops-metric ops-skeleton" />
        </div>
      </section>
    )
  }

  return (
    <section className="ops-hero" aria-label="运营总览">
      <article className="ops-load-card">
        <h1>{site?.campus ?? '园区'}实时负载</h1>
        <div className="ops-load-value">
          <strong>{formatNumber(currentLoad)}</strong>
          <span>kW</span>
        </div>
        <div className="ops-capacity">
          <div className="ops-bar-label">
            <span>站点容量 {formatNumber(loadLimit)} kW</span>
            <span>余量 {formatNumber(Math.max(0, loadLimit - currentLoad))} kW</span>
          </div>
          <div className="ops-bar" aria-label={`当前负载 ${formatNumber(loadRatio)}%`}>
            <span style={{ width: `${loadRatio}%` }} />
          </div>
        </div>
      </article>

      <div className="ops-metric-grid">
        <Metric title="可用枪口" value={site?.availableConnectors ?? 0} footer={`总枪口 ${site?.connectorCount ?? 0}`}>
          <StatusBadge tone="ok">正常</StatusBadge>
        </Metric>
        <Metric title="充电会话" value={site?.activeSessions ?? 0} footer={`桩机 ${site?.chargerCount ?? 0}`}>
          <StatusBadge tone="charge">运行</StatusBadge>
        </Metric>
        <Metric title="排队" value={queueCount} footer="预约 / 等待到场">
          <StatusBadge tone="warn">高峰</StatusBadge>
        </Metric>
        <Metric title="待核查" value={reviewCount} footer={`故障桩 ${site?.faultedChargers ?? 0}`}>
          <StatusBadge tone={reviewCount > 0 ? 'danger' : 'ok'}>{reviewCount > 0 ? '异常' : '正常'}</StatusBadge>
        </Metric>
      </div>
    </section>
  )
}

interface MetricProps {
  title: string
  value: number
  footer: string
  children: ReactNode
}

function Metric({ title, value, footer, children }: MetricProps) {
  return (
    <article className="ops-metric">
      <label>
        {title}
        {children}
      </label>
      <data>{value}</data>
      <footer>{footer}</footer>
    </article>
  )
}
