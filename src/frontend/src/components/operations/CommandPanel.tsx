import { RotateCcw, Send, Square, Zap } from 'lucide-react'
import type { CommandPathType, SessionDetail } from '../../types/operations'
import { canRunCommand, formatCompactTime, sessionStatusLabels, sessionStatusTone } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface CommandPanelProps {
  session?: SessionDetail
  targetConnectorCode?: string
  targetPowerKw: string
  isTargetPowerValid: boolean
  isPending: boolean
  lastMessage?: string
  errorMessage?: string
  canWrite: boolean
  onTargetPowerChange: (value: string) => void
  onCommand: (commandType: CommandPathType) => void
}

const commands: Array<{ type: CommandPathType; label: string; danger?: boolean }> = [
  { type: 'start', label: '启动' },
  { type: 'stop', label: '停止', danger: true },
  { type: 'pause', label: '暂停' },
  { type: 'resume', label: '恢复' },
  { type: 'reset', label: '复位' },
  { type: 'limit-power', label: '限功率' },
]

export function CommandPanel({
  session,
  targetConnectorCode,
  targetPowerKw,
  isTargetPowerValid,
  isPending,
  lastMessage,
  errorMessage,
  canWrite,
  onTargetPowerChange,
  onCommand,
}: CommandPanelProps) {
  const status = session?.status
  const events = session?.events.slice(-3).reverse() ?? []
  const statusTone = status ? sessionStatusTone(status) : 'idle'

  return (
    <>
      <section className="ops-panel">
        <header>
          <h2>远程控制</h2>
          <StatusBadge tone={statusTone}>
            {status ? sessionStatusLabels[status] : '未选择'}
          </StatusBadge>
        </header>
        <div className="ops-control-body">
          <div className="ops-target">
            <span>目标枪口</span>
            <strong>{session?.connectorCode ?? targetConnectorCode ?? '--'}</strong>
          </div>
          <div className="ops-actions">
            {commands.map((command) => (
              <button
                key={command.type}
                type="button"
                className={command.danger ? 'ops-danger-button' : undefined}
                disabled={!canWrite || isPending || !canRunCommand(status, command.type)}
                onClick={() => onCommand(command.type)}
              >
                {command.type === 'start' ? <Send size={16} aria-hidden="true" /> : null}
                {command.type === 'stop' ? <Square size={16} aria-hidden="true" /> : null}
                {command.type === 'reset' ? <RotateCcw size={16} aria-hidden="true" /> : null}
                {command.type === 'limit-power' ? <Zap size={16} aria-hidden="true" /> : null}
                {command.label}
              </button>
            ))}
          </div>
          <div className="ops-limit">
            <label>
              目标功率
              <input
                value={targetPowerKw}
                inputMode="decimal"
                aria-label="目标功率"
                aria-invalid={!isTargetPowerValid}
                name="targetPowerKw"
                onChange={(event) => onTargetPowerChange(event.target.value)}
              />
            </label>
            <button
              type="button"
              disabled={!canWrite || isPending || !canRunCommand(status, 'limit-power') || !isTargetPowerValid}
              onClick={() => onCommand('limit-power')}
            >
              应用
            </button>
          </div>
          {!isTargetPowerValid ? <output className="ops-inline-error">功率需大于 0</output> : null}
          {lastMessage ? <output className="ops-inline-result">{lastMessage}</output> : null}
          {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
        </div>
      </section>

      <section className="ops-panel">
        <header>
          <h2>命令时间线</h2>
          <span className="ops-mono">{formatCompactTime(events[0]?.occurredAt)}</span>
        </header>
        <div className="ops-event-list">
          {events.length > 0 ? (
            events.map((event) => (
              <div className="ops-event" key={event.id}>
                <time>{formatCompactTime(event.occurredAt)}</time>
                <div>
                  <strong>{event.eventType}</strong>
                  <code>{event.source}</code>
                </div>
              </div>
            ))
          ) : (
            <div className="ops-empty">暂无事件</div>
          )}
        </div>
      </section>
    </>
  )
}
