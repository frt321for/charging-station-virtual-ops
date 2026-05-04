import { useMemo, useState } from 'react'
import { LogOut, RefreshCw } from 'lucide-react'
import { useAuth } from '../../context/useAuth'
import { AccessPanel } from '../../components/operations/AccessPanel'
import { AIAssistantPanel } from '../../components/operations/AIAssistantPanel'
import { AuditLogPanel } from '../../components/operations/AuditLogPanel'
import { BillingDraftPanel } from '../../components/operations/BillingDraftPanel'
import { CommandPanel } from '../../components/operations/CommandPanel'
import { ConfigPanel } from '../../components/operations/ConfigPanel'
import { LoadControlPanel } from '../../components/operations/LoadControlPanel'
import { MaintenancePanel } from '../../components/operations/MaintenancePanel'
import { OverviewMetrics } from '../../components/operations/OverviewMetrics'
import { ReconciliationPanel } from '../../components/operations/ReconciliationPanel'
import { SessionDetailPanel } from '../../components/operations/SessionDetailPanel'
import { SessionTable } from '../../components/operations/SessionTable'
import { SimulatorPanel } from '../../components/operations/SimulatorPanel'
import { StatusBadge } from '../../components/operations/StatusBadge'
import { TopologyPanel } from '../../components/operations/TopologyPanel'
import {
  useConfirmBillingDraft,
  useAIAssistant,
  useCreateBillingCorrection,
  useCreateLoadControlRecord,
  useCreatePricingPolicy,
  useGenerateBillingDraft,
  useConfigSnapshot,
  useCreateWorkOrder,
  useExportReconciliation,
  useOperationsSnapshot,
  useReviewReconciliationException,
  useRemoteCommand,
  useSessionDetail,
  useTransitionWorkOrder,
  useUpdateConfig,
  useWorkOrderEvents,
} from '../../hooks/useOperations'
import { ApiError } from '../../services/api-client'
import type {
  CommandPathType,
  ConfirmBillRequest,
  CreateCorrectionRequest,
  CreateWorkOrderRequest,
  ReviewExceptionRequest,
  TransitionWorkOrderRequest,
} from '../../types/operations'
import { Permission } from '../../types/auth'

const navItems = [
  { id: 'overview', label: '总览', permission: Permission.SitesRead },
  { id: 'topology', label: '拓扑', permission: Permission.SitesRead },
  { id: 'sessions', label: '会话', permission: Permission.SessionsRead },
  { id: 'command', label: '控制', permission: Permission.CommandsWrite },
  { id: 'simulator', label: '模拟', permission: Permission.SimulatorControl },
  { id: 'load-control', label: '负载', permission: Permission.LoadControlRead },
  { id: 'billing', label: '计费', permission: Permission.BillingRead },
  { id: 'maintenance', label: '维保', permission: Permission.MaintenanceRead },
  { id: 'reconciliation', label: '核查', permission: Permission.BillingRead },
  { id: 'ai', label: 'AI', permission: Permission.AIRead },
  { id: 'users', label: '用户', permission: Permission.UsersManage },
  { id: 'audit', label: '审计', permission: Permission.AuditRead },
  { id: 'config', label: '配置', permission: Permission.ConfigManage },
] as const

type OperationsView = (typeof navItems)[number]['id']

