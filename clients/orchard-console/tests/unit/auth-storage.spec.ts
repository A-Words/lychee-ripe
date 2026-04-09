import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  AUTH_PENDING_KEY,
  AUTH_PRINCIPAL_KEY,
  AUTH_SESSION_KEY,
  clearLegacyWebAuthStorage,
  clearPendingLoginState,
  clearStoredAuth,
  getAuthStorage,
  loadPendingLoginState,
  loadStoredPrincipal,
  loadStoredSession,
  savePendingLoginState,
  saveStoredPrincipal,
  saveStoredSession
} from '../../app/utils/auth-storage'
import type { AuthSession, Principal } from '../../app/types/auth'

function createMemoryStorage() {
  const store = new Map<string, string>()
  return {
    getItem(key: string) {
      return store.has(key) ? store.get(key)! : null
    },
    setItem(key: string, value: string) {
      store.set(key, value)
    },
    removeItem(key: string) {
      store.delete(key)
    }
  }
}

const SESSION: AuthSession = {
  accessToken: 'access-token',
  idToken: 'id-token',
  expiresAt: 123456
}

const PRINCIPAL: Principal = {
  subject: 'sub-1',
  email: 'admin@example.com',
  display_name: 'Admin',
  role: 'admin',
  status: 'active',
  auth_mode: 'oidc',
  permissions: ['admin']
}

describe('auth storage helpers', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('uses sessionStorage for web auth and localStorage for tauri auth', () => {
    const local = createMemoryStorage()
    const session = createMemoryStorage()
    vi.stubGlobal('localStorage', local)
    vi.stubGlobal('sessionStorage', session)

    expect(getAuthStorage(false)).toBe(session)
    expect(getAuthStorage(true)).toBe(local)
  })

  it('stores web auth session and principal outside localStorage', () => {
    const local = createMemoryStorage()
    const session = createMemoryStorage()
    vi.stubGlobal('localStorage', local)
    vi.stubGlobal('sessionStorage', session)

    const storage = getAuthStorage(false)
    saveStoredSession(storage, SESSION)
    saveStoredPrincipal(storage, PRINCIPAL)

    expect(loadStoredSession(storage)).toEqual(SESSION)
    expect(loadStoredPrincipal(storage)).toEqual(PRINCIPAL)
    expect(local.getItem(AUTH_SESSION_KEY)).toBeNull()
    expect(local.getItem(AUTH_PRINCIPAL_KEY)).toBeNull()
  })

  it('clears legacy localStorage auth artifacts for web upgrades', () => {
    const local = createMemoryStorage()
    const session = createMemoryStorage()
    vi.stubGlobal('localStorage', local)
    vi.stubGlobal('sessionStorage', session)

    local.setItem(AUTH_SESSION_KEY, JSON.stringify(SESSION))
    local.setItem(AUTH_PRINCIPAL_KEY, JSON.stringify(PRINCIPAL))
    local.setItem(AUTH_PENDING_KEY, '{"legacy":true}')

    clearLegacyWebAuthStorage()

    expect(local.getItem(AUTH_SESSION_KEY)).toBeNull()
    expect(local.getItem(AUTH_PRINCIPAL_KEY)).toBeNull()
    expect(local.getItem(AUTH_PENDING_KEY)).toBeNull()
  })

  it('stores pending PKCE state per state value', () => {
    const session = createMemoryStorage()

    savePendingLoginState(session, {
      state: 'state-a',
      codeVerifier: 'verifier-a',
      redirectPath: '/dashboard'
    })
    savePendingLoginState(session, {
      state: 'state-b',
      codeVerifier: 'verifier-b',
      redirectPath: '/admin'
    })

    expect(loadPendingLoginState(session, 'state-a')).toEqual({
      state: 'state-a',
      codeVerifier: 'verifier-a',
      redirectPath: '/dashboard'
    })
    expect(loadPendingLoginState(session, 'state-b')).toEqual({
      state: 'state-b',
      codeVerifier: 'verifier-b',
      redirectPath: '/admin'
    })

    clearPendingLoginState(session, 'state-a')
    expect(loadPendingLoginState(session, 'state-a')).toBeNull()
    expect(loadPendingLoginState(session, 'state-b')?.codeVerifier).toBe('verifier-b')
  })

  it('clears stored auth from the selected storage boundary only', () => {
    const local = createMemoryStorage()
    const session = createMemoryStorage()

    saveStoredSession(session, SESSION)
    saveStoredPrincipal(session, PRINCIPAL)
    saveStoredSession(local, SESSION)

    clearStoredAuth(session)

    expect(loadStoredSession(session)).toBeNull()
    expect(loadStoredPrincipal(session)).toBeNull()
    expect(loadStoredSession(local)).toEqual(SESSION)
  })
})
