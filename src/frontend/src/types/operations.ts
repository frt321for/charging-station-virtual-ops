export type SiteStatus = 'active' | 'inactive' | 'maintenance'

export type ChargerStatus =
  | 'available'
  | 'occupied'
  | 'charging'
  | 'fault'
  | 'faulted'
  | 'offline'
  | 'maintenance'
  | 'disabled'

export type ConnectorStatus =
  | 'available'
  | 'reserved'
  | 'plugged'
  | 'charging'
  | 'fault'
  | 'faulted'
  | 'offline'
  | 'unavailable'

export type SessionStatus =
  | 'reserved'
  | 'waiting_arrival'
  | 'plugged_in'
  | 'starting'
  | 'charging'
  | 'paused'
  | 'stopping'
  | 'pending_billing'
  | 'billed'
  | 'pending_review'
  | 'cancelled'

export type CommandPathType = 'start' | 'stop' | 'pause' | 'resume' | 'reset' | 'limit-power'
export type RemoteCommandStatus = 'sent' | 'accepted' | 'rejected' | 'timeout' | 'failed' | 'retried'

export interface SiteSummary {
  id: string
  code: string
  name: string
  campus: string
  capacityKw: number
  loadLimitKw: number
  status: SiteStatus
  chargerCount: number
  connectorCount: number
  availableConnectors: number
  activeSessions: number
  faultedChargers: number
}

export interface SiteTopology {
  site: SiteNode
  areas: AreaNode[]
}

export interface SiteNode {
  id: string
  code: string
  name: string
  campus: string
  capacityKw: number
  loadLimitKw: number
  status: SiteStatus
}

export interface AreaNode {
  id: string
  code: string
  name: string
  loadLimitKw: number
  status: SiteStatus
  groups: GroupNode[]
}

export interface GroupNode {
  id: string
  code: string
  name: string
  electricalNode: string
  loadLimitKw: number
  priority: number
  status: SiteStatus
  chargers: ChargerNode[]
}

export interface ChargerNode {
  id: string
  code: string
  name: string
  chargerType: 'ac' | 'dc' | string
  ratedPowerKw: number
  status: ChargerStatus
  installationLocation: string
  connectors: ConnectorNode[]
}

export interface ConnectorNode {
  id: string
  code: string
  number: number
  maxPowerKw: number
  status: ConnectorStatus
}

export interface SessionSummary {
  id: string
  sessionNo: string
  status: SessionStatus
  siteId: string
  siteName: string
  chargerId: string
  chargerCode: string
  connectorId: string
  connectorCode: string
  meterStartKwh?: number
  meterStopKwh?: number
  reservationExpiry?: string
  startedAt?: string
  stoppedAt?: string
  stopReason: string
  updatedAt: string
}

export interface SessionEvent {
  id: string
  eventType: string
  occurredAt: string
  source: string
  payload: Record<string, unknown>
}

export interface MeterValue {
  time: string
  powerKw: number
  voltageV?: number
  currentA?: number
  meterKwh: number
  rawPayload: Record<string, unknown>
}

export interface SessionDetail extends SessionSummary {
  events: SessionEvent[]
  meterValues: MeterValue[]
}

export interface RemoteCommand {
  id: string
  commandNo: string
  sessionId?: string
  sessionNo: string
  chargerId: string
  chargerCode: string
  connectorId?: string
  connectorCode: string
  commandType: string
  status: RemoteCommandStatus
  requestedBy: string
  targetPowerKw?: number
  payload: Record<string, unknown>
  resultMessage: string
  sentAt: string
  acknowledgedAt?: string
}

export interface PricingPeriod {
  id: string
  label: string
  startMinute: number
  endMinute: number
  energyPricePerKwh: number
  serviceFeePerKwh: number
  occupancyFeePerMinute: number
}

export interface PricingPolicy {
  id: string
  siteId: string
  siteCode: string
  code: string
  name: string
  version: number
  chargerType: string
  effectiveFrom: string
  effectiveTo?: string
  status: 'draft' | 'active' | 'retired'
  periods: PricingPeriod[]
}

