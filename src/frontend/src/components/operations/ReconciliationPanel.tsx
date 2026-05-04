import { AlertTriangle } from 'lucide-react'
import type { ReconciliationException } from '../../types/operations'
import { exceptionSeverityLabels, formatCompactTime } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface ReconciliationPanelProps {
  exceptions: ReconciliationException[]
}

const severityTone: Record<ReconciliationException['severity'], 'ok' | 'charge' | 'warn' | 'danger' | 'idle'> = {
  low: 'idle',
  medium: 'warn',
  high: 'danger',
  critical: 'danger',
}

const exceptionLabels: Record<ReconciliationException['exceptionType'], string> = {
  energy: '电量',
  duration: '时长',
  amount: '金额',
  stop_reason: '停止原因',
}

export function ReconciliationPanel({ exceptions }: ReconciliationPanelProps) {
  return (
    <section className="ops-panel">
      <header>
        <h2>核查异常</h2>
        <StatusBadge tone={exceptions.length > 0 ? 'warn' : 'ok'}>{exceptions.length.toString()}</StatusBadge>
      </header>
      <div className="ops-reconciliation-list">
        {exceptions.slice(0, 5).map((item) => (
          <article key={item.id} className="ops-exception-item">
            <div>
              <AlertTriangle size={16} aria-hidden="true" />
              <strong>{exceptionLabels[item.exceptionType]}</strong>
              <StatusBadge tone={severityTone[item.severity]}>{exceptionSeverityLabels[item.severity]}</StatusBadge>
            </div>
            <p>{item.reason}</p>
            <footer>
              <code>{item.billNo}</code>
              <span>{formatCompactTime(item.detectedAt)}</span>
            </footer>
          </article>
        ))}
        {exceptions.length === 0 ? <div className="ops-empty">暂无核查异常</div> : null}
      </div>
    </section>
  )
}
