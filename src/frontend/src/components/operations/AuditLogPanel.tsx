import { RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useAuditLogs } from '../../hooks/useAuditLogs'
import { formatCompactTime } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface AuditLogPanelProps {
  enabled: boolean
}

export function AuditLogPanel({ enabled }: AuditLogPanelProps) {
  const [entityType, setEntityType] = useState('')
  const [action, setAction] = useState('')
  const [actorName, setActorName] = useState('')
  const filters = {
    entityType: entityType.trim() || undefined,
    action: action.trim() || undefined,
    actorName: actorName.trim() || undefined,
    page: 1,
    pageSize: 20,
  }
  const auditQuery = useAuditLogs(filters, enabled)
  const page = auditQuery.data
  const logs = page?.list ?? []

  return (
    <section className="ops-panel">
      <header>
        <h2>审计日志</h2>
        <StatusBadge tone={auditQuery.isFetching ? 'charge' : 'ok'}>
          {(page?.pagination.total ?? logs.length).toString()}
        </StatusBadge>
      </header>
      <div className="ops-audit-body">
        <div className="ops-audit-filters">
          <label>
            对象
            <input
              aria-label="审计对象"
              name="auditEntityType"
              value={entityType}
              onChange={(event) => setEntityType(event.target.value)}
            />
          </label>
          <label>
            动作
            <input
              aria-label="审计动作"
              name="auditAction"
              value={action}
              onChange={(event) => setAction(event.target.value)}
            />
          </label>
          <label>
            操作人
            <input
              aria-label="审计操作人"
              name="auditActorName"
              value={actorName}
              onChange={(event) => setActorName(event.target.value)}
            />
          </label>
          <button type="button" disabled={auditQuery.isFetching} onClick={() => void auditQuery.refetch()}>
            <RefreshCw size={16} aria-hidden="true" className={auditQuery.isFetching ? 'ops-spin' : undefined} />
            刷新
          </button>
        </div>

        <div className="ops-table-wrap">
          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>对象</th>
                <th>动作</th>
                <th>操作人</th>
                <th>链路</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td className="ops-num">{formatCompactTime(log.createdAt)}</td>
                  <td>{log.entityType}</td>
                  <td>{log.action}</td>
                  <td>
                    <strong>{log.actorName || '--'}</strong>
                    <span className="ops-cell-sub">{log.actorRoleCode}</span>
                  </td>
                  <td className="ops-num">{log.traceId || '--'}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {logs.length === 0 ? <div className="ops-empty">暂无审计日志</div> : null}
        </div>
      </div>
    </section>
  )
}
