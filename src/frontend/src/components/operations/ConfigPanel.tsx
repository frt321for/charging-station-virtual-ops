import { useMemo, useState, type FormEvent } from 'react'
import type {
  ConfigEntity,
  ConfigSnapshot,
  ConfigUpdateRequest,
  CreatePricingPolicyRequest,
} from '../../types/operations'

interface ConfigPanelProps {
  snapshot?: ConfigSnapshot
  enabled: boolean
  isLoading: boolean
  isPending: boolean
  errorMessage?: string
  onUpdate: (request: ConfigUpdateRequest) => void
  onCreatePricingPolicy: (request: CreatePricingPolicyRequest) => void
}

export function ConfigPanel({
  snapshot,
  enabled,
  isLoading,
  isPending,
  errorMessage,
  onUpdate,
  onCreatePricingPolicy,
}: ConfigPanelProps) {
  const [areaId, setAreaId] = useState('')
  const [groupId, setGroupId] = useState('')
  const [chargerId, setChargerId] = useState('')
  const [connectorId, setConnectorId] = useState('')
  const [loadPolicyId, setLoadPolicyId] = useState('')
  const [reservationRuleId, setReservationRuleId] = useState('')
  const [queueRuleId, setQueueRuleId] = useState('')

  const area = snapshot?.areas.find((item) => item.id === areaId) ?? snapshot?.areas[0]
  const group = snapshot?.groups.find((item) => item.id === groupId) ?? snapshot?.groups[0]
  const charger = snapshot?.chargers.find((item) => item.id === chargerId) ?? snapshot?.chargers[0]
  const connector = snapshot?.connectors.find((item) => item.id === connectorId) ?? snapshot?.connectors[0]
  const loadPolicy =
    snapshot?.loadPolicies.find((item) => item.id === loadPolicyId) ?? snapshot?.loadPolicies[0]
  const reservationRule =
    snapshot?.reservationRules.find((item) => item.id === reservationRuleId) ?? snapshot?.reservationRules[0]
  const queueRule = snapshot?.queueRules.find((item) => item.id === queueRuleId) ?? snapshot?.queueRules[0]
  const latestPricingPolicy = snapshot?.pricingPolicies[0]

  const pricingDefaults = useMemo(() => {
    const periods = latestPricingPolicy?.periods ?? []
    return {
      code: latestPricingPolicy?.code ?? 'HQ-STD',
      name: latestPricingPolicy ? `${latestPricingPolicy.name} V${latestPricingPolicy.version + 1}` : '标准计价',
      chargerType: latestPricingPolicy?.chargerType ?? 'all',
      periodA: periods[0],
      periodB: periods[1],
      periodC: periods[2],
    }
  }, [latestPricingPolicy])

  if (isLoading) {
    return <section className="ops-panel ops-config-panel">同步中</section>
  }

  if (!snapshot) {
    return <section className="ops-panel ops-config-panel">无配置数据</section>
  }

  const config = snapshot
  const locked = !enabled || isPending

  function submitSite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    onUpdate({
      entity: 'site',
      id: config.site.id,
      body: {
        name: text(data, 'name'),
        campus: text(data, 'campus'),
        capacityKw: number(data, 'capacityKw'),
        loadLimitKw: number(data, 'loadLimitKw'),
        slaResponseMinutes: number(data, 'slaResponseMinutes'),
        slaRecoveryMinutes: number(data, 'slaRecoveryMinutes'),
        status: text(data, 'status'),
      },
    })
  }

  function submitSelected(entity: ConfigEntity, id: string | undefined, event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!id) return
    onUpdate({ entity, id, body: formBody(new FormData(event.currentTarget)) })
  }

  function submitPricing(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    onCreatePricingPolicy({
      siteId: config.site.id,
      code: text(data, 'code'),
      name: text(data, 'name'),
      chargerType: text(data, 'chargerType'),
      status: text(data, 'status') as CreatePricingPolicyRequest['status'],
      periods: [
        period(data, 'a', pricingDefaults.periodA?.label ?? '平段', 0, 420),
        period(data, 'b', pricingDefaults.periodB?.label ?? '峰段', 420, 1380),
        period(data, 'c', pricingDefaults.periodC?.label ?? '谷段', 1380, 1440),
      ],
    })
  }

  return (
    <section className="ops-config-panel">
      {errorMessage ? <div className="ops-inline-error">{errorMessage}</div> : null}
      <div className="ops-config-grid">
        <form className="ops-panel ops-form-grid" onSubmit={submitSite}>
          <h2>站点 / SLA</h2>
          <label>
            名称
            <input name="name" defaultValue={snapshot.site.name} disabled={locked} />
          </label>
          <label>
            园区
            <input name="campus" defaultValue={snapshot.site.campus} disabled={locked} />
          </label>
          <label>
            容量 kW
            <input name="capacityKw" type="number" step="0.1" defaultValue={snapshot.site.capacityKw} disabled={locked} />
          </label>
          <label>
            上限 kW
            <input name="loadLimitKw" type="number" step="0.1" defaultValue={snapshot.site.loadLimitKw} disabled={locked} />
          </label>
          <label>
            响应 min
            <input name="slaResponseMinutes" type="number" defaultValue={snapshot.site.slaResponseMinutes} disabled={locked} />
          </label>
          <label>
            恢复 min
            <input name="slaRecoveryMinutes" type="number" defaultValue={snapshot.site.slaRecoveryMinutes} disabled={locked} />
          </label>
          <label>
            状态
            <select name="status" defaultValue={snapshot.site.status} disabled={locked}>
              <option value="active">active</option>
              <option value="inactive">inactive</option>
            </select>
          </label>
          <button type="submit" disabled={locked}>保存站点</button>
        </form>

        <EditableBlock
          title="区域"
          entity="area"
          selectedId={area?.id}
          options={snapshot.areas.map(option)}
          onSelect={setAreaId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'name', label: '名称', value: area?.name },
            { name: 'loadLimitKw', label: '上限 kW', value: area?.loadLimitKw, type: 'number', step: '0.1' },
            { name: 'sortOrder', label: '排序', value: area?.sortOrder, type: 'number' },
            { name: 'status', label: '状态', value: area?.status, type: 'status' },
          ]}
        />

        <EditableBlock
          title="桩组"
          entity="group"
          selectedId={group?.id}
          options={snapshot.groups.map(option)}
          onSelect={setGroupId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'name', label: '名称', value: group?.name },
            { name: 'electricalNode', label: '回路', value: group?.electricalNode },
            { name: 'loadLimitKw', label: '上限 kW', value: group?.loadLimitKw, type: 'number', step: '0.1' },
            { name: 'priority', label: '优先级', value: group?.priority, type: 'number' },
            { name: 'status', label: '状态', value: group?.status, type: 'status' },
          ]}
        />

        <EditableBlock
          title="桩机"
          entity="charger"
          selectedId={charger?.id}
          options={snapshot.chargers.map(option)}
          onSelect={setChargerId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'name', label: '名称', value: charger?.name },
            { name: 'ratedPowerKw', label: '额定 kW', value: charger?.ratedPowerKw, type: 'number', step: '0.1' },
            { name: 'installationLocation', label: '位置', value: charger?.installationLocation },
            { name: 'maintenanceTag', label: '维护标签', value: charger?.maintenanceTag },
            { name: 'status', label: '状态', value: charger?.status, type: 'chargerStatus' },
          ]}
        />

        <EditableBlock
          title="枪口"
          entity="connector"
          selectedId={connector?.id}
          options={snapshot.connectors.map((item) => ({ id: item.id, label: `${item.code} / ${item.number}` }))}
          onSelect={setConnectorId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'maxPowerKw', label: '最大 kW', value: connector?.maxPowerKw, type: 'number', step: '0.1' },
            { name: 'status', label: '状态', value: connector?.status, type: 'connectorStatus' },
          ]}
        />

        <EditableBlock
          title="负载策略"
          entity="loadPolicy"
          selectedId={loadPolicy?.id}
          options={snapshot.loadPolicies.map((item) => ({ id: item.id, label: `${item.scopeCode} / v${item.version}` }))}
          onSelect={setLoadPolicyId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'thresholdKw', label: '阈值 kW', value: loadPolicy?.thresholdKw, type: 'number', step: '0.1' },
            { name: 'warningKw', label: '预警 kW', value: loadPolicy?.warningKw, type: 'number', step: '0.1' },
            { name: 'actionMode', label: '动作', value: loadPolicy?.actionMode, type: 'loadAction' },
            { name: 'status', label: '状态', value: loadPolicy?.status, type: 'policyStatus' },
          ]}
        />

        <EditableBlock
          title="预约规则"
          entity="reservationRule"
          selectedId={reservationRule?.id}
          options={snapshot.reservationRules.map(option)}
          onSelect={setReservationRuleId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'name', label: '名称', value: reservationRule?.name },
            { name: 'holdMinutes', label: '保留 min', value: reservationRule?.holdMinutes, type: 'number' },
            { name: 'timeoutAction', label: '超时', value: reservationRule?.timeoutAction, type: 'timeoutAction' },
            { name: 'status', label: '状态', value: reservationRule?.status, type: 'policyStatus' },
          ]}
        />

        <EditableBlock
          title="排队规则"
          entity="queueRule"
          selectedId={queueRule?.id}
          options={snapshot.queueRules.map(option)}
          onSelect={setQueueRuleId}
          onSubmit={submitSelected}
          locked={locked}
          fields={[
            { name: 'name', label: '名称', value: queueRule?.name },
            { name: 'strategy', label: '策略', value: queueRule?.strategy, type: 'queueStrategy' },
            { name: 'maxQueueSize', label: '队列数', value: queueRule?.maxQueueSize, type: 'number' },
            { name: 'priorityFactor', label: '优先因子', value: queueRule?.priorityFactor },
            { name: 'status', label: '状态', value: queueRule?.status, type: 'policyStatus' },
          ]}
        />

        <form className="ops-panel ops-form-grid ops-config-wide" onSubmit={submitPricing}>
          <h2>价格版本</h2>
          <label>
            编码
            <input name="code" defaultValue={pricingDefaults.code} disabled={locked} />
          </label>
          <label>
            名称
            <input name="name" defaultValue={pricingDefaults.name} disabled={locked} />
          </label>
          <label>
            桩型
            <select name="chargerType" defaultValue={pricingDefaults.chargerType} disabled={locked}>
              <option value="all">all</option>
              <option value="ac">ac</option>
              <option value="dc">dc</option>
            </select>
          </label>
          <label>
            状态
            <select name="status" defaultValue="active" disabled={locked}>
              <option value="draft">draft</option>
              <option value="active">active</option>
              <option value="retired">retired</option>
            </select>
          </label>
          {(['a', 'b', 'c'] as const).map((key, index) => {
            const defaults = [pricingDefaults.periodA, pricingDefaults.periodB, pricingDefaults.periodC][index]
            return (
              <fieldset key={key} className="ops-price-period">
                <legend>{defaults?.label ?? `时段 ${index + 1}`}</legend>
                <input
                  aria-label="时段名称"
                  name={`${key}Label`}
                  defaultValue={defaults?.label ?? ''}
                  disabled={locked}
                />
                <input
                  aria-label="开始分钟"
                  name={`${key}Start`}
                  type="number"
                  defaultValue={defaults?.startMinute ?? 0}
                  disabled={locked}
                />
                <input
                  aria-label="结束分钟"
                  name={`${key}End`}
                  type="number"
                  defaultValue={defaults?.endMinute ?? 1440}
                  disabled={locked}
                />
                <input
                  aria-label="电费单价"
                  name={`${key}Energy`}
                  type="number"
                  step="0.0001"
                  defaultValue={defaults?.energyPricePerKwh ?? 0}
                  disabled={locked}
                />
                <input
                  aria-label="服务费单价"
                  name={`${key}Service`}
                  type="number"
                  step="0.0001"
                  defaultValue={defaults?.serviceFeePerKwh ?? 0}
                  disabled={locked}
                />
                <input
                  aria-label="占用费单价"
                  name={`${key}Occupancy`}
                  type="number"
                  step="0.0001"
                  defaultValue={defaults?.occupancyFeePerMinute ?? 0}
                  disabled={locked}
                />
              </fieldset>
            )
          })}
          <button type="submit" disabled={locked}>新建价格版本</button>
        </form>
      </div>
    </section>
  )
}

