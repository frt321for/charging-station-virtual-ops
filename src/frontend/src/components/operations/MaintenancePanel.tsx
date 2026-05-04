import { ArrowRight, Wrench } from 'lucide-react'
import { useMemo, useState } from 'react'
import type {
  CreateWorkOrderRequest,
  MaintenanceSnapshot,
  TransitionWorkOrderRequest,
  WorkOrder,
  WorkOrderEvent,
  WorkOrderStatus,
} from '../../types/operations'
import {
  exceptionSeverityLabels,
  faultStatusLabels,
  formatCompactTime,
  formatNumber,
  severityTone,
  workOrderStatusLabels,
  workOrderStatusTone,
} from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface MaintenancePanelProps {
  snapshot?: MaintenanceSnapshot
  events: WorkOrderEvent[]
  selectedWorkOrderId?: string
  isEventsFetching: boolean
  isCreatePending: boolean
  isTransitionPending: boolean
  lastWorkOrder?: WorkOrder
  errorMessage?: string
  canWrite: boolean
  actorName: string
  onSelectWorkOrder: (workOrderId: string) => void
  onCreateWorkOrder: (faultId: string, request: CreateWorkOrderRequest) => void
  onTransitionWorkOrder: (workOrderId: string, request: TransitionWorkOrderRequest) => void
}

const impactOptions = ['connector', 'charger', 'site', 'session']
const transitionMap: Partial<Record<WorkOrderStatus, WorkOrderStatus[]>> = {
  open: ['assigned', 'cancelled'],
  assigned: ['accepted', 'cancelled'],
  accepted: ['arrived', 'cancelled'],
  arrived: ['handling', 'cancelled'],
  handling: ['retest', 'cancelled'],
  retest: ['recovered', 'handling', 'cancelled'],
  recovered: ['closed'],
}

