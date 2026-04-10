import type { AuthMode, AuthSession, Principal } from '~/types/auth'
import {
  clearLegacyWebAuthStorage,
  clearStoredAuth,
  getPrincipalStorage,
  getSessionStorage,
  loadStoredPrincipal,
  loadStoredSession,
  saveStoredPrincipal,
  saveStoredSession
} from '~/utils/auth-storage'
import { buildAppPath, inferAppBasePath } from '~/utils/app-path'
import { resolveAuthenticatedRequest, resolveBootstrapPrincipal } from '~/utils/auth-session'
import { buildURLUnderBase } from '~/utils/base-url'
import { toWebSocketBase } from '~/utils/ws-url'

let initPromise: Promise<void> | null = null

export function useAuth() {
  const config = useRuntimeConfig()
  const gatewayBase = useGatewayBase()
  const router = useRouter()
  const mode = computed<AuthMode>(() => normalizeAuthMode(config.public.authMode))
  const principal = useState<Principal | null>('auth.principal', () => null)
  const session = useState<AuthSession | null>('auth.session', () => null)
  const initialized = useState<boolean>('auth.initialized', () => false)
  const initializing = useState<boolean>('auth.initializing', () => false)

  const isAuthenticated = computed(() => mode.value === 'disabled' || Boolean(principal.value))
  const isAdmin = computed(() => mode.value === 'disabled' || principal.value?.role === 'admin')
  const appBasePath = computed(() =>
    import.meta.client ? inferAppBasePath(window.location.pathname, readRoutePath()) : ''
  )

  async function init(force = false) {
    if (initialized.value && !force) {
      return
    }
    if (initPromise && !force) {
      return await initPromise
    }

    initPromise = (async () => {
      initializing.value = true
      try {
        if (mode.value === 'disabled') {
          principal.value = buildDisabledPrincipal()
          session.value = null
          initialized.value = true
          return
        }

        if (!isTauriRuntime()) {
          clearLegacyWebAuthStorage()
        }

        session.value = loadSession()
        principal.value = loadPrincipal()

        if (isTauriRuntime()) {
          if (!session.value?.accessToken) {
            clearSession()
            initialized.value = true
            return
          }
        }

        try {
          const nextPrincipal = await fetchPrincipal(session.value?.accessToken)
          setPrincipal(nextPrincipal)
        } catch (error) {
          const decision = resolveBootstrapPrincipal(principal.value, error)
          principal.value = decision.principal
          if (decision.clearPersistedAuth) {
            clearSession()
          }
        }

        initialized.value = true
      } finally {
        initializing.value = false
        initPromise = null
      }
    })()

    return await initPromise
  }

  async function login(redirectPath = '/dashboard') {
    const normalizedRedirectPath = normalizeRedirectPath(redirectPath)
    if (mode.value === 'disabled') {
      await navigateTo(resolveClientRedirect(appBasePath.value, normalizedRedirectPath))
      return
    }
    if (isTauriRuntime()) {
      await startTauriLogin(normalizedRedirectPath)
      return
    }
    startWebLogin(normalizedRedirectPath)
  }

  async function handleWebCallback() {
    await init(true)
    const route = readCurrentRoute()
    const redirectPath = normalizeRedirectPath(String(route?.query.redirect || '/dashboard'))
    if (isAuthenticated.value) {
      return redirectPath
    }
    const query = new URLSearchParams()
    query.set('redirect', redirectPath)
    const authError = String(route?.query.auth_error || '').trim()
    if (authError) {
      query.set('auth_error', authError)
    }
    return `/login?${query.toString()}`
  }

  async function logout() {
    const isTauri = isTauriRuntime()
    const currentMode = mode.value

    clearSession()
    principal.value = currentMode === 'disabled' ? buildDisabledPrincipal() : null
    initialized.value = true

    if (!import.meta.client || currentMode === 'disabled') {
      return
    }

    if (isTauri) {
      await navigateTo(buildAppPath(appBasePath.value, '/login'))
      return
    }

    try {
      const response = await $fetch<{ redirect_url?: string }>('/v1/auth/logout', {
        method: 'POST',
        baseURL: gatewayBase.value,
        credentials: 'include'
      })
      const redirectURL = String(response.redirect_url || '').trim()
      if (redirectURL) {
        window.location.href = redirectURL
        return
      }
    } catch {
      // local state is already cleared; fall through to login page
    }

    await navigateTo(buildAppPath(appBasePath.value, '/login'))
  }

  function authHeaders(): Record<string, string> {
    if (mode.value !== 'oidc' || !isTauriRuntime() || !session.value?.accessToken) {
      return {}
    }
    return {
      Authorization: `Bearer ${session.value.accessToken}`
    }
  }

  async function gatewayFetch<T>(path: string, options: Record<string, any> = {}) {
    await init()
    try {
      return await $fetch<T>(path, buildGatewayOptions(options))
    } catch (error) {
      handleAuthenticatedRequestFailure(error)
      throw error
    }
  }

  async function gatewayFetchRaw<T>(path: string, options: Record<string, any> = {}) {
    await init()
    try {
      return await $fetch.raw<T>(path, buildGatewayOptions(options))
    } catch (error) {
      handleAuthenticatedRequestFailure(error)
      throw error
    }
  }

  function websocketUrl(path: string) {
    const base = `${toWebSocketBase(gatewayBase.value)}${path}`
    if (mode.value !== 'oidc' || !isTauriRuntime() || !session.value?.accessToken) {
      return base
    }
    const url = new URL(base)
    url.searchParams.set('access_token', session.value.accessToken)
    return url.toString()
  }

  function clearSession() {
    session.value = null
    principal.value = null
    clearStoredAuth(getSessionStore())
    clearStoredAuth(getPrincipalStore())
    if (!isTauriRuntime()) {
      clearLegacyWebAuthStorage()
    }
  }

  function handleAuthenticatedRequestFailure(error: unknown) {
    if (mode.value !== 'oidc') {
      return
    }

    const decision = resolveAuthenticatedRequest(error)
    if (decision.clearPersistedAuth) {
      clearSession()
    }
  }

  return {
    mode,
    principal,
    session,
    initialized,
    initializing,
    isAuthenticated,
    isAdmin,
    init,
    login,
    logout,
    handleWebCallback,
    authHeaders,
    gatewayFetch,
    gatewayFetchRaw,
    websocketUrl,
    clearSession
  }

  function buildGatewayOptions(options: Record<string, any>) {
    const headers = {
      ...(options.headers as Record<string, string> | undefined),
      ...authHeaders()
    }
    return {
      ...options,
      baseURL: gatewayBase.value,
      headers,
      credentials: mode.value === 'oidc' && !isTauriRuntime() ? 'include' : options.credentials
    }
  }

  function startWebLogin(redirectPath: string) {
    if (!import.meta.client) {
      return
    }
    const target = buildURLUnderBase(gatewayBase.value, '/v1/auth/login')
    target.searchParams.set('redirect', normalizeRedirectPath(redirectPath))
    window.location.href = target.toString()
  }

  async function startTauriLogin(redirectPath: string) {
    const { invoke } = await import('@tauri-apps/api/core')
    const result = await invoke<{
      access_token: string
      id_token?: string
      expires_in?: number
    }>('run_oidc_loopback_login', {
      issuerUrl: String(config.public.oidcIssuerUrl || '').trim(),
      clientId: String(config.public.oidcTauriClientId || '').trim(),
      scope: String(config.public.oidcScope || 'openid profile email').trim()
    })

    const authSession: AuthSession = {
      accessToken: result.access_token,
      idToken: result.id_token,
      expiresAt: result.expires_in ? Date.now() + result.expires_in * 1000 : undefined
    }
    const nextPrincipal = await fetchPrincipal(authSession.accessToken)
    setAuthenticatedState(authSession, nextPrincipal)
    initialized.value = true
    await navigateTo(resolveClientRedirect(appBasePath.value, redirectPath))
  }

  async function fetchPrincipal(accessToken?: string) {
    const headers: Record<string, string> = {}
    if (accessToken) {
      headers.Authorization = `Bearer ${accessToken}`
    }
    return await $fetch<Principal>('/v1/auth/me', {
      baseURL: gatewayBase.value,
      headers,
      credentials: accessToken ? undefined : 'include'
    })
  }

  function setSession(nextSession: AuthSession) {
    session.value = nextSession
    saveStoredSession(getSessionStore(), nextSession)
  }

  function setPrincipal(nextPrincipal: Principal) {
    principal.value = nextPrincipal
    saveStoredPrincipal(getPrincipalStore(), nextPrincipal)
  }

  function setAuthenticatedState(nextSession: AuthSession, nextPrincipal: Principal) {
    setSession(nextSession)
    setPrincipal(nextPrincipal)
  }

  function readCurrentRoute() {
    return router.currentRoute.value
  }

  function readRoutePath() {
    return readCurrentRoute().path
  }
}

