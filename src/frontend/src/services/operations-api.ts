import type {
  BillingDraft,
  BillingCorrection,
  CommandPathType,
  CreateCommandRequest,
  CreateCorrectionRequest,
  CreateLoadControlRecordRequest,
  CreateWorkOrderRequest,
  ConfirmBillRequest,
  CreateReservationRequest,
  GenerateBillingDraftRequest,
  LoadControlRecord,
  LoadControlSnapshot,
  MaintenanceSnapshot,
  OperationsSnapshot,
  PricingPolicy,
  ReconciliationException,
  ReconciliationExport,
  ReviewExceptionRequest,
  RemoteCommand,
  SessionDetail,
  SessionSummary,
  SimulatorRequest,
  SimulatorStatus,
  SiteSummary,
  SiteTopology,
  TransitionWorkOrderRequest,
  WorkOrder,
  WorkOrderEvent,
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

export async function listPricingPolicies(siteId: string): Promise<PricingPolicy[]> {
  const data = await apiRequest<ListResponse<PricingPolicy>>(
    `/api/v1/pricing-policies?siteId=${encodeURIComponent(siteId)}`,
  )
  return data.list
}

export async function listBillingDrafts(siteId: string): Promise<BillingDraft[]> {
  const data = await apiRequest<ListResponse<BillingDraft>>(
    `/api/v1/billing-drafts?siteId=${encodeURIComponent(siteId)}`,
  )
  return data.list
}

export async function generateBillingDraft(
  sessionId: string,
  request: GenerateBillingDraftRequest,
): Promise<BillingDraft> {
  return apiRequest<BillingDraft>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/billing-draft`, {
    method: 'POST',
    body: jsonBody(request),
  })
}

export async function listReconciliationExceptions(siteId: string): Promise<ReconciliationException[]> {
  const data = await apiRequest<ListResponse<ReconciliationException>>(
    `/api/v1/reconciliation-exceptions?siteId=${encodeURIComponent(siteId)}`,
  )
  return data.list
}

export async function reviewReconciliationException(
  exceptionId: string,
  request: ReviewExceptionRequest,
): Promise<{ status: ReconciliationException['status'] }> {
  return apiRequest<{ status: ReconciliationException['status'] }>(
    `/api/v1/reconciliation-exceptions/${encodeURIComponent(exceptionId)}/review`,
    {
      method: 'POST',
      body: jsonBody(request),
    },
  )
}

export async function createBillingCorrection(
  exceptionId: string,
  request: CreateCorrectionRequest,
): Promise<BillingCorrection> {
  return apiRequest<BillingCorrection>(
    `/api/v1/reconciliation-exceptions/${encodeURIComponent(exceptionId)}/corrections`,
    {
      method: 'POST',
      body: jsonBody(request),
    },
  )
}

export async function exportReconciliation(
  siteId: string,
  generatedBy: string,
): Promise<ReconciliationExport> {
  return apiRequest<ReconciliationExport>(
    `/api/v1/reconciliation-exceptions/export?siteId=${encodeURIComponent(siteId)}&generatedBy=${encodeURIComponent(
      generatedBy,
    )}`,
  )
}

export async function confirmBillingDraft(
  billId: string,
  request: ConfirmBillRequest,
): Promise<{ status: BillingDraft['status'] }> {
  return apiRequest<{ status: BillingDraft['status'] }>(
    `/api/v1/billing-drafts/${encodeURIComponent(billId)}/confirm`,
    {
      method: 'POST',
      body: jsonBody(request),
    },
  )
}

export async function getLoadControlSnapshot(siteId: string): Promise<LoadControlSnapshot> {
  return apiRequest<LoadControlSnapshot>(`/api/v1/sites/${encodeURIComponent(siteId)}/load-control`)
}

export async function createLoadControlRecord(
  request: CreateLoadControlRecordRequest,
): Promise<LoadControlRecord> {
  return apiRequest<LoadControlRecord>('/api/v1/load-control/records', {
    method: 'POST',
    body: jsonBody(request),
  })
}

export async function getMaintenanceSnapshot(siteId: string): Promise<MaintenanceSnapshot> {
  return apiRequest<MaintenanceSnapshot>(`/api/v1/maintenance?siteId=${encodeURIComponent(siteId)}`)
}

export async function createWorkOrder(
  faultId: string,
  request: CreateWorkOrderRequest,
): Promise<WorkOrder> {
  return apiRequest<WorkOrder>(
    `/api/v1/faults/${encodeURIComponent(faultId)}/work-order`,
    {
      method: 'POST',
      body: jsonBody(request),
    },
  )
}

export async function listWorkOrderEvents(workOrderId: string): Promise<WorkOrderEvent[]> {
  const data = await apiRequest<ListResponse<WorkOrderEvent>>(
    `/api/v1/work-orders/${encodeURIComponent(workOrderId)}/events`,
  )
  return data.list
}

export async function transitionWorkOrder(
  workOrderId: string,
  request: TransitionWorkOrderRequest,
): Promise<WorkOrder> {
  return apiRequest<WorkOrder>(
    `/api/v1/work-orders/${encodeURIComponent(workOrderId)}/transition`,
    {
      method: 'POST',
      body: jsonBody(request),
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
  const [
    topology,
    pricingPolicies,
    billingDrafts,
    reconciliationExceptions,
    loadControl,
    maintenance,
    detailResults,
  ] =
    await Promise.all([
      getSiteTopology(site.code),
      listPricingPolicies(site.code),
      listBillingDrafts(site.code),
      listReconciliationExceptions(site.code),
      getLoadControlSnapshot(site.code),
      getMaintenanceSnapshot(site.code),
      Promise.allSettled(sessions.slice(0, 12).map((session) => getSessionDetail(session.sessionNo))),
    ])
  const sessionDetails = detailResults.flatMap((result) =>
    result.status === 'fulfilled' ? [result.value] : [],
  )

  return {
    sites,
    site,
    topology,
    sessions,
    sessionDetails,
    pricingPolicies,
    billingDrafts,
    reconciliationExceptions,
    loadControl,
    maintenance,
  }
}
