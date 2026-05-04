import type { AuditLogFilters, AuditLogPage, LoginResult, RoleDetail, UserSummary, AuthUser } from '../types/auth'
import { apiRequest, jsonBody } from './api-client'

interface ListResponse<TItem> {
  list: TItem[]
}

export async function login(username: string, password: string): Promise<LoginResult> {
  return apiRequest<LoginResult>('/api/v1/auth/login', {
    method: 'POST',
    body: jsonBody({ username, password }),
  })
}

export async function getCurrentUser(): Promise<AuthUser> {
  return apiRequest<AuthUser>('/api/v1/auth/me')
}

export async function logout(): Promise<{ status: string }> {
  return apiRequest<{ status: string }>('/api/v1/auth/logout', {
    method: 'POST',
    body: jsonBody({}),
  })
}

export async function listUsers(): Promise<UserSummary[]> {
  const data = await apiRequest<ListResponse<UserSummary>>('/api/v1/users')
  return data.list
}

export async function listRoles(): Promise<RoleDetail[]> {
  const data = await apiRequest<ListResponse<RoleDetail>>('/api/v1/roles')
  return data.list
}

export async function listAuditLogs(filters: AuditLogFilters): Promise<AuditLogPage> {
  const params = new URLSearchParams()
  if (filters.entityType) params.set('entityType', filters.entityType)
  if (filters.action) params.set('action', filters.action)
  if (filters.actorName) params.set('actorName', filters.actorName)
  params.set('page', String(filters.page ?? 1))
  params.set('pageSize', String(filters.pageSize ?? 20))
  return apiRequest<AuditLogPage>(`/api/v1/audit-logs?${params.toString()}`)
}
