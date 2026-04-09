import type { AuthSession, Principal } from '~/types/auth'

export const AUTH_SESSION_KEY = 'lychee-ripe.auth.session'
export const AUTH_PRINCIPAL_KEY = 'lychee-ripe.auth.principal'
export const AUTH_PENDING_KEY = 'lychee-ripe.auth.pending'

export type PendingLoginState = {
  state: string
  codeVerifier: string
  redirectPath: string
}

export type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

export function getAuthStorage(isTauri: boolean): StorageLike | null {
  if (!hasStorage('localStorage') || !hasStorage('sessionStorage')) {
    return null
  }
  return isTauri ? globalThis.localStorage : globalThis.sessionStorage
}

export function clearLegacyWebAuthStorage() {
  if (!hasStorage('localStorage')) {
    return
  }
  globalThis.localStorage.removeItem(AUTH_SESSION_KEY)
  globalThis.localStorage.removeItem(AUTH_PRINCIPAL_KEY)
  globalThis.localStorage.removeItem(AUTH_PENDING_KEY)
}

export function loadStoredSession(storage: StorageLike | null): AuthSession | null {
  const raw = storage?.getItem(AUTH_SESSION_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    storage?.removeItem(AUTH_SESSION_KEY)
    return null
  }
}

export function saveStoredSession(storage: StorageLike | null, session: AuthSession) {
  storage?.setItem(AUTH_SESSION_KEY, JSON.stringify(session))
}

export function loadStoredPrincipal(storage: StorageLike | null): Principal | null {
  const raw = storage?.getItem(AUTH_PRINCIPAL_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as Principal
  } catch {
    storage?.removeItem(AUTH_PRINCIPAL_KEY)
    return null
  }
}

export function saveStoredPrincipal(storage: StorageLike | null, principal: Principal) {
  storage?.setItem(AUTH_PRINCIPAL_KEY, JSON.stringify(principal))
}

export function clearStoredAuth(storage: StorageLike | null) {
  storage?.removeItem(AUTH_SESSION_KEY)
  storage?.removeItem(AUTH_PRINCIPAL_KEY)
}

export function savePendingLoginState(storage: StorageLike | null, state: PendingLoginState) {
  if (!storage) {
    return
  }
  storage.setItem(pendingKey(state.state), JSON.stringify(state))
}

export function loadPendingLoginState(storage: StorageLike | null, state: string): PendingLoginState | null {
  if (!storage) {
    return null
  }
  const raw = storage.getItem(pendingKey(state))
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as PendingLoginState
  } catch {
    storage.removeItem(pendingKey(state))
    return null
  }
}

export function clearPendingLoginState(storage: StorageLike | null, state: string) {
  storage?.removeItem(pendingKey(state))
}

function pendingKey(state: string) {
  return `${AUTH_PENDING_KEY}:${state.trim()}`
}

function hasStorage(key: 'localStorage' | 'sessionStorage') {
  return typeof globalThis !== 'undefined' && key in globalThis && globalThis[key] != null
}
