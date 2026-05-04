export const Permission = {
  SitesAny: 'sites:any',
  SitesRead: 'sites:read',
  SessionsRead: 'sessions:read',
  SessionsWrite: 'sessions:write',
  CommandsWrite: 'commands:write',
  SimulatorControl: 'simulator:control',
  LoadControlRead: 'load_control:read',
  LoadControlWrite: 'load_control:write',
  BillingRead: 'billing:read',
  BillingGenerate: 'billing:generate',
  FinanceReview: 'finance:review',
  MaintenanceRead: 'maintenance:read',
  MaintenanceWrite: 'maintenance:write',
  AuditRead: 'audit:read',
  ConfigManage: 'config:manage',
  UsersManage: 'users:manage',
  AIRead: 'ai:read',
} as const

export type PermissionCode = (typeof Permission)[keyof typeof Permission]

export interface Role {
  code: string
  name: string
}

export interface AuthorizedSite {
  id: string
  code: string
  name: string
  accessLevel: string
}

export interface AuthUser {
  userId: string
  username: string
  displayName: string
  status: string
  roles: Role[]
  permissions: string[]
  sites: AuthorizedSite[]
  expiresAt: string
}

export interface LoginResult {
  token: string
  expiresAt: string
  user: AuthUser
}

export interface UserSummary {
  id: string
  username: string
  displayName: string
  status: string
  roles: Role[]
  permissions: string[]
  sites: AuthorizedSite[]
  createdAt: string
  updatedAt: string
}

export interface RoleDetail {
  id: string
  code: string
  name: string
  description: string
  status: string
  permissions: string[]
  createdAt: string
  updatedAt: string
}

export interface AuditLog {
  id: string
  entityType: string
  entityId?: string
  action: string
  actorUserId?: string
  actorName: string
  actorRoleCode: string
  ipAddress: string
  traceId: string
  payload: Record<string, unknown>
  createdAt: string
}

export interface AuditLogPage {
  list: AuditLog[]
  pagination: {
    page: number
    pageSize: number
    total: number
    totalPages: number
  }
}

export interface AuditLogFilters {
  entityType?: string
  action?: string
  actorName?: string
  page?: number
  pageSize?: number
}
