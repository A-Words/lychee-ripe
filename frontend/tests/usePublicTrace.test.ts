import { describe, expect, it, vi } from 'vitest'
import { usePublicTrace } from '../app/composables/usePublicTrace'

describe('usePublicTrace', () => {
  it('fetches public trace and keeps contract field names', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({
        batch: {
          batch_id: 'batch_01',
          trace_code: 'TRC-ABCD-EFGH',
          status: 'anchored',
          orchard_name: '增城果园',
          plot_name: 'A-01',
          harvested_at: '2026-03-03T08:00:00Z',
          summary: {
            total: 20,
            green: 2,
            half: 6,
            red: 10,
            young: 2,
            unripe_count: 4,
            unripe_ratio: 0.2,
            unripe_handling: 'sorted_out',
          },
          created_at: '2026-03-03T09:00:00Z',
        },
        verify_result: {
          verify_status: 'pass',
          reason: 'anchor_hash matches on-chain record',
        },
      }),
    })) as unknown as typeof fetch

    const trace = usePublicTrace({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    const result = await trace.fetchTrace('TRC-ABCD-EFGH')
    expect(result.batch.trace_code).toBe('TRC-ABCD-EFGH')
    expect(result.verify_result.verify_status).toBe('pass')
    expect(trace.status.value).toBe('success')
  })

  it('maps not found error payload', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 404,
      json: async () => ({
        error: 'trace_not_found',
        message: 'trace code not found',
      }),
    })) as unknown as typeof fetch

    const trace = usePublicTrace({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    await expect(trace.fetchTrace('TRC-NOT-EXIST')).rejects.toThrow('trace code not found')
    expect(trace.lastError.value?.status).toBe(404)
    expect(trace.lastError.value?.code).toBe('trace_not_found')
  })
})
