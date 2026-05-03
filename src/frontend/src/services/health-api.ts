import type { ApiResponse } from '../types/api'
import type { HealthStatus } from '../types/health'

export async function getHealth(): Promise<HealthStatus> {
  const response = await fetch('/api/v1/health')
  const payload = (await response.json()) as ApiResponse<HealthStatus>

  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message)
  }

  return payload.data
}
