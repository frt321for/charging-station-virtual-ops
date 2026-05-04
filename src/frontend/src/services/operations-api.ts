import type {
  CommandPathType,
  CreateCommandRequest,
  CreateReservationRequest,
  OperationsSnapshot,
  RemoteCommand,
  SessionDetail,
  SessionSummary,
  SimulatorRequest,
  SimulatorStatus,
  SiteSummary,
  SiteTopology,
} from '../types/operations'
import { ApiError, apiRequest, jsonBody } from './api-client'

interface ListResponse<TItem> {
  list: TItem[]
}

export async function listSites(): Promise<SiteSummary[]> {
  const data = await apiRequest<ListResponse<SiteSummary>>('/api/v1/sites')
  return data.list
}

export async function getSiteTopology(siteId: string): Promise<SiteTopology> {
  return apiRequest<SiteTopology>(`/api/v1/sites/${encodeURIComponent(siteId)}/topology`)
}

export async function listSessions(): Promise<SessionSummary[]> {
  const data = await apiRequest<ListResponse<SessionSummary>>('/api/v1/sessions')
  return data.list
}

export async function getSessionDetail(sessionId: string): Promise<SessionDetail> {
  return apiRequest<SessionDetail>(`/api/v1/sessions/${encodeURIComponent(sessionId)}`)
}

export async function createReservation(request: CreateReservationRequest): Promise<SessionDetail> {
  return apiRequest<SessionDetail>('/api/v1/reservations', {
    method: 'POST',
    body: jsonBody({
      connectorCode: request.connectorCode,
      reservationMinutes: request.reservationMinutes,
      requestedBy: request.requestedBy,
      payload: request.payload ?? {},
    }),
  })
}

export async function createRemoteCommand(
  sessionId: string,
  commandType: CommandPathType,
  request: CreateCommandRequest,
): Promise<RemoteCommand> {
  return apiRequest<RemoteCommand>(
    `/api/v1/sessions/${encodeURIComponent(sessionId)}/commands/${commandType}`,
    {
      method: 'POST',
      body: jsonBody({
        requestedBy: request.requestedBy,
        targetPowerKw: request.targetPowerKw,
        payload: request.payload ?? {},
      }),
    },
  )
}

export async function getSimulatorStatus(): Promise<SimulatorStatus> {
  return apiRequest<SimulatorStatus>('/api/v1/simulator/status')
}

export async function runSimulatorOnce(request: SimulatorRequest): Promise<SimulatorStatus> {
  return apiRequest<SimulatorStatus>('/api/v1/simulator/once', {
    method: 'POST',
    body: jsonBody(request),
  })
}

export async function startSimulator(request: SimulatorRequest): Promise<SimulatorStatus> {
  return apiRequest<SimulatorStatus>('/api/v1/simulator/start', {
    method: 'POST',
    body: jsonBody(request),
  })
}

export async function stopSimulator(): Promise<SimulatorStatus> {
  return apiRequest<SimulatorStatus>('/api/v1/simulator/stop', {
    method: 'POST',
    body: jsonBody({}),
  })
}

export async function getOperationsSnapshot(siteCode?: string): Promise<OperationsSnapshot> {
  const [sites, allSessions] = await Promise.all([listSites(), listSessions()])
  const site = sites.find((item) => item.code === siteCode || item.id === siteCode) ?? sites[0]

  if (!site) {
    throw new ApiError('站点数据为空', { code: 30001, status: 404 })
  }

  const sessions = allSessions.filter((session) => session.siteId === site.id)
  const topology = await getSiteTopology(site.code)
  const detailResults = await Promise.allSettled(
    sessions.slice(0, 12).map((session) => getSessionDetail(session.sessionNo)),
  )
  const sessionDetails = detailResults.flatMap((result) =>
    result.status === 'fulfilled' ? [result.value] : [],
  )

  return {
    sites,
    site,
    topology,
    sessions,
    sessionDetails,
  }
}
