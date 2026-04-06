import { describe, expect, it } from 'vitest'
import { VERIFY_STATUS_META } from '../../app/constants/verify-status'

describe('verify status mapping', () => {
  it('maps pass/fail/pending to expected ui colors', () => {
    expect(VERIFY_STATUS_META.pass.color).toBe('success')
    expect(VERIFY_STATUS_META.pending.color).toBe('warning')
    expect(VERIFY_STATUS_META.fail.color).toBe('error')
    expect(VERIFY_STATUS_META.recorded.color).toBe('primary')
  })

  it('contains public labels', () => {
    expect(VERIFY_STATUS_META.pass.label).toBeTruthy()
    expect(VERIFY_STATUS_META.pending.label).toBeTruthy()
    expect(VERIFY_STATUS_META.fail.label).toBeTruthy()
    expect(VERIFY_STATUS_META.recorded.label).toBeTruthy()
  })
})
