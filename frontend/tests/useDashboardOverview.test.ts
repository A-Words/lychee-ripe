import { describe, expect, it, vi } from 'vitest'
import { useDashboardOverview } from '../app/composables/useDashboardOverview'

describe('useDashboardOverview', () => {
  it('fetches dashboard overview with contract fields', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({
        totals: { batch_total: 8 },
        status_distribution: {
          anchored: 5,
          pending_anchor: 2,
          anchor_failed: 1,
        },
        ripeness_distribution: {
          green: 10,
          half: 20,
          red: 30,
          young: 8,
        },
        unripe_metrics: {
          unripe_batch_count: 2,
          unripe_batch_ratio: 0.25,
          threshold: 0.15,
          unripe_handling: 'sorted_out',
        },
        recent_anchors: [
          {
            batch_id: 'batch_01',
            trace_code: 'TRC-01',
            status: 'anchored',
            tx_hash: '0xabc',
            anchored_at: '2026-03-03T08:00:00Z',
            created_at: '2026-03-03T08:00:00Z',
          },
        ],
        reconcile_stats: {
          pending_count: 2,
          retried_total: 4,
          failed_total: 1,
          last_reconcile_at: '2026-03-03T09:00:00Z',
        },
      }),
    })) as unknown as typeof fetch

    const dashboard = useDashboardOverview({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    const result = await dashboard.fetchOverview({ apiKey: 'demo-key' })
    expect(result.totals.batch_total).toBe(8)
    expect(result.unripe_metrics.unripe_handling).toBe('sorted_out')
    expect(result.recent_anchors[0]?.trace_code).toBe('TRC-01')
    expect(dashboard.status.value).toBe('success')
  })

  it('maps unauthorized error payload', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 401,
      json: async () => ({
        error: 'unauthorized',
        message: 'missing or invalid api key',
      }),
    })) as unknown as typeof fetch

    const dashboard = useDashboardOverview({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    await expect(dashboard.fetchOverview()).rejects.toThrow('missing or invalid api key')
    expect(dashboard.lastError.value?.status).toBe(401)
    expect(dashboard.lastError.value?.code).toBe('unauthorized')
  })
})
