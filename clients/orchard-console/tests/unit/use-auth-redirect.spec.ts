import { describe, expect, it } from 'vitest'

import { normalizeRedirectPath, resolveClientRedirect } from '../../app/composables/useAuth'

describe('useAuth redirect helpers', () => {
  it('normalizes invalid redirect values to dashboard', () => {
    expect(normalizeRedirectPath('dashboard')).toBe('/dashboard')
    expect(normalizeRedirectPath('//evil.example')).toBe('/dashboard')
  })

  it('resolves local redirects under the inferred app base path', () => {
    expect(resolveClientRedirect('/console', '/dashboard')).toBe('/console/dashboard')
  })
})
