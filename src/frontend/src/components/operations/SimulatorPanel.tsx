import { useState } from 'react'
import { useSimulatorControl } from '../../hooks/useOperations'
import { ApiError } from '../../services/api-client'
import type { SimulatorRequest } from '../../types/operations'
import { StatusBadge } from './StatusBadge'

interface SimulatorPanelProps {
  enabled: boolean
}

export function SimulatorPanel({ enabled }: SimulatorPanelProps) {
  const [form, setForm] = useState({
    chargers: '5',
    connectors: '1',
    onlineRate: '0.80',
    faultRate: '0.05',
    curve: 'commute',
  })
  const { statusQuery, onceMutation, startMutation, stopMutation } = useSimulatorControl()
  const isPending = onceMutation.isPending || startMutation.isPending || stopMutation.isPending
  const request = buildRequest(form)
  const isValid = Boolean(request)
  const status = statusQuery.data
  const stateLabel = status?.running ? '运行中' : isPending ? '执行中' : '空闲'
  const mutationError = onceMutation.error ?? startMutation.error ?? stopMutation.error
  const errorMessage =
    mutationError instanceof ApiError
      ? `${mutationError.message} / ${mutationError.details ?? mutationError.code}`
      : mutationError?.message ?? status?.lastError
  const resultMessage = status
    ? status.running
      ? `${status.config.chargerCount} 台 / ${status.config.loadCurve}`
      : `生成 ${status.generated} 台`
    : undefined

  function runOnce() {
    if (enabled && request) onceMutation.mutate(request)
  }

  function startLoop() {
    if (enabled && request) startMutation.mutate(request)
  }

  return (
    <section className="ops-panel">
      <header>
        <h2>模拟器</h2>
        <StatusBadge tone={status?.running ? 'charge' : 'ok'}>{stateLabel}</StatusBadge>
      </header>
      <div className="ops-sim-body">
        <label>
          桩数量
          <input
            value={form.chargers}
            inputMode="numeric"
            aria-label="桩数量"
            name="simChargers"
            onChange={(event) => setForm((value) => ({ ...value, chargers: event.target.value }))}
          />
        </label>
        <label>
          枪口数
          <input
            value={form.connectors}
            inputMode="numeric"
            aria-label="枪口数"
            name="simConnectors"
            onChange={(event) => setForm((value) => ({ ...value, connectors: event.target.value }))}
          />
        </label>
        <label>
          在线率
          <input
            value={form.onlineRate}
            inputMode="decimal"
            aria-label="在线率"
            name="simOnlineRate"
            onChange={(event) => setForm((value) => ({ ...value, onlineRate: event.target.value }))}
          />
        </label>
        <label>
          故障率
          <input
            value={form.faultRate}
            inputMode="decimal"
            aria-label="故障率"
            name="simFaultRate"
            onChange={(event) => setForm((value) => ({ ...value, faultRate: event.target.value }))}
          />
        </label>
        <label>
          曲线
          <select
            aria-label="负载曲线"
            name="simCurve"
            value={form.curve}
            onChange={(event) => setForm((value) => ({ ...value, curve: event.target.value }))}
          >
            <option value="commute">commute</option>
            <option value="flat">flat</option>
            <option value="random">random</option>
          </select>
        </label>
      </div>
      <div className="ops-sim-actions">
        <button type="button" className="ops-primary-button" disabled={!enabled || !isValid || isPending} onClick={runOnce}>
          运行一次
        </button>
        <button type="button" disabled={!enabled || !isValid || isPending || status?.running} onClick={startLoop}>
          持续运行
        </button>
        <button
          type="button"
          className="ops-danger-button"
          disabled={!enabled || isPending || !status?.running}
          onClick={() => stopMutation.mutate()}
        >
          停止
        </button>
      </div>
      {resultMessage ? <output className="ops-inline-result">{resultMessage}</output> : null}
      {!isValid ? <output className="ops-inline-error">参数不合法</output> : null}
      {errorMessage ? <output className="ops-inline-error">{errorMessage}</output> : null}
    </section>
  )
}

function buildRequest(form: {
  chargers: string
  connectors: string
  onlineRate: string
  faultRate: string
  curve: string
}): SimulatorRequest | undefined {
  const chargerCount = Number(form.chargers)
  const connectorCount = Number(form.connectors)
  const onlineRate = Number(form.onlineRate)
  const faultRate = Number(form.faultRate)
  const loadCurve = form.curve

  if (
    form.chargers.trim() === '' ||
    form.connectors.trim() === '' ||
    form.onlineRate.trim() === '' ||
    form.faultRate.trim() === ''
  ) {
    return undefined
  }
  if (!Number.isInteger(chargerCount) || chargerCount <= 0) return undefined
  if (!Number.isInteger(connectorCount) || connectorCount <= 0) return undefined
  if (!Number.isFinite(onlineRate) || onlineRate < 0 || onlineRate > 1) return undefined
  if (!Number.isFinite(faultRate) || faultRate < 0 || faultRate > 1) return undefined
  if (loadCurve !== 'commute' && loadCurve !== 'flat' && loadCurve !== 'random') return undefined

  return {
    chargerCount,
    connectorCount,
    onlineRate,
    faultRate,
    loadCurve,
  }
}
