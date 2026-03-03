import { describe, expect, it, vi } from 'vitest'
import {
  computeUnripeMetrics,
  needsUnripeConfirm,
  useBatchCreate,
} from '../app/composables/useBatchCreate'
import type { BatchCreateRequest } from '../app/types/batch'

function createRequest(confirmUnripe = false): BatchCreateRequest {
  return {
    orchard_id: 'orchard_zc_xc',
    orchard_name: '增城仙村果园',
    plot_id: 'plot_a01',
    plot_name: 'A-01',
    harvested_at: '2026-03-03T08:00:00Z',
    summary: {
      total: 20,
      green: 3,
      half: 4,
      red: 8,
      young: 5,
    },
    note: 'demo',
    confirm_unripe: confirmUnripe,
  }
}

describe('useBatchCreate helpers', () => {
  it('computes unripe count and ratio correctly', () => {
    const result = computeUnripeMetrics({
      total: 20,
      green: 2,
      half: 6,
      red: 9,
      young: 3,
    })
    expect(result.unripeCount).toBe(5)
    expect(result.unripeRatio).toBe(0.25)
  })

  it('checks unripe threshold boundary correctly', () => {
    expect(needsUnripeConfirm({
      total: 20,
      green: 2,
      half: 8,
      red: 7,
      young: 1,
    })).toBe(false)

    expect(needsUnripeConfirm({
      total: 20,
      green: 2,
      half: 8,
      red: 6,
      young: 4,
    })).toBe(true)
  })
})

describe('useBatchCreate request', () => {
  it('submits batch and handles anchored(201) response', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 201,
      json: async () => ({
        batch_id: 'batch_01',
        trace_code: 'TRC-AAAA-BBBB',
        status: 'anchored',
        orchard_id: 'orchard_zc_xc',
        orchard_name: '增城仙村果园',
        plot_id: 'plot_a01',
        plot_name: 'A-01',
        harvested_at: '2026-03-03T08:00:00Z',
        summary: {
          total: 20,
          green: 3,
          half: 4,
          red: 8,
          young: 5,
          unripe_count: 8,
          unripe_ratio: 0.4,
          unripe_handling: 'sorted_out',
        },
        note: 'demo',
        created_at: '2026-03-03T08:00:10Z',
        anchor_proof: null,
      }),
    })) as unknown as typeof fetch

    const batchCreate = useBatchCreate({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    const result = await batchCreate.createBatch(createRequest())
    expect(result.statusCode).toBe(201)
    expect(result.batch.status).toBe('anchored')
    expect(batchCreate.status.value).toBe('success')
  })

  it('submits batch and handles pending_anchor(202) response', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 202,
      json: async () => ({
        batch_id: 'batch_02',
        trace_code: 'TRC-CCCC-DDDD',
        status: 'pending_anchor',
        orchard_id: 'orchard_zc_xc',
        orchard_name: '增城仙村果园',
        plot_id: 'plot_a01',
        plot_name: 'A-01',
        harvested_at: '2026-03-03T08:00:00Z',
        summary: {
          total: 20,
          green: 3,
          half: 4,
          red: 8,
          young: 5,
          unripe_count: 8,
          unripe_ratio: 0.4,
          unripe_handling: 'sorted_out',
        },
        note: 'demo',
        created_at: '2026-03-03T08:00:10Z',
        anchor_proof: null,
      }),
    })) as unknown as typeof fetch

    const batchCreate = useBatchCreate({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    const result = await batchCreate.createBatch(createRequest(true))
    expect(result.statusCode).toBe(202)
    expect(result.batch.status).toBe('pending_anchor')
    expect(batchCreate.status.value).toBe('success')
  })

  it('maps API error payload to throwable message', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 400,
      json: async () => ({
        error: 'invalid_request',
        message: 'confirm_unripe must be true when unripe_ratio > 0.15',
        request_id: 'req-123',
      }),
    })) as unknown as typeof fetch

    const batchCreate = useBatchCreate({
      gatewayBase: 'http://127.0.0.1:9000',
      fetchImpl: fetchMock,
    })

    await expect(batchCreate.createBatch(createRequest(false))).rejects.toThrow(
      'confirm_unripe must be true when unripe_ratio > 0.15',
    )
    expect(batchCreate.status.value).toBe('error')
    expect(batchCreate.lastError.value?.code).toBe('invalid_request')
  })
})
