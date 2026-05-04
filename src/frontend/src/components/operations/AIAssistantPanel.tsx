import { Bot, SendHorizontal } from 'lucide-react'
import { useMemo, useState } from 'react'
import type {
  AIInsight,
  AIInsightKind,
  AIInsightRequest,
  SessionSummary,
  WorkOrder,
} from '../../types/operations'
import { formatCompactTime } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface AIAssistantPanelProps {
  sessions: SessionSummary[]
  workOrders: WorkOrder[]
  siteCode?: string
  insight?: AIInsight
  isPending: boolean
  errorMessage?: string
  canGenerate: boolean
  onGenerate: (request: AIInsightRequest) => void
}

const kindOptions: Array<{ value: AIInsightKind; label: string }> = [
  { value: 'session_explanation', label: '会话解释' },
  { value: 'work_order_summary', label: '工单摘要' },
  { value: 'congestion_risk', label: '拥堵风险' },
  { value: 'daily_report', label: '站点日报' },
  { value: 'station_qa', label: '站点问答' },
]

export function AIAssistantPanel({
  sessions,
  workOrders,
  siteCode,
  insight,
  isPending,
  errorMessage,
  canGenerate,
  onGenerate,
}: AIAssistantPanelProps) {
  const [kind, setKind] = useState<AIInsightKind>('session_explanation')
  const [sessionId, setSessionId] = useState('')
  const [workOrderId, setWorkOrderId] = useState('')
  const [horizonHours, setHorizonHours] = useState('4')
  const [businessDate, setBusinessDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [question, setQuestion] = useState('当前站点最大的运营风险是什么')
  const [draft, setDraft] = useState({ requestId: '', value: '' })

  const activeSessionId = sessionId || sessions[0]?.sessionNo || ''
  const activeWorkOrderId = workOrderId || workOrders[0]?.id || ''
  const parsedHorizon = Number(horizonHours)
  const activeInsight = insight?.kind === kind ? insight : undefined
  const draftRequestId = activeInsight?.requestId ?? ''
  const draftValue = draft.requestId === draftRequestId ? draft.value : activeInsight?.content ?? ''
  const hasSideContent = Boolean(
    activeInsight && (activeInsight.suggestions.length > 0 || activeInsight.evidence.length > 0),
  )
  const canSubmit = useMemo(() => {
    if (!canGenerate || isPending || !siteCode) return false
    if (kind === 'session_explanation') return activeSessionId !== ''
    if (kind === 'work_order_summary') return activeWorkOrderId !== ''
    if (kind === 'congestion_risk') return Number.isFinite(parsedHorizon) && parsedHorizon > 0
    if (kind === 'station_qa') return question.trim() !== ''
    return true
  }, [activeSessionId, activeWorkOrderId, canGenerate, isPending, kind, parsedHorizon, question, siteCode])

  function submit() {
    if (!canSubmit || !siteCode) return
    if (kind === 'session_explanation') {
      onGenerate({ kind, sessionId: activeSessionId })
      return
    }
    if (kind === 'work_order_summary') {
      onGenerate({ kind, workOrderId: activeWorkOrderId })
      return
    }
    if (kind === 'congestion_risk') {
      onGenerate({ kind, siteId: siteCode, horizonHours: parsedHorizon })
      return
    }
    if (kind === 'daily_report') {
      onGenerate({ kind, siteId: siteCode, businessDate })
      return
    }
    onGenerate({ kind, siteId: siteCode, question: question.trim() })
  }

  return (
    <section className="ops-panel">
      <header>
        <h2>AI 运维助手</h2>
        {activeInsight ? <StatusBadge tone="charge">{formatCompactTime(activeInsight.generatedAt)}</StatusBadge> : null}
      </header>

      <div className="ops-ai">
        <div className="ops-ai-form">
          <label>
            类型
            <select
              aria-label="AI 类型"
              name="aiKind"
              disabled={!canGenerate || isPending}
              value={kind}
              onChange={(event) => setKind(event.target.value as AIInsightKind)}
            >
              {kindOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>

          {kind === 'session_explanation' ? (
            <label>
              会话
              <select
                aria-label="AI 会话"
                name="aiSession"
                disabled={!canGenerate || isPending}
                value={activeSessionId}
                onChange={(event) => setSessionId(event.target.value)}
              >
                {sessions.map((session) => (
                  <option key={session.id} value={session.sessionNo}>
                    {session.sessionNo} / {session.connectorCode}
                  </option>
                ))}
              </select>
            </label>
          ) : null}

          {kind === 'work_order_summary' ? (
            <label>
              工单
              <select
                aria-label="AI 工单"
                name="aiWorkOrder"
                disabled={!canGenerate || isPending}
                value={activeWorkOrderId}
                onChange={(event) => setWorkOrderId(event.target.value)}
              >
                {workOrders.map((order) => (
                  <option key={order.id} value={order.id}>
                    {order.workOrderNo} / {order.chargerCode}
                  </option>
                ))}
              </select>
            </label>
          ) : null}

          {kind === 'congestion_risk' ? (
            <label>
              小时
              <input
                aria-label="AI 预测小时"
                name="aiHorizonHours"
                disabled={!canGenerate || isPending}
                value={horizonHours}
                inputMode="numeric"
                onChange={(event) => setHorizonHours(event.target.value)}
              />
            </label>
          ) : null}

          {kind === 'daily_report' ? (
            <label>
              日期
              <input
                aria-label="AI 日报日期"
                name="aiBusinessDate"
                disabled={!canGenerate || isPending}
                type="date"
                value={businessDate}
                onChange={(event) => setBusinessDate(event.target.value)}
              />
            </label>
          ) : null}

          {kind === 'station_qa' ? (
            <label className="ops-ai-question">
              问题
              <input
                aria-label="AI 问题"
                name="aiQuestion"
                disabled={!canGenerate || isPending}
                value={question}
                onChange={(event) => setQuestion(event.target.value)}
              />
            </label>
          ) : null}

          <button type="button" disabled={!canSubmit} onClick={submit}>
            {kind === 'station_qa' ? (
              <Bot size={16} aria-hidden="true" />
            ) : (
              <SendHorizontal size={16} aria-hidden="true" />
            )}
            {isPending ? '生成中' : '生成'}
          </button>
        </div>

        {activeInsight ? (
          <div className={hasSideContent ? 'ops-ai-result' : 'ops-ai-result ops-ai-result--plain'}>
            <label>
              草稿
              <textarea
                aria-label="AI 草稿"
                name="aiDraft"
                value={draftValue}
                onChange={(event) => setDraft({ requestId: draftRequestId, value: event.target.value })}
              />
            </label>

            {hasSideContent ? (
              <div className="ops-ai-side">
                {activeInsight.suggestions.length > 0 ? (
                  <div>
                    <strong>建议</strong>
                    {activeInsight.suggestions.map((item) => (
                      <span key={item}>{item}</span>
                    ))}
                  </div>
                ) : null}
                {activeInsight.evidence.length > 0 ? (
                  <div>
                    <strong>依据</strong>
                    {activeInsight.evidence.map((item) => (
                      <span key={`${item.label}-${item.value}`}>
                        {item.label} / {item.value}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : null}

        {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
      </div>
    </section>
  )
}