function normalizeAuthMode(value: unknown): AuthMode {
  return String(value || 'disabled').trim() === 'oidc' ? 'oidc' : 'disabled'
}

function buildDisabledPrincipal(): Principal {
  return {
    subject: 'dev-admin',
    email: 'dev-admin@local',
    display_name: 'Dev Admin',
    role: 'admin',
    status: 'active',
    auth_mode: 'disabled',
    permissions: ['admin']
  }
}

function getPrincipalStore() {
  return getPrincipalStorage(isTauriRuntime())
}

function getSessionStore() {
  return getSessionStorage(isTauriRuntime())
}

function loadSession(): AuthSession | null {
  return loadStoredSession(getSessionStore())
}

function loadPrincipal(): Principal | null {
  return loadStoredPrincipal(getPrincipalStore())
}

function isTauriRuntime() {
  if (!import.meta.client) {
    return false
  }
  return typeof window !== 'undefined' && ('__TAURI_INTERNALS__' in window || '__TAURI__' in window)
}

export function normalizeRedirectPath(raw: string) {
  const trimmed = String(raw || '').trim()
  if (!trimmed || !trimmed.startsWith('/') || trimmed.startsWith('//')) {
    return '/dashboard'
  }
  return trimmed
}

export function resolveClientRedirect(appBasePath: string, raw: string) {
  return buildAppPath(appBasePath, normalizeRedirectPath(raw))
}
