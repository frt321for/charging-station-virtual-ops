import { AlertTriangle, CheckCircle2, Download, FilePenLine } from 'lucide-react'
import { useState } from 'react'
import type {
  BillingCorrection,
  BillingDraft,
  ConfirmBillRequest,
  CreateCorrectionRequest,
  ReconciliationException,
  ReconciliationExport,
  ReviewExceptionRequest,
} from '../../types/operations'
import {
  billingStatusLabels,
  exceptionSeverityLabels,
  formatCompactTime,
  formatMoney,
  reconciliationStatusLabels,
  severityTone,
} from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface ReconciliationPanelProps {
  exceptions: ReconciliationException[]
  drafts: BillingDraft[]
  siteCode?: string
  isReviewPending: boolean
  isCorrectionPending: boolean
  isConfirmPending: boolean
  isExportPending: boolean
  lastCorrection?: BillingCorrection
  lastExport?: ReconciliationExport
  errorMessage?: string
  canReview: boolean
  reviewerName: string
  onReview: (exceptionId: string, request: ReviewExceptionRequest) => void
  onCreateCorrection: (exceptionId: string, request: CreateCorrectionRequest) => void
  onConfirmBill: (billId: string, request: ConfirmBillRequest) => void
  onExport: (siteCode: string) => void
}

const reviewStatuses: ReconciliationException['status'][] = ['reviewing', 'resolved', 'open']

const exceptionLabels: Record<ReconciliationException['exceptionType'], string> = {
  energy: '电量',
  duration: '时长',
  amount: '金额',
  stop_reason: '停止原因',
}