export function MaintenancePanel({
  snapshot,
  events,
  selectedWorkOrderId,
  isEventsFetching,
  isCreatePending,
  isTransitionPending,
  lastWorkOrder,
  errorMessage,
  canWrite,
  actorName,
  onSelectWorkOrder,
  onCreateWorkOrder,
  onTransitionWorkOrder,
}: MaintenancePanelProps) {
  const [selectedFaultId, setSelectedFaultId] = useState('')
  const [assigneeName, setAssigneeName] = useState('maintenance-shift-a')
  const [impactScope, setImpactScope] = useState('connector')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [targetStatus, setTargetStatus] = useState<WorkOrderStatus>('assigned')
  const [transitionNote, setTransitionNote] = useState('现场确认')

  const faults = snapshot?.faults ?? []
  const workOrders = snapshot?.workOrders ?? []
  const faultSelectValue = selectedFaultId || faults[0]?.id || ''
  const selectedFault = faults.find((fault) => fault.id === faultSelectValue)
  const latestSelectedWorkOrder =
    lastWorkOrder && (!selectedWorkOrderId || lastWorkOrder.id === selectedWorkOrderId) ? lastWorkOrder : undefined
  const selectedWorkOrder =
    latestSelectedWorkOrder ?? workOrders.find((item) => item.id === selectedWorkOrderId) ?? workOrders[0]
  const availableTargets = useMemo(
    () => (selectedWorkOrder ? transitionMap[selectedWorkOrder.status] ?? [] : []),
    [selectedWorkOrder],
  )
  const targetStatusValue = availableTargets.includes(targetStatus)
    ? targetStatus
    : availableTargets[0] ?? targetStatus

  function submitWorkOrder() {
    if (!canWrite || !selectedFault || assigneeName.trim() === '') return
    onCreateWorkOrder(selectedFault.id, {
      assigneeName,
      impactScope,
      title: title.trim() || `${selectedFault.chargerCode} / ${selectedFault.faultCode}`,
      description: description.trim() || selectedFault.faultCode,
      actorName,
    })
  }

  function submitTransition() {
    if (!canWrite || !selectedWorkOrder || !availableTargets.includes(targetStatusValue)) return
    onTransitionWorkOrder(selectedWorkOrder.id, {
      targetStatus: targetStatusValue,
      assigneeName,
      actorName,
      note: transitionNote,
      payload: { source: 'operations-console' },
    })
  }

  return (
    <section className="ops-panel">
      <header>
        <h2>维保工单</h2>
        <StatusBadge tone={(snapshot?.sla.openWorkOrders ?? 0) > 0 ? 'warn' : 'ok'}>
          {(snapshot?.sla.openWorkOrders ?? 0).toString()}
        </StatusBadge>
      </header>

      <div className="ops-maintenance">
        <div className="ops-sla-grid">
          <div>
            <span>响应超时</span>
            <data>{snapshot?.sla.overdueResponse ?? 0}</data>
          </div>
          <div>
            <span>恢复超时</span>
            <data>{snapshot?.sla.overdueRecovery ?? 0}</data>
          </div>
          <div>
            <span>SLA 命中</span>
            <data>{formatNumber(snapshot?.sla.recoverySlaHitRate ?? 100, 1)}%</data>
          </div>
          <div>
            <span>重复故障</span>
            <data>{snapshot?.sla.repeatedFaults ?? 0}</data>
          </div>
        </div>

        <div className="ops-maintenance-form">
          <label>
            故障
            <select
              aria-label="故障对象"
              name="faultId"
              disabled={!canWrite}
              value={faultSelectValue}
              onChange={(event) => setSelectedFaultId(event.target.value)}
            >
              {faults.map((fault) => (
                <option key={fault.id} value={fault.id}>
                  {fault.faultNo} / {fault.chargerCode}
                </option>
              ))}
            </select>
          </label>
          <label>
            班组
            <input
              aria-label="维保班组"
              name="maintenanceAssignee"
              disabled={!canWrite}
              value={assigneeName}
              onChange={(event) => setAssigneeName(event.target.value)}
            />
          </label>
          <label>
            范围
            <select
              aria-label="影响范围"
              name="impactScope"
              disabled={!canWrite}
              value={impactScope}
              onChange={(event) => setImpactScope(event.target.value)}
            >
              {impactOptions.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>
          <label>
            标题
            <input
              aria-label="工单标题"
              name="workOrderTitle"
              placeholder={selectedFault ? `${selectedFault.chargerCode} / ${selectedFault.faultCode}` : ''}
              disabled={!canWrite}
              value={title}
              onChange={(event) => setTitle(event.target.value)}
            />
          </label>
          <label>
            记录
            <input
              aria-label="工单记录"
              name="workOrderDescription"
              placeholder={selectedFault?.faultCode ?? ''}
              disabled={!canWrite}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </label>
        </div>
        <button type="button" className="ops-wide-button" disabled={!canWrite || !selectedFault || isCreatePending} onClick={submitWorkOrder}>
          <Wrench size={16} aria-hidden="true" />
          {selectedFault?.status === 'open' ? '派单' : '关联工单'}
        </button>

        <div className="ops-maintenance-grid">
          <div className="ops-fault-list">
            {faults.slice(0, 5).map((fault) => (
              <article key={fault.id}>
                <div>
                  <strong>{fault.faultNo}</strong>
                  <StatusBadge tone={severityTone(fault.severity)}>{exceptionSeverityLabels[fault.severity]}</StatusBadge>
                </div>
                <span>{fault.chargerCode} / {fault.connectorCode || fault.faultCode}</span>
                <footer>
                  <code>{faultStatusLabels[fault.status]}</code>
                  <time>{formatCompactTime(fault.occurredAt)}</time>
                </footer>
              </article>
            ))}
            {faults.length === 0 ? <div className="ops-empty">暂无故障</div> : null}
          </div>

          <div className="ops-workorder-list">
            {workOrders.slice(0, 6).map((order) => (
              <button
                key={order.id}
                type="button"
                className={order.id === selectedWorkOrder?.id ? 'ops-row-selected' : undefined}
                onClick={() => onSelectWorkOrder(order.id)}
              >
                <span>
                  <strong>{order.workOrderNo}</strong>
                  <code>{order.assigneeName}</code>
                </span>
                <StatusBadge tone={workOrderStatusTone(order.status)}>
                  {workOrderStatusLabels[order.status]}
                </StatusBadge>
              </button>
            ))}
            {workOrders.length === 0 ? <div className="ops-empty">暂无工单</div> : null}
          </div>
        </div>

        <div className="ops-transition-form">
          <label>
            目标状态
            <select
              aria-label="工单目标状态"
              name="workOrderTargetStatus"
              disabled={!canWrite}
              value={targetStatusValue}
              onChange={(event) => setTargetStatus(event.target.value as WorkOrderStatus)}
            >
              {availableTargets.map((status) => (
                <option key={status} value={status}>
                  {workOrderStatusLabels[status]}
                </option>
              ))}
            </select>
          </label>
          <label>
            处理记录
            <input
              aria-label="工单处理记录"
              name="workOrderTransitionNote"
              disabled={!canWrite}
              value={transitionNote}
              onChange={(event) => setTransitionNote(event.target.value)}
            />
          </label>
          <button
            type="button"
            disabled={!canWrite || !selectedWorkOrder || availableTargets.length === 0 || isTransitionPending}
            onClick={submitTransition}
          >
            <ArrowRight size={16} aria-hidden="true" />
            流转
          </button>
        </div>

        <div className="ops-event-list">
          {events.map((event) => (
            <div className="ops-event" key={event.id}>
              <time>{formatCompactTime(event.occurredAt)}</time>
              <div>
                <strong>{event.toStatus ? workOrderStatusLabels[event.toStatus as WorkOrderStatus] : event.eventType}</strong>
                <code>{event.actorName}</code>
              </div>
            </div>
          ))}
          {isEventsFetching ? <StatusBadge tone="charge">同步中</StatusBadge> : null}
          {events.length === 0 ? <div className="ops-empty">暂无工单事件</div> : null}
        </div>

        {lastWorkOrder ? <output className="ops-inline-result">{lastWorkOrder.workOrderNo}</output> : null}
        {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
      </div>
    </section>
  )
}
