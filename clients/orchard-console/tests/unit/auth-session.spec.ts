import { describe, expect, it } from 'vitest'
import {
  getErrorStatusCode,
  resolveAuthenticatedRequest,
  resolveBootstrapPrincipal,
  shouldClearSessionForPrincipalError
} from '../../app/utils/auth-session'
import type { Principal } from '../../app/types/auth'

const CACHED_PRINCIPAL: Principal = {
  subject: 'sub-1',
  email: 'admin@example.com',
  display_name: 'Admin',
  role: 'admin',
  status: 'active',
  auth_mode: 'oidc',
  permissions: ['admin']
}

describe('auth session bootstrap helpers', () => {
  it('keeps cached principal on transient auth service failures', () => {
    expect(resolveBootstrapPrincipal(CACHED_PRINCIPAL, { status: 503 })).toEqual({
      principal: CACHED_PRINCIPAL,
      clearPersistedAuth: false
    })
    expect(resolveBootstrapPrincipal(CACHED_PRINCIPAL, new Error('network hiccup'))).toEqual({
      principal: CACHED_PRINCIPAL,
      clearPersistedAuth: false
    })
  })

  it('clears persisted auth on explicit auth rejection', () => {
    expect(resolveBootstrapPrincipal(CACHED_PRINCIPAL, { status: 401 })).toEqual({
      principal: null,
      clearPersistedAuth: true
    })
    expect(resolveBootstrapPrincipal(CACHED_PRINCIPAL, { response: { status: 403 } })).toEqual({
      principal: null,
      clearPersistedAuth: true
    })
  })

  it('extracts status codes from common fetch error shapes', () => {
    expect(getErrorStatusCode({ status: 503 })).toBe(503)
    expect(getErrorStatusCode({ statusCode: 401 })).toBe(401)
    expect(getErrorStatusCode({ response: { status: 403 } })).toBe(403)
    expect(getErrorStatusCode(new Error('boom'))).toBeUndefined()
  })

  it('classifies only 401 and 403 as session-clearing auth failures', () => {
    expect(shouldClearSessionForPrincipalError({ status: 401 })).toBe(true)
    expect(shouldClearSessionForPrincipalError({ status: 403 })).toBe(true)
    expect(shouldClearSessionForPrincipalError({ status: 503 })).toBe(false)
    expect(shouldClearSessionForPrincipalError({})).toBe(false)
  })

  it('clears persisted auth only for explicit auth rejection in later authenticated requests', () => {
    expect(resolveAuthenticatedRequest({ status: 401 })).toEqual({
      clearPersistedAuth: true
    })
    expect(resolveAuthenticatedRequest({ response: { status: 403 } })).toEqual({
      clearPersistedAuth: true
    })
    expect(resolveAuthenticatedRequest({ status: 503 })).toEqual({
      clearPersistedAuth: false
    })
    expect(resolveAuthenticatedRequest(new Error('network hiccup'))).toEqual({
      clearPersistedAuth: false
    })
  })
})
