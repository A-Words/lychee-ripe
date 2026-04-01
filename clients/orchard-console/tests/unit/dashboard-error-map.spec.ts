import { describe, expect, it } from 'vitest'
import { mapDashboardErrorMessage } from '../../app/composables/useDashboardApi'

describe('dashboard error mapping', () => {
  it('maps auth status to fixed hint', () => {
    const message401 = mapDashboardErrorMessage(401, '')
    const message403 = mapDashboardErrorMessage(403, '')

    expect(message401).toContain('不传 API Key')
    expect(message403).toContain('不传 API Key')
  })

  it('keeps fallback message for 503', () => {
    const message = mapDashboardErrorMessage(503, 'service unavailable')
    expect(message).toBe('service unavailable')
  })

  it('uses default fallback for unknown status', () => {
    const message = mapDashboardErrorMessage(520, '')
    expect(message).toContain('获取看板数据失败')
  })
})
