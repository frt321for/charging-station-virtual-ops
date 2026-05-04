import { useMemo, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { CommandPanel } from '../../components/operations/CommandPanel'
import { OverviewMetrics } from '../../components/operations/OverviewMetrics'
import { QueuePanel } from '../../components/operations/QueuePanel'
import { SessionTable } from '../../components/operations/SessionTable'
import { SimulatorPanel } from '../../components/operations/SimulatorPanel'
import { StatusBadge } from '../../components/operations/StatusBadge'
import { TopologyPanel } from '../../components/operations/TopologyPanel'
import { useOperationsSnapshot, useRemoteCommand, useSessionDetail } from '../../hooks/useOperations'
import { ApiError } from '../../services/api-client'
import type { CommandPathType } from '../../types/operations'
import { canRunCommand } from '../../utils/operations'

const navItems = [
  { id: 'ops-overview', label: '总览' },
  { id: 'ops-topology', label: '拓扑' },
  { id: 'ops-sessions', label: '会话' },
  { id: 'ops-command', label: '控制' },
  { id: 'ops-simulator', label: '模拟' },
  { id: 'ops-queue', label: '负载' },
  { id: 'ops-sla', label: 'SLA' },
]

export function OperationsPage() {
  const [siteCode, setSiteCode] = useState<string | undefined>()
  const [searchText, setSearchText] = useState('')
  const [selectedSessionNo, setSelectedSessionNo] = useState<string | undefined>()
  const [selectedConnectorCode, setSelectedConnectorCode] = useState<string | undefined>()
  const [targetPowerKw, setTargetPowerKw] = useState('3.5')
  const [activeSectionId, setActiveSectionId] = useState(navItems[0].id)

  const snapshotQuery = useOperationsSnapshot(siteCode)
  const snapshot = snapshotQuery.data
  const preferredSession =
    snapshot?.sessions.find((session) => session.status === 'charging') ?? snapshot?.sessions[0]
  const activeSessionNo = selectedSessionNo ?? (selectedConnectorCode ? undefined : preferredSession?.sessionNo)
  const activeConnectorCode = selectedConnectorCode ?? preferredSession?.connectorCode
  const detailQuery = useSessionDetail(activeSessionNo)
  const commandMutation = useRemoteCommand()
  const targetPowerKwNumber = Number(targetPowerKw)
  const isTargetPowerValid =
    targetPowerKw.trim() !== '' && Number.isFinite(targetPowerKwNumber) && targetPowerKwNumber > 0

  const filteredSessions = useMemo(() => {
    if (!snapshot) return []
    const query = searchText.trim().toLowerCase()
    if (!query) return snapshot.sessions

    return snapshot.sessions.filter(
      (session) =>
        session.sessionNo.toLowerCase().includes(query) ||
        session.connectorCode.toLowerCase().includes(query) ||
        session.chargerCode.toLowerCase().includes(query),
    )
  }, [searchText, snapshot])

  const selectedDetail =
    detailQuery.data ?? snapshot?.sessionDetails.find((detail) => detail.sessionNo === activeSessionNo)
  const errorMessage =
    commandMutation.error instanceof ApiError
      ? `${commandMutation.error.message} / ${commandMutation.error.details ?? commandMutation.error.code}`
      : commandMutation.error?.message
  const lastMessage = commandMutation.data
    ? `${commandMutation.data.commandNo} / ${commandMutation.data.status}`
    : undefined

  function selectSession(sessionNo: string) {
    const session = snapshot?.sessions.find((item) => item.sessionNo === sessionNo)
    setSelectedSessionNo(sessionNo)
    setSelectedConnectorCode(session?.connectorCode)
  }

  function selectConnector(connectorCode: string) {
    const session = snapshot?.sessions.find((item) => item.connectorCode === connectorCode)
    setSelectedConnectorCode(connectorCode)
    setSelectedSessionNo(session?.sessionNo)
  }

  function sendCommand(commandType: CommandPathType, sessionNo = activeSessionNo) {
    if (!sessionNo) return
    if (commandType === 'limit-power' && !isTargetPowerValid) return
    commandMutation.mutate({
      sessionId: sessionNo,
      commandType,
      requestedBy: 'station-manager',
      targetPowerKw: commandType === 'limit-power' ? targetPowerKwNumber : undefined,
    })
  }

  function activateSection(sectionId: string) {
    setActiveSectionId(sectionId)
    document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <main className="ops-shell">
      <header className="ops-topline">
        <div className="ops-brand">
          <strong>充电运营中台</strong>
          <span>{snapshot?.site.code ?? 'HQ-CAMPUS'} / OPS-02</span>
        </div>
        <nav className="ops-nav" aria-label="主导航">
          {navItems.map((item, index) => (
            <button
              key={item.id}
              type="button"
              aria-current={item.id === activeSectionId ? 'page' : undefined}
              onClick={() => activateSection(item.id)}
            >
              <span>{item.label}</span>
              <code>{(index + 1).toString().padStart(2, '0')}</code>
            </button>
          ))}
        </nav>
        <div className="ops-tools">
          <select
            aria-label="站点"
            name="siteCode"
            value={snapshot?.site.code ?? siteCode ?? ''}
            onChange={(event) => setSiteCode(event.target.value)}
          >
            {snapshot?.sites.map((site) => (
              <option key={site.id} value={site.code}>
                {site.campus}
              </option>
            ))}
          </select>
          <input
            aria-label="会话或桩编号"
            name="operationsSearch"
            value={searchText}
            placeholder="AC-N-001-01"
            onChange={(event) => setSearchText(event.target.value)}
          />
          <button type="button" onClick={() => void snapshotQuery.refetch()} disabled={snapshotQuery.isFetching}>
            <RefreshCw size={16} aria-hidden="true" className={snapshotQuery.isFetching ? 'ops-spin' : undefined} />
            刷新
          </button>
          <button
            type="button"
            className="ops-primary-button"
            disabled={!canRunCommand(selectedDetail?.status, 'limit-power') || !isTargetPowerValid}
            onClick={() => sendCommand('limit-power')}
          >
            下发命令
          </button>
        </div>
      </header>

      {snapshotQuery.error ? (
        <section className="ops-page-error" role="alert">
          <strong>运营数据加载失败</strong>
          <button type="button" onClick={() => void snapshotQuery.refetch()}>
            重试
          </button>
        </section>
      ) : (
        <div className="ops-main">
          <div className="ops-left">
            <div id="ops-overview">
              <OverviewMetrics snapshot={snapshot} isLoading={snapshotQuery.isLoading} />
            </div>
            <div id="ops-topology">
              <TopologyPanel
                snapshot={snapshot}
                selectedConnectorCode={activeConnectorCode}
                onSelectConnector={selectConnector}
              />
            </div>
            <div id="ops-sessions">
              <SessionTable
                sessions={filteredSessions}
                details={snapshot?.sessionDetails ?? []}
                selectedSessionNo={activeSessionNo}
                onSelectSession={selectSession}
                onStopSession={(sessionNo) => sendCommand('stop', sessionNo)}
              />
            </div>
          </div>
          <aside className="ops-right">
            <div id="ops-command">
            <CommandPanel
              session={selectedDetail}
              targetConnectorCode={activeConnectorCode}
              targetPowerKw={targetPowerKw}
              isTargetPowerValid={isTargetPowerValid}
              isPending={commandMutation.isPending}
                lastMessage={lastMessage}
                errorMessage={errorMessage}
                onTargetPowerChange={setTargetPowerKw}
                onCommand={sendCommand}
              />
            </div>
            <div id="ops-simulator">
              <SimulatorPanel />
            </div>
            <QueuePanel sessions={snapshot?.sessions ?? []} queueId="ops-queue" slaId="ops-sla" />
            {detailQuery.isFetching ? <StatusBadge tone="charge">同步中</StatusBadge> : null}
          </aside>
        </div>
      )}
    </main>
  )
}
