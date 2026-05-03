export interface HealthStatus {
  status: 'ok' | 'degraded'
  dependencies: Record<string, 'ok' | 'error' | 'missing'>
}
