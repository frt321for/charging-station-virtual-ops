export type SiteStatus = 'active' | 'inactive' | 'maintenance'

export type ChargerStatus =
  | 'available'
  | 'occupied'
  | 'charging'
  | 'faulted'
  | 'offline'
  | 'maintenance'
  | 'disabled'

export type ConnectorStatus =
  | 'available'
  | 'reserved'
  | 'plugged'
  | 'charging'
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
  topology: SiteTopology
  sessions: SessionSummary[]
  sessionDetails: SessionDetail[]
}
