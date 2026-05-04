import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createLoadControlRecord,
  createRemoteCommand,
  createReservation,
  generateBillingDraft,
  getOperationsSnapshot,
  getSessionDetail,
  getSimulatorStatus,
  runSimulatorOnce,
  startSimulator,
  stopSimulator,
} from '../services/operations-api'
import type {
  CommandPathType,
  CreateLoadControlRecordRequest,
  CreateReservationRequest,
  SimulatorRequest,
} from '../types/operations'

export function useOperationsSnapshot(siteCode?: string) {
  return useQuery({
    queryKey: ['operations', 'snapshot', siteCode],
    queryFn: () => getOperationsSnapshot(siteCode),
    refetchInterval: 15_000,
  })
}

export function useSessionDetail(sessionId?: string) {
  return useQuery({
    queryKey: ['operations', 'session', sessionId],
    queryFn: () => getSessionDetail(sessionId ?? ''),
    enabled: Boolean(sessionId),
    refetchInterval: 10_000,
  })
}

export function useRemoteCommand() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: {
      sessionId: string
      commandType: CommandPathType
      requestedBy: string
      targetPowerKw?: number
    }) =>
      createRemoteCommand(request.sessionId, request.commandType, {
        requestedBy: request.requestedBy,
        targetPowerKw: request.targetPowerKw,
        payload: { source: 'operations-console' },
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['operations'] })
    },
  })
}

export function useCreateReservation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateReservationRequest) => createReservation(request),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['operations'] })
    },
  })
}

export function useGenerateBillingDraft() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: { sessionId: string; generatedBy: string }) =>
      generateBillingDraft(request.sessionId, { generatedBy: request.generatedBy }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['operations'] })
    },
  })
}

export function useCreateLoadControlRecord() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateLoadControlRecordRequest) => createLoadControlRecord(request),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['operations'] })
    },
  })
}

export function useSimulatorControl() {
  const queryClient = useQueryClient()
  const statusQuery = useQuery({
    queryKey: ['operations', 'simulator'],
    queryFn: getSimulatorStatus,
    refetchInterval: 10_000,
  })

  const onceMutation = useMutation({
    mutationFn: (request: SimulatorRequest) => runSimulatorOnce(request),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['operations', 'simulator'] }),
        queryClient.invalidateQueries({ queryKey: ['operations', 'snapshot'] }),
      ])
    },
  })

  const startMutation = useMutation({
    mutationFn: (request: SimulatorRequest) => startSimulator(request),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['operations', 'simulator'] })
    },
  })

  const stopMutation = useMutation({
    mutationFn: stopSimulator,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['operations', 'simulator'] }),
        queryClient.invalidateQueries({ queryKey: ['operations', 'snapshot'] }),
      ])
    },
  })

  return {
    statusQuery,
    onceMutation,
    startMutation,
    stopMutation,
  }
}
