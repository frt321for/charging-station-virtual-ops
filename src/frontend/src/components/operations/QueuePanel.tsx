import type { SessionSummary } from '../../types/operations'
import { formatCompactTime } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface QueuePanelProps {
  sessions: SessionSummary[]
  queueId?: string
  slaId?: string
}

export function QueuePanel({ sessions, queueId, slaId }: QueuePanelProps) {
  const queued = sessions.filter(
    (session) => session.status === 'reserved' || session.status === 'waiting_arrival',
  )
  const risk = sessions.find((session) => session.status === 'pending_review') ?? sessions[0]

  return (
    <>
      <section className="ops-panel" id={queueId}>
        <header>
          <h2>负载队列</h2>
          <StatusBadge tone="warn">{queued.length.toString()}</StatusBadge>
        </header>
        <div className="ops-queue-list">
          {queued.slice(0, 2).map((session, index) => (
            <div className="ops-queue-item" key={session.id}>
              <span className="ops-rank">{(index + 1).toString().padStart(2, '0')}</span>
              <div>
                <strong>{session.connectorCode}</strong>
                <span>{formatCompactTime(session.reservationExpiry)} 到期</span>
              </div>
              <span className="ops-mono">{formatCompactTime(session.updatedAt)}</span>
            </div>
          ))}
          {queued.length === 0 ? <div className="ops-empty">暂无排队会话</div> : null}
        </div>
      </section>

      <section className="ops-panel" id={slaId}>
        <header>
          <h2>SLA</h2>
          <StatusBadge tone={risk?.status === 'pending_review' ? 'danger' : 'ok'}>
            {risk?.status === 'pending_review' ? '风险' : '正常'}
          </StatusBadge>
        </header>
        <div className="ops-sla">
          <strong>{risk?.sessionNo ?? '--'}</strong>
          <span>{risk ? `${risk.connectorCode} / ${risk.status}` : '--'}</span>
        </div>
      </section>
    </>
  )
}
