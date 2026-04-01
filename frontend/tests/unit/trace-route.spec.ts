import { describe, expect, it } from 'vitest'
import { buildTracePath, getTraceCodeFromRouteParam, normalizeTraceCode } from '../../app/utils/trace-route'

describe('trace route helpers', () => {
  it('normalizes trace code by trimming and uppercasing', () => {
    expect(normalizeTraceCode('  trc-9a7x-11qf  ')).toBe('TRC-9A7X-11QF')
  })

  it('builds landing path for trace query', () => {
    expect(buildTracePath('trc-9a7x-11qf')).toBe('/trace/TRC-9A7X-11QF')
  })

  it('extracts route param from string and array', () => {
    expect(getTraceCodeFromRouteParam('trc-abcd-0001')).toBe('TRC-ABCD-0001')
    expect(getTraceCodeFromRouteParam(['trc-xyz-0002'])).toBe('TRC-XYZ-0002')
  })
})