export interface BillingDraft {
  id: string
  billNo: string
  sessionId: string
  sessionNo: string
  siteId: string
  siteCode: string
  connectorCode: string
  pricingPolicyId: string
  policyCode: string
  policyVersion: number
  energyKwh: number
  durationMinutes: number
  energyAmount: number
  serviceAmount: number
  occupancyAmount: number
  totalAmount: number
  currency: string
  status: 'draft' | 'confirmed' | 'pending_review'
  exceptionFlag: boolean
  generatedBy: string
  generatedAt: string
}

export interface ReconciliationException {
  id: string
  exceptionNo: string
  billId: string
  billNo: string
  sessionId: string
  sessionNo: string
  exceptionType: 'energy' | 'duration' | 'amount' | 'stop_reason'
  severity: 'low' | 'medium' | 'high' | 'critical'
  status: 'open' | 'reviewing' | 'resolved'
  reason: string
  suggestedAction: string
  detectedAt: string
  resolvedAt?: string
}

export interface BillingCorrection {
  id: string
  correctionNo: string
  billId: string
  billNo: string
  exceptionId?: string
  exceptionNo: string
  correctedEnergyKwh?: number
  correctedDurationMinutes?: number
  correctedTotalAmount?: number
  reason: string
  reviewerName: string
  status: 'draft' | 'applied' | 'voided'
  createdAt: string
}

export interface ReconciliationExportRow {
  exceptionNo: string
  billNo: string
  sessionNo: string
  exceptionType: ReconciliationException['exceptionType']
  severity: ReconciliationException['severity']
  status: ReconciliationException['status']
  reason: string
  totalAmount: number
}

export interface ReconciliationExport {
  exportNo: string
  filename: string
  generatedBy: string
  generatedAt: string
  rows: ReconciliationExportRow[]
  csv: string
}

export interface ReviewExceptionRequest {
  status: ReconciliationException['status']
  reviewerName: string
  note: string
}

export interface CreateCorrectionRequest {
  correctedEnergyKwh?: number
  correctedDurationMinutes?: number
  correctedTotalAmount?: number
  reason: string
  reviewerName: string
}

export interface ConfirmBillRequest {
  reviewerName: string
  note: string
}

export interface LoadPolicy {
  id: string
  siteId: string
  scopeType: 'site' | 'area' | 'group'
  scopeCode: string
  scopeName: string
  thresholdKw: number
  warningKw: number
  actionMode: 'limit_power' | 'pause' | 'queue' | 'reject'
  version: number
  status: 'draft' | 'active' | 'retired'
}

export interface ConfigSite {
  id: string
  code: string
  name: string
  campus: string
  capacityKw: number
  loadLimitKw: number
  timezone: string
  slaResponseMinutes: number
  slaRecoveryMinutes: number
  status: 'active' | 'inactive'
  updatedAt: string
}

export interface ConfigArea {
  id: string
  siteId: string
  code: string
  name: string
  loadLimitKw: number
  sortOrder: number
  status: 'active' | 'inactive'
  updatedAt: string
}

export interface ConfigGroup {
  id: string
  siteId: string
  areaId: string
  code: string
  name: string
  electricalNode: string
  loadLimitKw: number
  priority: number
  status: 'active' | 'inactive'
  updatedAt: string
}

export interface ConfigCharger {
  id: string
  groupId: string
  groupCode: string
  code: string
  name: string
  chargerType: 'ac' | 'dc' | string
  ratedPowerKw: number
  connectorCount: number
  status: ChargerStatus | 'unregistered'
  installationLocation: string
  maintenanceTag: string
  updatedAt: string
}

export interface ConfigConnector {
  id: string
  chargerId: string
  chargerCode: string
  code: string
  number: number
  maxPowerKw: number
  status: ConnectorStatus | 'maintenance' | 'disabled'
  updatedAt: string
}

export interface ReservationRule {
  id: string
  siteId: string
  siteCode: string
  code: string
  name: string
  holdMinutes: number
  timeoutAction: 'release' | 'promote_queue' | 'cancel'
  version: number
  status: 'draft' | 'active' | 'retired'
  updatedAt: string
}

