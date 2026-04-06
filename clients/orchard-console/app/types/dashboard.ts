import type { BatchStatus, TraceMode } from '~/types/trace'

export interface DashboardTotals {
  batch_total: number
}

export interface BatchStatusDistribution {
  stored?: number
  anchored?: number
  pending_anchor?: number
  anchor_failed?: number
}

export interface RipenessDistribution {
  green: number
  half: number
  red: number
  young: number
}

export interface UnripeMetrics {
  unripe_batch_count: number
  unripe_batch_ratio: number
  threshold: number
  unripe_handling: 'sorted_out'
}

export interface RecentAnchorRecord {
  batch_id: string
  trace_code: string
  status: BatchStatus
  tx_hash: string | null
  anchored_at: string | null
  created_at: string
}

export interface ReconcileStats {
  pending_count: number
  retried_total: number
  failed_total: number
  last_reconcile_at: string | null
}

export interface DashboardOverviewResponse {
  trace_mode: TraceMode
  totals: DashboardTotals
  status_distribution: BatchStatusDistribution
  ripeness_distribution: RipenessDistribution
  unripe_metrics: UnripeMetrics
  recent_anchors: RecentAnchorRecord[]
  reconcile_stats?: ReconcileStats | null
}

export interface DashboardErrorResponse {
  error: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}

export interface DashboardApiError {
  statusCode: number
  error: string
  message: string
  requestId?: string
}

export type DashboardViewState = 'loading' | 'ready' | 'empty' | 'auth_blocked' | 'unavailable'