interface FieldSpec {
  name: string
  label: string
  value?: string | number
  type?: string
  step?: string
}

function EditableBlock(props: {
  title: string
  entity: ConfigEntity
  selectedId?: string
  options: Array<{ id: string; label: string }>
  fields: FieldSpec[]
  locked: boolean
  onSelect: (id: string) => void
  onSubmit: (entity: ConfigEntity, id: string | undefined, event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <form key={props.selectedId ?? props.title} className="ops-panel ops-form-grid" onSubmit={(event) => props.onSubmit(props.entity, props.selectedId, event)}>
      <h2>{props.title}</h2>
      <label>
        对象
        <select value={props.selectedId ?? ''} onChange={(event) => props.onSelect(event.target.value)} disabled={props.locked || props.options.length === 0}>
          {props.options.map((item) => (
            <option key={item.id} value={item.id}>{item.label}</option>
          ))}
        </select>
      </label>
      {props.fields.map((field) => (
        <label key={field.name}>
          {field.label}
          {selectOptions(field.type) ? (
            <select name={field.name} defaultValue={String(field.value ?? '')} disabled={props.locked}>
              {selectOptions(field.type)?.map((value) => <option key={value} value={value}>{value}</option>)}
            </select>
          ) : (
            <input name={field.name} type={field.type === 'number' ? 'number' : 'text'} step={field.step} defaultValue={field.value ?? ''} disabled={props.locked} />
          )}
        </label>
      ))}
      <button type="submit" disabled={props.locked || !props.selectedId}>保存{props.title}</button>
    </form>
  )
}

function selectOptions(type?: string) {
  switch (type) {
    case 'status':
      return ['active', 'inactive']
    case 'policyStatus':
      return ['draft', 'active', 'retired']
    case 'chargerStatus':
      return ['unregistered', 'available', 'occupied', 'charging', 'fault', 'offline', 'maintenance', 'disabled']
    case 'connectorStatus':
      return ['available', 'reserved', 'plugged', 'charging', 'fault', 'offline', 'maintenance', 'disabled']
    case 'loadAction':
      return ['limit_power', 'pause', 'queue', 'reject']
    case 'timeoutAction':
      return ['release', 'promote_queue', 'cancel']
    case 'queueStrategy':
      return ['reservation_time', 'member_priority', 'load_priority']
    default:
      return undefined
  }
}

function option(item: { id: string; code: string; name: string }) {
  return { id: item.id, label: `${item.code} / ${item.name}` }
}

function formBody(data: FormData) {
  const body: Record<string, string | number> = {}
  data.forEach((value, key) => {
    const raw = String(value).trim()
    if (raw === '') return
    body[key] = Number.isNaN(Number(raw)) ? raw : Number(raw)
  })
  return body
}

function text(data: FormData, key: string) {
  return String(data.get(key) ?? '').trim()
}

function number(data: FormData, key: string) {
  return Number(data.get(key) ?? 0)
}

function period(data: FormData, key: 'a' | 'b' | 'c', fallbackLabel: string, start: number, end: number) {
  return {
    label: text(data, `${key}Label`) || fallbackLabel,
    startMinute: number(data, `${key}Start`) || start,
    endMinute: number(data, `${key}End`) || end,
    energyPricePerKwh: number(data, `${key}Energy`),
    serviceFeePerKwh: number(data, `${key}Service`),
    occupancyFeePerMinute: number(data, `${key}Occupancy`),
  }
}
