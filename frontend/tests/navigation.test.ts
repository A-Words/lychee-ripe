import { describe, expect, it } from 'vitest'
import { APP_NAV_ITEMS, TRACE_DEMO_CODE, buildTracePath } from '../app/constants/navigation'

describe('navigation constants', () => {
  it('contains exactly three core navigation entries', () => {
    expect(APP_NAV_ITEMS).toHaveLength(3)
    expect(APP_NAV_ITEMS.map(item => item.label)).toEqual(['识别建批', '溯源查询', '数据看板'])
    expect(APP_NAV_ITEMS.map(item => item.to)).toEqual([
      '/',
      `/trace/${TRACE_DEMO_CODE}`,
      '/dashboard',
    ])
  })

  it('builds encoded trace route path', () => {
    expect(buildTracePath('TRC-9A7X-11QF')).toBe('/trace/TRC-9A7X-11QF')
    expect(buildTracePath(' trc 中文 ')).toBe('/trace/trc%20%E4%B8%AD%E6%96%87')
  })
})
