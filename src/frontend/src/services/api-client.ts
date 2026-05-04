import type { ApiResponse } from '../types/api'

const authTokenKey = 'charging_ops_token'
let cachedToken = readStoredToken()

export class ApiError extends Error {
  readonly code: number
  readonly status: number
  readonly traceId?: string
  readonly details?: string

  constructor(message: string, options: { code: number; status: number; traceId?: string; details?: string }) {
    super(message)
    this.name = 'ApiError'
    this.code = options.code
    this.status = options.status
    this.traceId = options.traceId
    this.details = options.details
  }
}

export async function apiRequest<TData>(path: string, init?: RequestInit): Promise<TData> {
  const headers = new Headers(init?.headers)
  headers.set('Content-Type', headers.get('Content-Type') ?? 'application/json')
  const token = getAuthToken()
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers,
  })
  const payload = (await response.json()) as ApiResponse<TData>

  if (!response.ok || payload.code !== 0) {
    throw new ApiError(payload.message || '请求失败', {
      code: payload.code,
      status: response.status,
      traceId: payload.traceId,
      details: payload.details,
    })
  }

  if (payload.data === undefined) {
    throw new ApiError('响应数据为空', {
      code: payload.code,
      status: response.status,
      traceId: payload.traceId,
    })
  }

  return payload.data
}

export function jsonBody(value: unknown): string {
  return JSON.stringify(value)
}

export function getAuthToken(): string | undefined {
  return cachedToken
}

export function setAuthToken(token?: string) {
  cachedToken = token?.trim() || undefined
  if (typeof window === 'undefined') return
  if (cachedToken) {
    window.localStorage.setItem(authTokenKey, cachedToken)
  } else {
    window.localStorage.removeItem(authTokenKey)
  }
}

function readStoredToken(): string | undefined {
  if (typeof window === 'undefined') return undefined
  return window.localStorage.getItem(authTokenKey) ?? undefined
}