export interface QueueRule {
  id: string
  siteId: string
  siteCode: string
  code: string
  name: string
  strategy: 'reservation_time' | 'member_priority' | 'load_priority'
  maxQueueSize: number
  priorityFactor: string
  version: number
  status: 'draft' | 'active' | 'retired'
  updatedAt: string
}

export interface ConfigSnapshot {
  site: ConfigSite
  areas: ConfigArea[]
  groups: ConfigGroup[]
  chargers: ConfigCharger[]
  connectors: ConfigConnector[]
  pricingPolicies: PricingPolicy[]
  loadPolicies: LoadPolicy[]
  reservationRules: ReservationRule[]
  queueRules: QueueRule[]
}

export type ConfigEntity =
  | 'site'
  | 'area'
  | 'group'
  | 'charger'
  | 'connector'
  | 'loadPolicy'
  | 'reservationRule'
  | 'queueRule'

export interface ConfigUpdateRequest {
  entity: ConfigEntity
  id: string
  body: Record<string, string | number | undefined>
}

export interface CreatePricingPolicyRequest {
  siteId: string
  code: string
  name: string
  chargerType: string
  status: PricingPolicy['status']
  periods: Array<{
    label: string
    startMinute: number
    endMinute: number
    energyPricePerKwh: number
    serviceFeePerKwh: number
    occupancyFeePerMinute: number
  }>
}

export interface QueueItem {
  position: number
  sessionId: string
  sessionNo: string
  connectorCode: string
  status: SessionStatus
  reservationExpiry?: string
  waitMinutes: number
  reason: string
  updatedAt: string
}

export type LoadActionType = 'limit_power' | 'pause' | 'resume' | 'queue' | 'reject' | 'promote' | 'release'
export type LoadRecordStatus = 'recommended' | 'sent' | 'applied' | 'rejected'

export interface LoadControlRecord {
  id: string
  recordNo: string
  siteId: string
  siteCode: string
  scopeCode: string
  sessionId?: string
  sessionNo: string
  connectorCode: string
  actionType: LoadActionType
  triggerType: 'manual' | 'auto'
  reason: string
  beforeLoadKw: number
  afterLoadKw: number
  targetPowerKw?: number
  status: LoadRecordStatus
  operatorName: string
  createdAt: string
}

export interface LoadControlSnapshot {
  siteId: string
  siteCode: string
  siteName: string
  currentLoadKw: number
  loadLimitKw: number
  availableCapacityKw: number
  policies: LoadPolicy[]
  queue: QueueItem[]
  records: LoadControlRecord[]
}

export type FaultSeverity = 'low' | 'medium' | 'high' | 'critical'
export type FaultStatus = 'open' | 'linked_work_order' | 'resolved'
export type WorkOrderStatus =
  | 'open'
  | 'assigned'
  | 'accepted'
  | 'arrived'
  | 'handling'
  | 'retest'
  | 'recovered'
  | 'closed'
  | 'cancelled'

export interface Fault {
  id: string
  faultNo: string
  siteId: string
  siteCode: string
  chargerId: string
  chargerCode: string
  connectorId?: string
  connectorCode: string
  sessionId?: string
  sessionNo: string
  faultCode: string
  severity: FaultSeverity
  status: FaultStatus
  workOrderId?: string
  workOrderNo: string
  repeatCount: number
  occurredAt: string
  resolvedAt?: string
}

export interface WorkOrder {
  id: string
  workOrderNo: string
  faultId?: string
  faultNo: string
  siteId: string
  siteCode: string
  chargerId: string
  chargerCode: string
  connectorId?: string
  connectorCode: string
  sessionId?: string
  sessionNo: string
  severity: FaultSeverity
  status: WorkOrderStatus
  impactScope: string
  title: string
  description: string
  assigneeName: string
  responseDueAt: string
  recoveryDueAt: string
  acceptedAt?: string
  arrivedAt?: string
  handlingAt?: string
  retestAt?: string
  recoveredAt?: string
  closedAt?: string
  slaResponseBreached: boolean
  slaRecoveryBreached: boolean
  createdAt: string
  updatedAt: string
}

