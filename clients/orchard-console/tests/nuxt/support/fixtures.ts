import type { Batch, BatchSummary } from '../../../app/types/batch'
import type { DashboardOverviewResponse } from '../../../app/types/dashboard'
import type { TraceResponse } from '../../../app/types/trace'
import type { SessionAggregateSummary } from '../../../app/utils/session-aggregator'

export function buildBatchSummary(overrides: Partial<BatchSummary> = {}): BatchSummary {
  return {
    total: 10,
    green: 2,
    half: 3,
    red: 4,
    young: 1,
    unripe_count: 3,
    unripe_ratio: 0.3,
    unripe_handling: 'sorted_out',
    ...overrides
  }
}

export function buildSessionSummary(overrides: Partial<SessionAggregateSummary> = {}): SessionAggregateSummary {
  const summary = buildBatchSummary(overrides)
  return {
    ...summary,
    ripeness_ratio: {
      green: summary.total > 0 ? summary.green / summary.total : 0,
      half: summary.total > 0 ? summary.half / summary.total : 0,
      red: summary.total > 0 ? summary.red / summary.total : 0,
      young: summary.total > 0 ? summary.young / summary.total : 0
    },
    harvest_suggestion: 'partially_ready',
    ...overrides
  }
}

export function buildBatch(overrides: Partial<Batch> = {}): Batch {
  return {
    batch_id: 'batch-001',
    trace_code: 'TRC-9A7X-11QF',
    trace_mode: 'blockchain',
    status: 'anchored',
    orchard_id: 'orchard-demo-01',
    orchard_name: '荔枝示范园',
    plot_id: 'plot-a01',
    plot_name: 'A1 区',
    harvested_at: '2026-03-30T08:00:00.000Z',
    summary: buildBatchSummary(),
    note: '首批果园采摘批次',
    created_at: '2026-03-30T09:00:00.000Z',
    anchor_proof: {
      tx_hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
      block_number: 128,
      chain_id: '31337',
      contract_address: '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd',
      anchor_hash: '0x9876543210abcdef9876543210abcdef9876543210abcdef9876543210abcdef',
      anchored_at: '2026-03-30T09:05:00.000Z'
    },
    ...overrides
  }
}

export function buildTraceResponse(overrides: Partial<TraceResponse> = {}): TraceResponse {
  const batchOverrides = overrides.batch ?? {}
  const verifyOverrides = overrides.verify_result ?? {}

  return {
    batch: {
      batch_id: 'batch-001',
      trace_code: 'TRC-9A7X-11QF',
      trace_mode: 'blockchain',
      status: 'anchored',
      orchard_name: '荔枝示范园',
      plot_name: 'A1 区',
      harvested_at: '2026-03-30T08:00:00.000Z',
      summary: buildBatchSummary(),
      created_at: '2026-03-30T09:00:00.000Z',
      ...batchOverrides
    },
    verify_result: {
      verify_status: 'pass',
      reason: '链上摘要与库内摘要一致',
      ...verifyOverrides
    }
  }
}

export function buildDashboardOverview(overrides: Partial<DashboardOverviewResponse> = {}): DashboardOverviewResponse {
  return {
    trace_mode: overrides.trace_mode ?? 'blockchain',
    totals: {
      batch_total: 8,
      ...overrides.totals
    },
    status_distribution: {
      anchored: 5,
      pending_anchor: 2,
      anchor_failed: 1,
      ...overrides.status_distribution
    },
    ripeness_distribution: {
      green: 6,
      half: 10,
      red: 18,
      young: 2,
      ...overrides.ripeness_distribution
    },
    unripe_metrics: {
      unripe_batch_count: 2,
      unripe_batch_ratio: 0.25,
      threshold: 0.15,
      unripe_handling: 'sorted_out',
      ...overrides.unripe_metrics
    },
    recent_anchors: overrides.recent_anchors ?? [
      {
        batch_id: 'batch-001',
        trace_code: 'TRC-9A7X-11QF',
        status: 'anchored',
        tx_hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
        anchored_at: '2026-03-30T09:05:00.000Z',
        created_at: '2026-03-30T09:00:00.000Z'
      }
    ],
    reconcile_stats: overrides.reconcile_stats === null
      ? null
      : {
          pending_count: 2,
          retried_total: 4,
          failed_total: 1,
          last_reconcile_at: '2026-03-30T10:00:00.000Z',
          ...overrides.reconcile_stats
        }
  }
}
