import type { Principal } from '~/types/auth'

type AuthBootstrapDecision = {
  principal: Principal | null
  clearPersistedAuth: boolean
}

export function resolveBootstrapPrincipal(cachedPrincipal: Principal | null, error: unknown): AuthBootstrapDecision {
  if (shouldClearSessionForPrincipalError(error)) {
    return {
      principal: null,
      clearPersistedAuth: true
    }
  }

  return {
    principal: cachedPrincipal,
    clearPersistedAuth: false
  }
}

export function shouldClearSessionForPrincipalError(error: unknown) {
  const statusCode = getErrorStatusCode(error)
  return statusCode === 401 || statusCode === 403
}

export function getErrorStatusCode(error: unknown) {
  if (!error || typeof error !== 'object') {
    return undefined
  }

  const value = error as {
    status?: unknown
    statusCode?: unknown
    response?: { status?: unknown }
  }

  for (const candidate of [value.status, value.statusCode, value.response?.status]) {
    if (typeof candidate === 'number') {
      return candidate
    }
  }

  return undefined
}
