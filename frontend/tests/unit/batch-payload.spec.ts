import { describe, expect, it } from 'vitest'
import {
  buildBatchCreateRequest,
  toRFC3339FromLocal,
  validateBatchSummaryInput
} from '../../app/composables/useBatchCreate'

describe('batch payload builder', () => {
  it('converts datetime-local to RFC3339 string', () => {
    const local = '2026-03-04T10:30'
    const converted = toRFC3339FromLocal(local)

    expect(converted).toBeTruthy()
    expect(converted).toBe(new Date(local).toISOString())
  })

  it('maps and trims form input fields into request payload', () => {
    const payload = buildBatchCreateRequest(
      {
        orchard_id: ' orchard-demo-01 ',
        orchard_name: ' 荔枝示范园 ',
        plot_id: ' plot-a01 ',
        plot_name: ' A1区 ',
        harvested_at: '2026-03-04T02:30:00.000Z',
        note: ' 首批果园采摘批次 ',
        confirm_unripe: true
      },
      {
        total: 100,
        green: 10,
        half: 20,
        red: 60,
        young: 10
      }
    )

    expect(payload.orchard_id).toBe('orchard-demo-01')
    expect(payload.orchard_name).toBe('荔枝示范园')
    expect(payload.plot_id).toBe('plot-a01')
    expect(payload.plot_name).toBe('A1区')
    expect(payload.note).toBe('首批果园采摘批次')
    expect(payload.summary.total).toBe(100)
    expect(payload.confirm_unripe).toBe(true)
  })

  it('validates summary count consistency', () => {
    const ok = validateBatchSummaryInput({
      total: 4,
      green: 1,
      half: 1,
      red: 1,
      young: 1
    })
    expect(ok).toBeNull()

    const invalid = validateBatchSummaryInput({
      total: 4,
      green: 2,
      half: 1,
      red: 1,
      young: 1
    })
    expect(invalid).toContain('不一致')
  })
})
