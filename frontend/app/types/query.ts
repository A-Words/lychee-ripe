import type { BatchStatus, BatchSummary } from './batch'

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

export type VerifyStatus = 'pass' | 'fail' | 'pending'

export interface TraceVerifyResult {
  verify_status: VerifyStatus
  reason: string
}

export interface TraceResponse {
  batch: TraceBatch
  verify_result: TraceVerifyResult
}

export interface DashboardTotals {
  batch_total: number
}

export interface BatchStatusDistribution {
  anchored: number
  pending_anchor: number
  anchor_failed: number
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
  totals: DashboardTotals
  status_distribution: BatchStatusDistribution
  ripeness_distribution: RipenessDistribution
  unripe_metrics: UnripeMetrics
  recent_anchors: RecentAnchorRecord[]
  reconcile_stats: ReconcileStats
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}
