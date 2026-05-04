import type { SessionDetail, SessionSummary } from '../../types/operations'
import {
  formatNumber,
  latestMeterPower,
  sessionEnergy,
  sessionStatusLabels,
  sessionStatusTone,
} from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface SessionTableProps {
  sessions: SessionSummary[]
  details: SessionDetail[]
  selectedSessionNo?: string
  canStopSessions: boolean
  onSelectSession: (sessionNo: string) => void
  onStopSession: (sessionNo: string) => void
}

export function SessionTable({
  sessions,
  details,
  selectedSessionNo,
  canStopSessions,
  onSelectSession,
  onStopSession,
}: SessionTableProps) {
  const detailBySessionNo = new Map(details.map((detail) => [detail.sessionNo, detail]))

  return (
    <section className="ops-panel">
      <header>
        <h2>会话队列</h2>
        <StatusBadge tone="warn">{`${sessions.length} 条记录`}</StatusBadge>
      </header>
      <div className="ops-table-wrap">
        <table>
          <thead>
            <tr>
              <th>会话</th>
              <th>枪口</th>
              <th>状态</th>
              <th className="ops-num">功率</th>
              <th className="ops-num">电量</th>
              <th>处理</th>
            </tr>
          </thead>
          <tbody>
            {sessions.slice(0, 8).map((session) => {
              const detail = detailBySessionNo.get(session.sessionNo)
              const isSelected = selectedSessionNo === session.sessionNo
              const canStop = canStopSessions && (session.status === 'charging' || session.status === 'paused')

              return (
                <tr key={session.id} className={isSelected ? 'ops-row-selected' : undefined}>
                  <td className="ops-num">{session.sessionNo}</td>
                  <td>{session.connectorCode}</td>
                  <td>
                    <StatusBadge tone={sessionStatusTone(session.status)}>
                      {sessionStatusLabels[session.status]}
                    </StatusBadge>
                  </td>
                  <td className="ops-num">{formatNumber(latestMeterPower(detail))} kW</td>
                  <td className="ops-num">
                    {sessionEnergy(detail) === undefined ? '--' : `${formatNumber(sessionEnergy(detail) ?? 0, 2)} kWh`}
                  </td>
                  <td>
                    <div className="ops-row-actions">
                      <button type="button" onClick={() => onSelectSession(session.sessionNo)}>
                        详情
                      </button>
                      <button
                        type="button"
                        className="ops-danger-text"
                        disabled={!canStop}
                        onClick={() => onStopSession(session.sessionNo)}
                      >
                        停止
                      </button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}