export function OperationsPage() {
  const { user, hasPermission, logout } = useAuth()
  const [siteCode, setSiteCode] = useState<string | undefined>()
  const [searchText, setSearchText] = useState('')
  const [selectedSessionNo, setSelectedSessionNo] = useState<string | undefined>()
  const [selectedConnectorCode, setSelectedConnectorCode] = useState<string | undefined>()
  const [selectedWorkOrderId, setSelectedWorkOrderId] = useState<string | undefined>()
  const [targetPowerKw, setTargetPowerKw] = useState('3.5')
  const [requestedView, setRequestedView] = useState<OperationsView>('overview')

  const operatorName = user?.displayName ?? user?.username ?? 'operations'
  const primaryRoleCode = user?.roles[0]?.code ?? ''
  const canWriteCommands = hasPermission(Permission.CommandsWrite)
  const canRunSimulator = hasPermission(Permission.SimulatorControl)
  const canWriteLoadControl = hasPermission(Permission.LoadControlWrite)
  const canGenerateBilling = hasPermission(Permission.BillingGenerate)
  const canReviewFinance = hasPermission(Permission.FinanceReview)
  const canWriteMaintenance = hasPermission(Permission.MaintenanceWrite)
  const canReadAI = hasPermission(Permission.AIRead)
  const canManageUsers = hasPermission(Permission.UsersManage)
  const canReadAudit = hasPermission(Permission.AuditRead)
  const canManageConfig = hasPermission(Permission.ConfigManage)
  const allowedNavItems = useMemo(
    () => navItems.filter((item) => hasPermission(item.permission)),
    [hasPermission],
  )
  const snapshotAccess = useMemo(
    () => ({
      canReadBilling: hasPermission(Permission.BillingRead),
      canReadLoadControl: hasPermission(Permission.LoadControlRead),
      canReadMaintenance: hasPermission(Permission.MaintenanceRead),
      canReadSessions: hasPermission(Permission.SessionsRead),
    }),
    [hasPermission],
  )

  const activeView = allowedNavItems.some((item) => item.id === requestedView)
    ? requestedView
    : allowedNavItems[0]?.id ?? 'overview'

  const snapshotQuery = useOperationsSnapshot(siteCode, snapshotAccess)
  const snapshot = snapshotQuery.data
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

  const preferredSession =
    snapshot?.sessions.find((session) => session.status === 'charging') ?? snapshot?.sessions[0]
  const candidateSessionNo = selectedSessionNo ?? (selectedConnectorCode ? undefined : preferredSession?.sessionNo)
  const fallbackSession =
    activeView === 'sessions' &&
    searchText.trim() !== '' &&
    !filteredSessions.some((session) => session.sessionNo === candidateSessionNo)
      ? filteredSessions[0]
      : undefined
  const activeSessionNo = fallbackSession?.sessionNo ?? candidateSessionNo
  const activeConnectorCode = fallbackSession?.connectorCode ?? selectedConnectorCode ?? preferredSession?.connectorCode
  const detailQuery = useSessionDetail(activeSessionNo)
  const commandMutation = useRemoteCommand()
  const billingMutation = useGenerateBillingDraft()
  const loadControlMutation = useCreateLoadControlRecord()
  const createWorkOrderMutation = useCreateWorkOrder()
  const transitionWorkOrderMutation = useTransitionWorkOrder()
  const workOrderEventsQuery = useWorkOrderEvents(
    canWriteMaintenance || hasPermission(Permission.MaintenanceRead)
      ? selectedWorkOrderId ?? snapshot?.maintenance?.workOrders[0]?.id
      : undefined,
  )
  const reviewExceptionMutation = useReviewReconciliationException()
  const correctionMutation = useCreateBillingCorrection()
  const confirmBillMutation = useConfirmBillingDraft()
  const exportReconciliationMutation = useExportReconciliation()
  const aiMutation = useAIAssistant()
  const configQuery = useConfigSnapshot(snapshot?.site.code, canManageConfig)
  const updateConfigMutation = useUpdateConfig()
  const createPricingPolicyMutation = useCreatePricingPolicy()
  const targetPowerKwNumber = Number(targetPowerKw)
  const isTargetPowerValid =
    targetPowerKw.trim() !== '' && Number.isFinite(targetPowerKwNumber) && targetPowerKwNumber > 0

  const selectedDetail =
    detailQuery.data ?? snapshot?.sessionDetails.find((detail) => detail.sessionNo === activeSessionNo)
  const errorMessage =
    commandMutation.error instanceof ApiError
      ? `${commandMutation.error.message} / ${commandMutation.error.details ?? commandMutation.error.code}`
      : commandMutation.error?.message
  const billingErrorMessage =
    billingMutation.error instanceof ApiError
      ? `${billingMutation.error.message} / ${billingMutation.error.details ?? billingMutation.error.code}`
      : billingMutation.error?.message
  const loadControlErrorMessage =
    loadControlMutation.error instanceof ApiError
      ? `${loadControlMutation.error.message} / ${
          loadControlMutation.error.details ?? loadControlMutation.error.code
        }`
      : loadControlMutation.error?.message
  const maintenanceErrorMessage =
    firstApiError(createWorkOrderMutation.error, transitionWorkOrderMutation.error) ??
    createWorkOrderMutation.error?.message ??
    transitionWorkOrderMutation.error?.message
  const financeErrorMessage =
    firstApiError(
      reviewExceptionMutation.error,
      correctionMutation.error,
      confirmBillMutation.error,
      exportReconciliationMutation.error,
    ) ??
    reviewExceptionMutation.error?.message ??
    correctionMutation.error?.message ??
    confirmBillMutation.error?.message ??
    exportReconciliationMutation.error?.message
  const aiErrorMessage =
    aiMutation.error instanceof ApiError
      ? `${aiMutation.error.message} / ${aiMutation.error.details ?? aiMutation.error.code}`
      : aiMutation.error?.message
  const configErrorMessage =
    firstApiError(updateConfigMutation.error, createPricingPolicyMutation.error, configQuery.error) ??
    updateConfigMutation.error?.message ??
    createPricingPolicyMutation.error?.message ??
    configQuery.error?.message
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
    if (!canWriteCommands || !sessionNo) return
    if (commandType === 'limit-power' && !isTargetPowerValid) return
    commandMutation.mutate({
      sessionId: sessionNo,
      commandType,
      requestedBy: operatorName,
      targetPowerKw: commandType === 'limit-power' ? targetPowerKwNumber : undefined,
    })
  }

  function generateDraft(sessionNo: string) {
    if (!canGenerateBilling) return
    billingMutation.mutate({ sessionId: sessionNo, generatedBy: operatorName })
  }

  function createWorkOrder(faultId: string, request: CreateWorkOrderRequest) {
    if (!canWriteMaintenance) return
    createWorkOrderMutation.mutate({ faultId, body: request })
  }

  function transitionWorkOrder(workOrderId: string, request: TransitionWorkOrderRequest) {
    if (!canWriteMaintenance) return
    setSelectedWorkOrderId(workOrderId)
    transitionWorkOrderMutation.mutate({ workOrderId, body: request })
  }

  function reviewException(exceptionId: string, request: ReviewExceptionRequest) {
    if (!canReviewFinance) return
    reviewExceptionMutation.mutate({ exceptionId, body: request })
  }

  function createCorrection(exceptionId: string, request: CreateCorrectionRequest) {
    if (!canReviewFinance) return
    correctionMutation.mutate({ exceptionId, body: request })
  }

  function confirmBill(billId: string, request: ConfirmBillRequest) {
    if (!canReviewFinance) return
    confirmBillMutation.mutate({ billId, body: request })
  }

  function exportReconciliation(siteCode: string) {
    if (!canReviewFinance) return
    exportReconciliationMutation.mutate({ siteId: siteCode, generatedBy: operatorName })
  }

  return (
    <main className="ops-shell">
      <header className="ops-topline">
        <div className="ops-headbar">
          <div className="ops-brand">
            <strong>充电运营中台</strong>
            <span>{snapshot?.site.code ?? 'HQ-CAMPUS'}</span>
          </div>
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
            <div className="ops-user">
              <span>{operatorName}</span>
              <code>{primaryRoleCode}</code>
              <button type="button" aria-label="退出登录" onClick={() => void logout()}>
                <LogOut size={16} aria-hidden="true" />
                退出
              </button>
            </div>
          </div>
        </div>
        <nav className="ops-nav" aria-label="主导航">
          {allowedNavItems.map((item) => (
            <button
              key={item.id}
              type="button"
              aria-current={item.id === activeView ? 'page' : undefined}
              onClick={() => setRequestedView(item.id)}
            >
              <span>{item.label}</span>
            </button>
          ))}
        </nav>
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
          {activeView === 'overview' ? (
            <section className="ops-view">
              <OverviewMetrics snapshot={snapshot} isLoading={snapshotQuery.isLoading} />
            </section>
          ) : null}

          {activeView === 'topology' ? (
            <section className="ops-view">
              <TopologyPanel
                snapshot={snapshot}
                selectedConnectorCode={activeConnectorCode}
                onSelectConnector={selectConnector}
              />
            </section>
          ) : null}

          {activeView === 'sessions' ? (
            <section className="ops-workspace ops-workspace--split">
              <div className="ops-workspace-main">
                <SessionTable
                  sessions={filteredSessions}
                  details={snapshot?.sessionDetails ?? []}
                  selectedSessionNo={activeSessionNo}
                  onSelectSession={selectSession}
                  onStopSession={(sessionNo) => sendCommand('stop', sessionNo)}
                  canStopSessions={canWriteCommands}
                />
              </div>
              <aside className="ops-workspace-aside">
                <SessionDetailPanel
                  session={selectedDetail}
                  drafts={snapshot?.billingDrafts ?? []}
                  isFetching={detailQuery.isFetching}
                />
              </aside>
            </section>
          ) : null}

          {activeView === 'command' ? (
            <section className="ops-workspace ops-workspace--split ops-workspace--reverse">
              <div className="ops-workspace-main">
                <CommandPanel
                  session={selectedDetail}
                  targetConnectorCode={activeConnectorCode}
                  targetPowerKw={targetPowerKw}
                  isTargetPowerValid={isTargetPowerValid}
                  isPending={commandMutation.isPending}
                  lastMessage={lastMessage}
                  errorMessage={errorMessage}
                  canWrite={canWriteCommands}
                  onTargetPowerChange={setTargetPowerKw}
                  onCommand={sendCommand}
                />
                {detailQuery.isFetching ? <StatusBadge tone="charge">同步中</StatusBadge> : null}
              </div>
              <aside className="ops-workspace-aside">
                <SessionTable
                  sessions={filteredSessions}
                  details={snapshot?.sessionDetails ?? []}
                  selectedSessionNo={activeSessionNo}
                  onSelectSession={selectSession}
                  onStopSession={(sessionNo) => sendCommand('stop', sessionNo)}
                  canStopSessions={canWriteCommands}
                />
              </aside>
            </section>
          ) : null}

          {activeView === 'simulator' ? (
            <section className="ops-view">
              <SimulatorPanel enabled={canRunSimulator} />
            </section>
          ) : null}

          {activeView === 'load-control' ? (
            <section className="ops-view">
              <LoadControlPanel
                site={snapshot?.site}
                snapshot={snapshot?.loadControl}
                isPending={loadControlMutation.isPending}
                lastRecord={loadControlMutation.data}
                errorMessage={loadControlErrorMessage}
                canWrite={canWriteLoadControl}
                operatorName={operatorName}
                onCreateRecord={(request) => loadControlMutation.mutate(request)}
              />
            </section>
          ) : null}

          {activeView === 'billing' ? (
            <section className="ops-view">
              <BillingDraftPanel
                sessions={snapshot?.sessions ?? []}
                policies={snapshot?.pricingPolicies ?? []}
                drafts={snapshot?.billingDrafts ?? []}
                isPending={billingMutation.isPending}
                lastDraft={billingMutation.data}
                errorMessage={billingErrorMessage}
                canGenerate={canGenerateBilling}
                onGenerate={generateDraft}
              />
            </section>
          ) : null}

          {activeView === 'maintenance' ? (
            <section className="ops-view">
              <MaintenancePanel
                snapshot={snapshot?.maintenance}
                events={workOrderEventsQuery.data ?? []}
                selectedWorkOrderId={selectedWorkOrderId}
                isEventsFetching={workOrderEventsQuery.isFetching}
                isCreatePending={createWorkOrderMutation.isPending}
                isTransitionPending={transitionWorkOrderMutation.isPending}
                lastWorkOrder={transitionWorkOrderMutation.data ?? createWorkOrderMutation.data}
                errorMessage={maintenanceErrorMessage}
                canWrite={canWriteMaintenance}
                actorName={operatorName}
                onSelectWorkOrder={setSelectedWorkOrderId}
                onCreateWorkOrder={createWorkOrder}
                onTransitionWorkOrder={transitionWorkOrder}
              />
            </section>
          ) : null}

          {activeView === 'reconciliation' ? (
            <section className="ops-view">
              <ReconciliationPanel
                exceptions={snapshot?.reconciliationExceptions ?? []}
                drafts={snapshot?.billingDrafts ?? []}
                siteCode={snapshot?.site.code}
                isReviewPending={reviewExceptionMutation.isPending}
                isCorrectionPending={correctionMutation.isPending}
                isConfirmPending={confirmBillMutation.isPending}
                isExportPending={exportReconciliationMutation.isPending}
                lastCorrection={correctionMutation.data}
                lastExport={exportReconciliationMutation.data}
                errorMessage={financeErrorMessage}
                canReview={canReviewFinance}
                reviewerName={operatorName}
                onReview={reviewException}
                onCreateCorrection={createCorrection}
                onConfirmBill={confirmBill}
                onExport={exportReconciliation}
              />
            </section>
          ) : null}

          {activeView === 'ai' ? (
            <section className="ops-view">
              <AIAssistantPanel
                sessions={filteredSessions}
                workOrders={snapshot?.maintenance?.workOrders ?? []}
                siteCode={snapshot?.site.code}
                insight={aiMutation.data}
                isPending={aiMutation.isPending}
                errorMessage={aiErrorMessage}
                canGenerate={canReadAI}
                onGenerate={(request) => aiMutation.mutate(request)}
              />
            </section>
          ) : null}

          {activeView === 'users' ? (
            <section className="ops-view">
              <AccessPanel enabled={canManageUsers} />
            </section>
          ) : null}

          {activeView === 'audit' ? (
            <section className="ops-view">
              <AuditLogPanel enabled={canReadAudit} />
            </section>
          ) : null}

          {activeView === 'config' ? (
            <section className="ops-view">
              <ConfigPanel
                snapshot={configQuery.data}
                enabled={canManageConfig}
                isLoading={configQuery.isLoading}
                isPending={updateConfigMutation.isPending || createPricingPolicyMutation.isPending}
                errorMessage={configErrorMessage}
                onUpdate={(request) => updateConfigMutation.mutate(request)}
                onCreatePricingPolicy={(request) => createPricingPolicyMutation.mutate(request)}
              />
            </section>
          ) : null}
        </div>
      )}
    </main>
  )
}

function firstApiError(...errors: unknown[]): string | undefined {
  const apiError = errors.find((error): error is ApiError => error instanceof ApiError)
  if (!apiError) return undefined
  return `${apiError.message} / ${apiError.details ?? apiError.code}`
}
