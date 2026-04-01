import { describe, expect, it } from 'vitest'
import { buildTopNavItems } from '../../app/utils/top-nav-items'

function getItem(path: string, key: 'batch_create' | 'dashboard' | 'trace') {
  const item = buildTopNavItems(path).find((candidate) => candidate.key === key)
  if (!item) {
    throw new Error(`missing nav item: ${key}`)
  }
  return item
}

describe('top nav items', () => {
  it('uses internal trace entry with from=index', () => {
    expect(getItem('/dashboard', 'trace').to).toBe('/trace?from=index')
  })

  it('marks dashboard route as active', () => {
    expect(getItem('/dashboard', 'dashboard').active).toBe(true)
    expect(getItem('/dashboard', 'batch_create').active).toBe(false)
    expect(getItem('/dashboard', 'trace').active).toBe(false)
  })

  it('marks batch create route as active', () => {
    expect(getItem('/batch/create', 'batch_create').active).toBe(true)
    expect(getItem('/batch/create', 'dashboard').active).toBe(false)
    expect(getItem('/batch/create', 'trace').active).toBe(false)
  })

  it('marks trace route prefix as active', () => {
    expect(getItem('/trace', 'trace').active).toBe(true)
    expect(getItem('/trace/TRC-9A7X-11QF', 'trace').active).toBe(true)
  })
})
