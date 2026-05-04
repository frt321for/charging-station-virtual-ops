import type { BillingDraft, SessionDetail } from '../../types/operations'
import {
  billingStatusLabels,
  formatCompactTime,
  formatMoney,
  formatNumber,
  sessionEnergy,
  sessionStatusLabels,
  sessionStatusTone,
} from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface SessionDetailPanelProps {
  session?: SessionDetail
  drafts: BillingDraft[]
  isFetching: boolean
}

export function SessionDetailPanel({ session, drafts, isFetching }: SessionDetailPanelProps) {
  const sessionDrafts = session ? drafts.filter((draft) => draft.sessionNo === session.sessionNo) : []
  const latestMeter = session?.meterValues.at(-1)
  const events = session?.events.slice().reverse() ?? []

  return (
    <section className="ops-panel">
      <header>
        <h2>会话详情</h2>
        <StatusBadge tone={session ? sessionStatusTone(session.status) : 'idle'}>
          {session ? sessionStatusLabels[session.status] : '未选择'}
        </StatusBadge>
      </header>

      {session ? (
        <div className="ops-session-detail">
          <div className="ops-detail-grid">
            <div>
              <span>会话</span>
              <strong>{session.sessionNo}</strong>
            </div>
            <div>
              <span>枪口</span>
              <strong>{session.connectorCode}</strong>
            </div>
            <div>
              <span>电量</span>
              <strong>{formatDetailEnergy(session)}</strong>
            </div>
            <div>
              <span>停止原因</span>
              <strong>{session.stopReason || '--'}</strong>
            </div>
          </div>

          <div className="ops-detail-grid ops-detail-grid--meter">
            <div>
              <span>起始读数</span>
              <strong>{formatOptionalKwh(session.meterStartKwh)}</strong>
            </div>
            <div>
              <span>最终读数</span>
              <strong>{formatOptionalKwh(session.meterStopKwh ?? latestMeter?.meterKwh)}</strong>
            </div>
            <div>
              <span>最新功率</span>
              <strong>{latestMeter ? `${formatNumber(latestMeter.powerKw)} kW` : '--'}</strong>
            </div>
            <div>
              <span>更新时间</span>
              <strong>{formatCompactTime(session.updatedAt)}</strong>
            </div>
          </div>

          <div className="ops-detail-section">
            <div className="ops-detail-title">
              <strong>事件时间线</strong>
              <span>{events.length.toString()}</span>
            </div>
            <div className="ops-event-list">
              {events.slice(0, 8).map((event) => (
                <div className="ops-event" key={event.id}>
                  <time>{formatCompactTime(event.occurredAt)}</time>
                  <div>
                    <strong>{event.eventType}</strong>
                    <code>{event.source}</code>
                  </div>
                </div>
              ))}
              {events.length === 0 ? <div className="ops-empty">暂无事件</div> : null}
            </div>
          </div>

          <div className="ops-detail-section">
            <div className="ops-detail-title">
              <strong>电表读数</strong>
              <span>{(session.meterValues.length).toString()}</span>
            </div>
            <div className="ops-meter-list">
              {session.meterValues.slice(-6).reverse().map((meter) => (
                <div key={`${meter.time}-${meter.meterKwh}`}>
                  <time>{formatCompactTime(meter.time)}</time>
                  <span>{formatNumber(meter.powerKw)} kW</span>
                  <strong>{formatNumber(meter.meterKwh, 3)} kWh</strong>
                </div>
              ))}
              {session.meterValues.length === 0 ? <div className="ops-empty">暂无读数</div> : null}
            </div>
          </div>

          <div className="ops-detail-section">
            <div className="ops-detail-title">
              <strong>账单</strong>
              <span>{sessionDrafts.length.toString()}</span>
            </div>
            <div className="ops-meter-list">
              {sessionDrafts.map((draft) => (
                <div key={draft.id}>
                  <time>{formatCompactTime(draft.generatedAt)}</time>
                  <span>{billingStatusLabels[draft.status]}</span>
                  <strong>{formatMoney(draft.totalAmount)}</strong>
                </div>
              ))}
              {sessionDrafts.length === 0 ? <div className="ops-empty">暂无账单</div> : null}
            </div>
          </div>
        </div>
      ) : (
        <div className="ops-empty ops-session-empty">{isFetching ? '同步中' : '暂无会话'}</div>
      )}
    </section>
  )
}

function formatDetailEnergy(session: SessionDetail): string {
  const energy = sessionEnergy(session)
  return energy === undefined ? '--' : `${formatNumber(energy, 2)} kWh`
}

function formatOptionalKwh(value?: number): string {
  return value === undefined ? '--' : `${formatNumber(value, 3)} kWh`
}
