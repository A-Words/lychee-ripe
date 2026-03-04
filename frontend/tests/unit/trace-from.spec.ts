import { describe, expect, it } from 'vitest'
import {
  buildTracePathWithFrom,
  getTraceBackTarget,
  getTraceFromQuery
} from '../../app/utils/trace-from'

describe('trace from helpers', () => {
  it('builds trace detail path with from parameter', () => {
    expect(buildTracePathWithFrom('trc-9a7x-11qf', 'dashboard')).toBe(
      '/trace/TRC-9A7X-11QF?from=dashboard'
    )
  })

  it('parses valid and invalid from query value', () => {
    expect(getTraceFromQuery('dashboard')).toBe('dashboard')
    expect(getTraceFromQuery(['batch_create'])).toBe('batch_create')
    expect(getTraceFromQuery('unexpected')).toBe('unknown')
    expect(getTraceFromQuery(undefined)).toBe('unknown')
  })

  it('maps source to back target', () => {
    expect(getTraceBackTarget('dashboard')).toEqual({
      to: '/dashboard',
      label: '返回数据看板'
    })
    expect(getTraceBackTarget('batch_create')).toEqual({
      to: '/batch/create',
      label: '返回识别建批'
    })
    expect(getTraceBackTarget('unknown')).toBeNull()
  })
})
