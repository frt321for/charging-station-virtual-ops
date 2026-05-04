import type { ApiResponse } from '../types/api'

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
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
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
