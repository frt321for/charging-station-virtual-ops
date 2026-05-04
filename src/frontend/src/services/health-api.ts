import type { HealthStatus } from '../types/health'
import { apiRequest } from './api-client'

export async function getHealth(): Promise<HealthStatus> {
  return apiRequest<HealthStatus>('/api/v1/health')
}
