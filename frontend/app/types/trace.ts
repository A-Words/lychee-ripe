export type RipenessLabel = 'green' | 'half' | 'red' | 'young'
export type BatchStatus = 'pending_anchor' | 'anchored' | 'anchor_failed'
export type VerifyStatus = 'pass' | 'fail' | 'pending'

export interface BatchSummary {
  total: number
  green: number
  half: number
  red: number
  young: number
  unripe_count: number
  unripe_ratio: number
  unripe_handling: 'sorted_out'
}

export interface TraceBatch {
  batch_id: string
  trace_code: string
  status: BatchStatus
  orchard_name: string
  plot_name: string
  harvested_at: string
  summary: BatchSummary
  created_at: string
}

export interface TraceVerifyResult {
  verify_status: VerifyStatus
  reason: string
}

export interface TraceResponse {
  batch: TraceBatch
  verify_result: TraceVerifyResult
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}

export type TraceViewState = 'loading' | 'success' | 'not_found' | 'unavailable'

export interface TraceApiError {
  statusCode: number
  error: string
  message: string
  requestId?: string
}
