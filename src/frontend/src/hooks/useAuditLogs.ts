import { useQuery } from '@tanstack/react-query'
import { listAuditLogs } from '../services/auth-api'
import type { AuditLogFilters } from '../types/auth'

export function useAuditLogs(filters: AuditLogFilters, enabled: boolean) {
  return useQuery({
    queryKey: ['audit-logs', filters],
    queryFn: () => listAuditLogs(filters),
    enabled,
  })
}
