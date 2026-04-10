import { describe, expect, it } from 'vitest'
import { shouldShowTopNav } from '../../app/utils/top-nav-visibility'

describe('top nav visibility', () => {
  it('always shows nav outside trace pages', () => {
    expect(shouldShowTopNav('/dashboard', undefined)).toBe(true)
    expect(shouldShowTopNav('/batch/create', undefined)).toBe(true)
  })

  it('hides nav on login routes', () => {
    expect(shouldShowTopNav('/login', undefined)).toBe(false)
    expect(shouldShowTopNav('/auth/callback', undefined)).toBe(false)
  })

  it('hides nav for public trace access', () => {
    expect(shouldShowTopNav('/trace', undefined)).toBe(false)
    expect(shouldShowTopNav('/trace/TRC-9A7X-11QF', 'invalid')).toBe(false)
  })

  it('shows nav for internal trace access', () => {
    expect(shouldShowTopNav('/trace', 'index')).toBe(true)
    expect(shouldShowTopNav('/trace/TRC-9A7X-11QF', 'dashboard')).toBe(true)
    expect(shouldShowTopNav('/trace/TRC-9A7X-11QF', ['batch_create'])).toBe(true)
  })
})
