import { ReceiptText } from 'lucide-react'
import { useMemo, useState } from 'react'
import type { BillingDraft, PricingPolicy, SessionSummary } from '../../types/operations'
import {
  billingStatusLabels,
  formatCompactTime,
  formatMinuteRange,
  formatMoney,
  formatNumber,
  sessionStatusLabels,
} from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface BillingDraftPanelProps {
  sessions: SessionSummary[]
  policies: PricingPolicy[]
  drafts: BillingDraft[]
  isPending: boolean
  lastDraft?: BillingDraft
  errorMessage?: string
  onGenerate: (sessionNo: string) => void
}

export function BillingDraftPanel({
  sessions,
  policies,
  drafts,
  isPending,
  lastDraft,
  errorMessage,
  onGenerate,
}: BillingDraftPanelProps) {
  const [selectedSessionNo, setSelectedSessionNo] = useState('')
  const billableSessions = useMemo(
    () =>
      sessions.filter(
        (session) =>
          session.status === 'pending_billing' ||
          session.status === 'pending_review' ||
          session.status === 'billed',
      ),
    [sessions],
  )
  const selectedSession =
    billableSessions.find((session) => session.sessionNo === selectedSessionNo) ?? billableSessions[0]
  const selectedSessionHasDraft = Boolean(
    selectedSession && drafts.some((draft) => draft.sessionNo === selectedSession.sessionNo),
  )
  const activePolicy = policies[0]

  return (
    <section className="ops-panel">
      <header>
        <h2>计费草稿</h2>
        <StatusBadge tone={drafts.some((draft) => draft.exceptionFlag) ? 'warn' : 'ok'}>
          {drafts.length.toString()}
        </StatusBadge>
      </header>
      <div className="ops-billing-body">
        <div className="ops-billing-form">
          <label>
            会话
            <select
              aria-label="计费会话"
              name="billingSessionNo"
              value={selectedSession?.sessionNo ?? ''}
              onChange={(event) => setSelectedSessionNo(event.target.value)}
            >
              {billableSessions.map((session) => (
                <option key={session.id} value={session.sessionNo}>
                  {session.sessionNo} / {sessionStatusLabels[session.status]}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            disabled={isPending || !selectedSession || selectedSessionHasDraft}
            onClick={() => selectedSession && onGenerate(selectedSession.sessionNo)}
          >
            <ReceiptText size={16} aria-hidden="true" />
            {selectedSessionHasDraft ? '已生成' : '生成草稿'}
          </button>
        </div>

        <div className="ops-price-strip">
          {activePolicy?.periods.map((period) => (
            <div key={period.id}>
              <strong>{period.label}</strong>
              <span>{formatMinuteRange(period.startMinute, period.endMinute)}</span>
              <data>{formatMoney(period.energyPricePerKwh)}</data>
            </div>
          ))}
          {!activePolicy ? <div className="ops-empty">暂无定价策略</div> : null}
        </div>

        <div className="ops-table-wrap">
          <table>
            <thead>
              <tr>
                <th>账单</th>
                <th>枪口</th>
                <th className="ops-num">电量</th>
                <th className="ops-num">金额</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {drafts.slice(0, 5).map((draft) => (
                <tr key={draft.id}>
                  <td>
                    <strong>{draft.billNo}</strong>
                    <span className="ops-cell-sub">{formatCompactTime(draft.generatedAt)}</span>
                  </td>
                  <td>{draft.connectorCode}</td>
                  <td className="ops-num">{formatNumber(draft.energyKwh, 3)} kWh</td>
                  <td className="ops-num">{formatMoney(draft.totalAmount)}</td>
                  <td>
                    <StatusBadge tone={draft.exceptionFlag ? 'warn' : 'ok'}>
                      {billingStatusLabels[draft.status]}
                    </StatusBadge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {drafts.length === 0 ? <div className="ops-empty">暂无账单草稿</div> : null}
        </div>

        {lastDraft ? <output className="ops-inline-result">{lastDraft.billNo}</output> : null}
        {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
      </div>
    </section>
  )
}
