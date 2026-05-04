export interface ApiResponse<TData> {
  code: number
  message: string
  data?: TData
  details?: string
  timestamp: string
  traceId: string
}
