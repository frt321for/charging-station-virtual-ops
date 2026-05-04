import { Gauge, SlidersHorizontal } from 'lucide-react'
import { useMemo, useState } from 'react'
import type {
  CreateLoadControlRecordRequest,
  LoadActionType,
  LoadControlRecord,
  LoadControlSnapshot,
  LoadRecordStatus,
  SiteSummary,
} from '../../types/operations'
import { formatCompactTime, formatNumber, loadActionLabels, loadRecordStatusLabels } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface LoadControlPanelProps {
  site?: SiteSummary
  snapshot?: LoadControlSnapshot
  isPending: boolean
  lastRecord?: LoadControlRecord
  errorMessage?: string
  canWrite: boolean
  operatorName: string
  onCreateRecord: (request: CreateLoadControlRecordRequest) => void
}

const actionOptions: LoadActionType[] = ['limit_power', 'pause', 'resume', 'queue', 'reject', 'promote', 'release']
const statusOptions: LoadRecordStatus[] = ['applied', 'sent', 'recommended', 'rejected']

export function LoadControlPanel({
  site,
  snapshot,
  isPending,
  lastRecord,
  errorMessage,
  canWrite,
  operatorName,
  onCreateRecord,
}: LoadControlPanelProps) {
  const [targetSessionNo, setTargetSessionNo] = useState('')
  const [actionType, setActionType] = useState<LoadActionType>('limit_power')
  const [recordStatus, setRecordStatus] = useState<LoadRecordStatus>('applied')
  const [targetPowerKw, setTargetPowerKw] = useState('3.5')
  const [afterLoadKw, setAfterLoadKw] = useState('')
  const [reason, setReason] = useState('site-load-limit')
  const selectedQueueItem =
    snapshot?.queue.find((item) => item.sessionNo === targetSessionNo) ?? snapshot?.queue[0]
  const loadLimit = snapshot?.loadLimitKw ?? site?.loadLimitKw ?? 0
  const currentLoad = snapshot?.currentLoadKw ?? 0
  const targetPower = Number(targetPowerKw)
  const afterLoadInput = afterLoadKw.trim()
  const afterLoad = afterLoadInput === '' ? currentLoad : Number(afterLoadInput)
  const isTargetPowerValid =
    targetPowerKw.trim() !== '' && Number.isFinite(targetPower) && targetPower > 0
  const isAfterLoadValid = Number.isFinite(afterLoad) && afterLoad >= 0
  const loadRatio = loadLimit > 0 ? Math.min(100, (currentLoad / loadLimit) * 100) : 0
  const policiesByScope = useMemo(
    () => ({
      site: snapshot?.policies.filter((policy) => policy.scopeType === 'site') ?? [],
      area: snapshot?.policies.filter((policy) => policy.scopeType === 'area') ?? [],
      group: snapshot?.policies.filter((policy) => policy.scopeType === 'group') ?? [],
    }),
    [snapshot?.policies],
  )
  const canSubmit =
    Boolean(snapshot?.siteCode) &&
    reason.trim() !== '' &&
    isAfterLoadValid &&
    (actionType !== 'limit_power' || isTargetPowerValid)

  function submitRecord() {
    if (!snapshot || !canWrite || !canSubmit) return
    onCreateRecord({
      siteId: snapshot.siteCode,
      sessionId: selectedQueueItem?.sessionNo,
      actionType,
      reason,
      beforeLoadKw: currentLoad,
      afterLoadKw: afterLoad,
      targetPowerKw: actionType === 'limit_power' ? targetPower : undefined,
      status: recordStatus,
      operatorName,
    })
  }

  return (
    <section className="ops-panel">
      <header>
        <h2>负载控制</h2>
        <StatusBadge tone={loadRatio >= 90 ? 'danger' : loadRatio >= 75 ? 'warn' : 'ok'}>
          {`${formatNumber(loadRatio, 0)}%`}
        </StatusBadge>
      </header>
      <div className="ops-load-control">
        <div className="ops-load-meter">
          <Gauge size={20} aria-hidden="true" />
          <data>{formatNumber(currentLoad, 1)} kW</data>
          <span>{formatNumber(snapshot?.availableCapacityKw ?? 0, 1)} kW</span>
        </div>
        <div className="ops-bar" aria-label="站点负载率">
          <span style={{ width: `${loadRatio}%` }} />
        </div>

        <div className="ops-policy-grid">
          {[...policiesByScope.site, ...policiesByScope.area, ...policiesByScope.group].slice(0, 5).map((policy) => (
            <div key={policy.id}>
              <strong>{policy.scopeCode}</strong>
              <span>{loadActionLabels[policy.actionMode]}</span>
              <data>{formatNumber(policy.thresholdKw, 0)} kW</data>
            </div>
          ))}
        </div>

        <div className="ops-load-form">
          <label>
            对象
            <select
              aria-label="负载控制对象"
              name="loadControlTarget"
              value={selectedQueueItem?.sessionNo ?? ''}
              onChange={(event) => setTargetSessionNo(event.target.value)}
            >
              {snapshot?.queue.map((item) => (
                <option key={item.sessionId} value={item.sessionNo}>
                  {item.position.toString().padStart(2, '0')} / {item.connectorCode}
                </option>
              ))}
              {snapshot?.queue.length === 0 ? <option value="">站点</option> : null}
            </select>
          </label>
          <label>
            动作
            <select
              aria-label="负载控制动作"
              name="loadControlAction"
              value={actionType}
              onChange={(event) => setActionType(event.target.value as LoadActionType)}
            >
              {actionOptions.map((action) => (
                <option key={action} value={action}>
                  {loadActionLabels[action]}
                </option>
              ))}
            </select>
          </label>
          <label>
            状态
            <select
              aria-label="负载控制状态"
              name="loadControlStatus"
              value={recordStatus}
              onChange={(event) => setRecordStatus(event.target.value as LoadRecordStatus)}
            >
              {statusOptions.map((status) => (
                <option key={status} value={status}>
                  {loadRecordStatusLabels[status]}
                </option>
              ))}
            </select>
          </label>
          <label>
            目标功率
            <input
              aria-label="负载目标功率"
              aria-invalid={actionType === 'limit_power' && !isTargetPowerValid}
              inputMode="decimal"
              name="loadTargetPowerKw"
              value={targetPowerKw}
              onChange={(event) => setTargetPowerKw(event.target.value)}
            />
          </label>
          <label>
            调整后
            <input
              aria-label="调整后负载"
              aria-invalid={!isAfterLoadValid}
              inputMode="decimal"
              name="loadAfterKw"
              placeholder={formatNumber(currentLoad, 1)}
              value={afterLoadKw}
              onChange={(event) => setAfterLoadKw(event.target.value)}
            />
          </label>
          <label>
            原因
            <input
              aria-label="负载控制原因"
              name="loadReason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
        </div>
        <button type="button" className="ops-wide-button" disabled={!canWrite || isPending || !canSubmit} onClick={submitRecord}>
          <SlidersHorizontal size={16} aria-hidden="true" />
          写入记录
        </button>

        <div className="ops-queue-list">
          {snapshot?.queue.slice(0, 3).map((item) => (
            <div className="ops-queue-item" key={item.sessionId}>
              <span className="ops-rank">{item.position.toString().padStart(2, '0')}</span>
              <div>
                <strong>{item.connectorCode}</strong>
                <span>{item.reason}</span>
              </div>
              <span className="ops-mono">{item.waitMinutes}m</span>
            </div>
          ))}
          {snapshot?.queue.length === 0 ? <div className="ops-empty">暂无排队会话</div> : null}
        </div>

        <div className="ops-record-list">
          {snapshot?.records.slice(0, 4).map((record) => (
            <div key={record.id}>
              <strong>{loadActionLabels[record.actionType]}</strong>
              <span>{record.connectorCode || record.scopeCode}</span>
              <code>{formatCompactTime(record.createdAt)}</code>
            </div>
          ))}
          {snapshot?.records.length === 0 ? <div className="ops-empty">暂无控制记录</div> : null}
        </div>

        {lastRecord ? <output className="ops-inline-result">{lastRecord.recordNo}</output> : null}
        {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
      </div>
    </section>
  )
}