export function ReconciliationPanel({
  exceptions,
  drafts,
  siteCode,
  isReviewPending,
  isCorrectionPending,
  isConfirmPending,
  isExportPending,
  lastCorrection,
  lastExport,
  errorMessage,
  canReview,
  reviewerName,
  onReview,
  onCreateCorrection,
  onConfirmBill,
  onExport,
}: ReconciliationPanelProps) {
  const [selectedExceptionId, setSelectedExceptionId] = useState('')
  const [reviewStatus, setReviewStatus] = useState<ReconciliationException['status']>('reviewing')
  const [note, setNote] = useState('账单复核')
  const [correctedEnergyKwh, setCorrectedEnergyKwh] = useState('')
  const [correctedDurationMinutes, setCorrectedDurationMinutes] = useState('')
  const [correctedTotalAmount, setCorrectedTotalAmount] = useState('')
  const [correctionReason, setCorrectionReason] = useState('人工复核修正')
  const [selectedBillId, setSelectedBillId] = useState('')

  const selectedException =
    exceptions.find((item) => item.id === selectedExceptionId) ?? exceptions[0]
  const confirmableDraft =
    drafts.find((draft) => draft.id === selectedBillId) ??
    drafts.find((draft) => draft.status !== 'confirmed') ??
    drafts[0]
  const energyValue = parseOptionalNumber(correctedEnergyKwh)
  const durationValue = parseOptionalInteger(correctedDurationMinutes)
  const amountValue = parseOptionalNumber(correctedTotalAmount)
  const canCorrect =
    Boolean(selectedException) &&
    correctionReason.trim() !== '' &&
    reviewerName.trim() !== '' &&
    [energyValue, durationValue, amountValue].some((value) => value !== undefined)

  function submitReview() {
    if (!canReview || !selectedException) return
    onReview(selectedException.id, {
      status: reviewStatus,
      reviewerName,
      note,
    })
  }

  function submitCorrection() {
    if (!canReview || !selectedException || !canCorrect) return
    onCreateCorrection(selectedException.id, {
      correctedEnergyKwh: energyValue,
      correctedDurationMinutes: durationValue,
      correctedTotalAmount: amountValue,
      reason: correctionReason,
      reviewerName,
    })
  }

  return (
    <section className="ops-panel">
      <header>
        <h2>核查异常</h2>
        <StatusBadge tone={exceptions.length > 0 ? 'warn' : 'ok'}>{exceptions.length.toString()}</StatusBadge>
      </header>
      <div className="ops-reconciliation-list">
        <div className="ops-finance-form">
          <label>
            异常
            <select
              aria-label="核查异常"
              name="reconciliationException"
              disabled={!canReview}
              value={selectedException?.id ?? ''}
              onChange={(event) => setSelectedExceptionId(event.target.value)}
            >
              {exceptions.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.exceptionNo} / {exceptionLabels[item.exceptionType]}
                </option>
              ))}
            </select>
          </label>
          <label>
            状态
            <select
              aria-label="核查状态"
              name="reconciliationStatus"
              disabled={!canReview}
              value={reviewStatus}
              onChange={(event) => setReviewStatus(event.target.value as ReconciliationException['status'])}
            >
              {reviewStatuses.map((status) => (
                <option key={status} value={status}>
                  {reconciliationStatusLabels[status]}
                </option>
              ))}
            </select>
          </label>
          <label>
            备注
            <input
              aria-label="核查备注"
              name="financeReviewNote"
              disabled={!canReview}
              value={note}
              onChange={(event) => setNote(event.target.value)}
            />
          </label>
          <button type="button" disabled={!canReview || !selectedException || isReviewPending} onClick={submitReview}>
            <FilePenLine size={16} aria-hidden="true" />
            复核
          </button>
        </div>

        <div className="ops-correction-grid">
          <label>
            修正电量
            <input
              aria-label="修正电量"
              inputMode="decimal"
              name="correctedEnergyKwh"
              disabled={!canReview}
              value={correctedEnergyKwh}
              onChange={(event) => setCorrectedEnergyKwh(event.target.value)}
            />
          </label>
          <label>
            修正时长
            <input
              aria-label="修正时长"
              inputMode="numeric"
              name="correctedDurationMinutes"
              disabled={!canReview}
              value={correctedDurationMinutes}
              onChange={(event) => setCorrectedDurationMinutes(event.target.value)}
            />
          </label>
          <label>
            修正金额
            <input
              aria-label="修正金额"
              inputMode="decimal"
              name="correctedTotalAmount"
              disabled={!canReview}
              value={correctedTotalAmount}
              onChange={(event) => setCorrectedTotalAmount(event.target.value)}
            />
          </label>
          <label>
            原因
            <input
              aria-label="修正原因"
              name="correctionReason"
              disabled={!canReview}
              value={correctionReason}
              onChange={(event) => setCorrectionReason(event.target.value)}
            />
          </label>
          <button type="button" disabled={!canReview || !canCorrect || isCorrectionPending} onClick={submitCorrection}>
            <CheckCircle2 size={16} aria-hidden="true" />
            修正
          </button>
        </div>

        <div className="ops-confirm-row">
          <label>
            账单
            <select
              aria-label="确认账单"
              name="confirmBill"
              disabled={!canReview}
              value={confirmableDraft?.id ?? ''}
              onChange={(event) => setSelectedBillId(event.target.value)}
            >
              {drafts.map((draft) => (
                <option key={draft.id} value={draft.id}>
                  {draft.billNo} / {billingStatusLabels[draft.status]}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            disabled={!canReview || !confirmableDraft || isConfirmPending || confirmableDraft.status === 'confirmed'}
            onClick={() => confirmableDraft && onConfirmBill(confirmableDraft.id, { reviewerName, note })}
          >
            <CheckCircle2 size={16} aria-hidden="true" />
            确认
          </button>
          <button type="button" disabled={!canReview || !siteCode || isExportPending} onClick={() => siteCode && onExport(siteCode)}>
            <Download size={16} aria-hidden="true" />
            导出
          </button>
        </div>

        {exceptions.slice(0, 5).map((item) => (
          <article key={item.id} className="ops-exception-item">
            <div>
              <AlertTriangle size={16} aria-hidden="true" />
              <strong>{exceptionLabels[item.exceptionType]}</strong>
              <StatusBadge tone={severityTone(item.severity)}>{exceptionSeverityLabels[item.severity]}</StatusBadge>
              <StatusBadge tone={item.status === 'resolved' ? 'ok' : 'warn'}>
                {reconciliationStatusLabels[item.status]}
              </StatusBadge>
            </div>
            <p>{item.reason}</p>
            <footer>
              <code>{item.billNo}</code>
              <span>{formatCompactTime(item.detectedAt)}</span>
            </footer>
          </article>
        ))}

        {exceptions.length === 0 ? <div className="ops-empty">暂无核查异常</div> : null}
        {lastCorrection ? <output className="ops-inline-result">{lastCorrection.correctionNo}</output> : null}
        {lastExport ? (
          <output className="ops-inline-result">
            {lastExport.filename} / {lastExport.rows.length} / {formatMoney(totalExportAmount(lastExport))}
          </output>
        ) : null}
        {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
      </div>
    </section>
  )
}

function parseOptionalNumber(value: string): number | undefined {
  if (value.trim() === '') return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined
}

function parseOptionalInteger(value: string): number | undefined {
  if (value.trim() === '') return undefined
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : undefined
}

function totalExportAmount(exportPackage: ReconciliationExport): number {
  return exportPackage.rows.reduce((total, row) => total + row.totalAmount, 0)
}
