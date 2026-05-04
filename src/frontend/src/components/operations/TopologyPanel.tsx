import type { OperationsSnapshot } from '../../types/operations'
import { connectorStatusTone, findConnectorByCode, flattenChargers, formatNumber } from '../../utils/operations'
import { StatusBadge } from './StatusBadge'

interface TopologyPanelProps {
  snapshot?: OperationsSnapshot
  selectedConnectorCode?: string
  onSelectConnector: (connectorCode: string) => void
}

const connectorLabels: Record<string, string> = {
  available: '可用',
  reserved: '预约',
  plugged: '已插枪',
  charging: '充电中',
  fault: '故障',
  faulted: '故障',
  offline: '离线',
  unavailable: '不可用',
}

export function TopologyPanel({ snapshot, selectedConnectorCode, onSelectConnector }: TopologyPanelProps) {
  const chargers = flattenChargers(snapshot)
  const selected = findConnectorByCode(chargers, selectedConnectorCode)
  const primaryCharger =
    selected?.charger ?? chargers.find((charger) => charger.connectors.some((item) => item.status === 'charging')) ?? chargers[0]
  const otherAreas = snapshot?.topology?.areas.filter((area) =>
    area.groups.every((group) => !group.chargers.some((charger) => charger.id === primaryCharger?.id)),
  )

  return (
    <section className="ops-panel">
      <header>
        <h2>{primaryCharger ? `${primaryCharger.code} 拓扑` : '桩机拓扑'}</h2>
        <StatusBadge tone={primaryCharger?.status === 'charging' ? 'charge' : 'idle'}>
          {selectedConnectorCode ?? '未选择'}
        </StatusBadge>
      </header>
      <div className="ops-topology">
        {primaryCharger ? (
          <div className="ops-circuit">
            <div className="ops-circuit-head">
              <div>
                <h3>{primaryCharger.name}</h3>
                <span>
                  {formatNumber(primaryCharger.ratedPowerKw)} kW / {primaryCharger.chargerType.toUpperCase()}
                </span>
              </div>
              <span className="ops-mono">{primaryCharger.installationLocation}</span>
            </div>
            <div className="ops-ports">
              {primaryCharger.connectors.map((connector) => (
                <button
                  key={connector.id}
                  type="button"
                  className={
                    connector.code === selectedConnectorCode ? 'ops-port ops-port--selected' : 'ops-port'
                  }
                  onClick={() => onSelectConnector(connector.code)}
                >
                  <StatusBadge tone={connectorStatusTone(connector.status)}>
                    {`${connector.number.toString().padStart(2, '0')} ${
                      connectorLabels[connector.status] ?? connector.status
                    }`}
                  </StatusBadge>
                  <data>{formatNumber(connector.maxPowerKw)} kW</data>
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div className="ops-empty">暂无拓扑数据</div>
        )}

        <div className="ops-zone-stack">
          {(otherAreas ?? []).slice(0, 3).map((area) => {
            const activeCount = area.groups.reduce(
              (total, group) =>
                total +
                group.chargers.reduce(
                  (chargerTotal, charger) =>
                    chargerTotal +
                    charger.connectors.filter((connector) => connector.status === 'charging').length,
                  0,
                ),
              0,
            )

            return (
              <button
                key={area.id}
                type="button"
                className="ops-zone-row"
                onClick={() => {
                  const firstConnector = area.groups[0]?.chargers[0]?.connectors[0]
                  if (firstConnector) onSelectConnector(firstConnector.code)
                }}
              >
                <strong>{area.name}</strong>
                <span>
                  {formatNumber(area.loadLimitKw)} kW / {activeCount} 个会话
                </span>
              </button>
            )
          })}
        </div>
      </div>
    </section>
  )
}