export interface WorkOrderEvent {
  id: string
  workOrderId: string
  eventType: string
  fromStatus: string
  toStatus: string
  actorName: string
  note: string
  payload: Record<string, unknown>
  occurredAt: string
}

export interface SLASummary {
  siteId: string
  siteCode: string
  openWorkOrders: number
  overdueResponse: number
  overdueRecovery: number
  criticalOpen: number
  repeatedFaults: number
  averageRecoveryMins: number
  responseSlaHitRate: number
  recoverySlaHitRate: number
}

export interface MaintenanceSnapshot {
  faults: Fault[]
  workOrders: WorkOrder[]
  sla: SLASummary
}

export type AIInsightKind =
  | 'session_explanation'
  | 'work_order_summary'
  | 'congestion_risk'
  | 'daily_report'
  | 'station_qa'

export interface AIInsightEvidence {
  label: string
  value: string
}

export interface AIInsightSite {
  id: string
  code: string
  name: string
}

export interface AIInsightTarget {
  type: string
  id: string
  code: string
}

export interface AIInsight {
  requestId: string
  kind: AIInsightKind
  site: AIInsightSite
  target: AIInsightTarget
  content: string
  suggestions: string[]
  evidence: AIInsightEvidence[]
  provider: string
  model: string
  generatedAt: string
}

export type AIInsightRequest =
  | { kind: 'session_explanation'; sessionId: string }
  | { kind: 'work_order_summary'; workOrderId: string }
  | { kind: 'congestion_risk'; siteId: string; horizonHours: number }
  | { kind: 'daily_report'; siteId: string; businessDate: string }
  | { kind: 'station_qa'; siteId: string; question: string }

export interface CreateWorkOrderRequest {
  assigneeName: string
  impactScope: string
  title: string
  description: string
  actorName: string
}

export interface TransitionWorkOrderRequest {
  targetStatus: WorkOrderStatus
  assigneeName: string
  actorName: string
  note: string
  payload?: Record<string, unknown>
}

export interface CreateReservationRequest {
  connectorCode: string
  reservationMinutes: number
  requestedBy: string
  payload?: Record<string, unknown>
}

export interface CreateCommandRequest {
  requestedBy: string
  targetPowerKw?: number
  payload?: Record<string, unknown>
}

export interface CreateLoadControlRecordRequest {
  siteId: string
  sessionId?: string
  actionType: LoadActionType
  reason: string
  beforeLoadKw: number
  afterLoadKw: number
  targetPowerKw?: number
  status: LoadRecordStatus
  operatorName: string
}

export interface GenerateBillingDraftRequest {
  generatedBy: string
}

export interface SimulatorRequest {
  chargerCount: number
  connectorCount: number
  onlineRate: number
  faultRate: number
  loadCurve: 'commute' | 'flat' | 'random'
}

export interface SimulatorResult {
  chargerCode: string
  connectorCode: string
  sessionNo?: string
  finalStatus: string
  faultRaised: boolean
  samples: number
  finalMeterKwh?: number
}

export interface SimulatorStatus {
  running: boolean
  mode: string
  config: SimulatorRequest
  startedAt?: string
  lastRunAt?: string
  lastStoppedAt?: string
  lastError?: string
  generated: number
  results: SimulatorResult[]
}

export interface OperationsSnapshot {
  sites: SiteSummary[]
  site: SiteSummary
  topology?: SiteTopology
  sessions: SessionSummary[]
  sessionDetails: SessionDetail[]
  pricingPolicies: PricingPolicy[]
  billingDrafts: BillingDraft[]
  reconciliationExceptions: ReconciliationException[]
  loadControl?: LoadControlSnapshot
  maintenance?: MaintenanceSnapshot
}

export interface OperationsSnapshotAccess {
  canReadBilling: boolean
  canReadLoadControl: boolean
  canReadMaintenance: boolean
  canReadSessions: boolean
}
