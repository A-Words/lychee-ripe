import { describe, expect, it } from 'vitest'
import {
  formatDateTime,
  formatPercent,
  truncateTxHash
} from '../../app/utils/dashboard-format'

describe('dashboard format helpers', () => {
  it('formats percentage and handles invalid input', () => {
    expect(formatPercent(0.256)).toBe('25.6%')
    expect(formatPercent(Number.NaN)).toBe('--')
  })

  it('formats datetime and keeps invalid raw text', () => {
    expect(formatDateTime('2026-03-04T10:30:00Z')).not.toBe('--')
    expect(formatDateTime('invalid-time')).toBe('invalid-time')
    expect(formatDateTime(null)).toBe('--')
  })

  it('truncates tx hash with defaults', () => {
    const value = truncateTxHash('0x1234567890abcdef1234567890abcdef1234567890abcdef')
    expect(value).toContain('...')
    expect(truncateTxHash('')).toBe('--')
  })
})
