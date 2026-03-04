import { describe, expect, it } from 'vitest'
import { buildDashboardTracePath } from '../../app/utils/dashboard-route'

describe('dashboard route helpers', () => {
  it('builds trace route path from trace code', () => {
    expect(buildDashboardTracePath('trc-9a7x-11qf')).toBe('/trace/TRC-9A7X-11QF')
  })
})
