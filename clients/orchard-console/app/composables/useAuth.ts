import type { AuthMode, AuthSession, OIDCDiscoveryDocument, Principal } from '~/types/auth'
import { toWebSocketBase } from '~/utils/ws-url'

const AUTH_SESSION_KEY = 'lychee-ripe.auth.session'
const AUTH_PENDING_KEY = 'lychee-ripe.auth.pending'

type PendingLoginState = {
  state: string
  codeVerifier: string
  redirectPath: string
}

let initPromise: Promise<void> | null = null

export function useAuth() {
  const config = useRuntimeConfig()
  const gatewayBase = useGatewayBase()
  const mode = computed<AuthMode>(() => normalizeAuthMode(config.public.authMode))
  const principal = useState<Principal | null>('auth.principal', () => null)
  const session = useState<AuthSession | null>('auth.session', () => null)
  const initialized = useState<boolean>('auth.initialized', () => false)
  const initializing = useState<boolean>('auth.initializing', () => false)

  const isAuthenticated = computed(() => mode.value === 'disabled' || Boolean(principal.value))
  const isAdmin = computed(() => mode.value === 'disabled' || principal.value?.role === 'admin')

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

        session.value = loadSession()
        if (!session.value?.accessToken) {
          principal.value = null
          initialized.value = true
          return
        }

        try {
          principal.value = await fetchPrincipal(session.value.accessToken)
        } catch {
          clearSession()
          principal.value = null
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
    if (mode.value === 'disabled') {
      await navigateTo(redirectPath)
      return
    }
    if (isTauriRuntime()) {
      await startTauriLogin(redirectPath)
      return
    }
    await startWebLogin(redirectPath)
  }

  async function handleWebCallback(query: { code?: string | null, state?: string | null }) {
    const code = query.code?.trim()
    const state = query.state?.trim()
    const pending = loadPendingState()
    if (!code || !state || !pending || pending.state !== state) {
      clearPendingState()
      throw new Error('invalid callback state')
    }

    const discovery = await fetchDiscovery()
    const redirectUri = String(config.public.oidcWebRedirectUri || '').trim()
    const token = await exchangeAuthorizationCode({
      tokenEndpoint: discovery.token_endpoint,
      clientId: String(config.public.oidcWebClientId || '').trim(),
      code,
      codeVerifier: pending.codeVerifier,
      redirectUri
    })

    setSession(token)
    principal.value = await fetchPrincipal(token.accessToken)
    initialized.value = true

    const target = pending.redirectPath || '/dashboard'
    clearPendingState()
    return target
  }

  async function logout() {
    const currentSession = session.value
    clearSession()
    principal.value = mode.value === 'disabled' ? buildDisabledPrincipal() : null
    initialized.value = true

    if (!import.meta.client || mode.value === 'disabled') {
      return
    }

    if (isTauriRuntime()) {
      await navigateTo('/login')
      return
    }

    try {
      const discovery = await fetchDiscovery()
      if (discovery.end_session_endpoint && currentSession?.idToken) {
        const url = new URL(discovery.end_session_endpoint)
        url.searchParams.set('id_token_hint', currentSession.idToken)
        const postLogout = String(config.public.oidcWebPostLogoutRedirectUri || '').trim()
        if (postLogout) {
          url.searchParams.set('post_logout_redirect_uri', postLogout)
        }
        window.location.href = url.toString()
        return
      }
    } catch {
      // fall through to local redirect
    }

    await navigateTo('/login')
  }

  function authHeaders(): Record<string, string> {
    if (mode.value === 'disabled') {
      return {}
    }
    if (!session.value?.accessToken) {
      return {}
    }
    return {
      Authorization: `Bearer ${session.value.accessToken}`
    }
  }

  async function gatewayFetch<T>(path: string, options: Record<string, any> = {}) {
    await init()
    const headers = {
      ...(options.headers as Record<string, string> | undefined),
      ...authHeaders()
    }
    return await $fetch<T>(path, {
      ...options,
      baseURL: gatewayBase.value,
      headers
    })
  }

  async function gatewayFetchRaw<T>(path: string, options: Record<string, any> = {}) {
    await init()
    const headers = {
      ...(options.headers as Record<string, string> | undefined),
      ...authHeaders()
    }
    return await $fetch.raw<T>(path, {
      ...options,
      baseURL: gatewayBase.value,
      headers
    })
  }

  function websocketUrl(path: string) {
    const base = `${toWebSocketBase(gatewayBase.value)}${path}`
    if (mode.value === 'disabled' || !session.value?.accessToken) {
      return base
    }
    const url = new URL(base)
    url.searchParams.set('access_token', session.value.accessToken)
    return url.toString()
  }

  function clearSession() {
    session.value = null
    if (import.meta.client) {
      localStorage.removeItem(AUTH_SESSION_KEY)
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

  async function startWebLogin(redirectPath: string) {
    const discovery = await fetchDiscovery()
    const clientId = String(config.public.oidcWebClientId || '').trim()
    const redirectUri = String(config.public.oidcWebRedirectUri || '').trim()
    const scope = String(config.public.oidcScope || 'openid profile email').trim()
    if (!clientId || !redirectUri) {
      throw new Error('missing oidc web configuration')
    }

    const state = randomString()
    const codeVerifier = randomString(64)
    const codeChallenge = await pkceChallenge(codeVerifier)
    savePendingState({ state, codeVerifier, redirectPath })

    const url = new URL(discovery.authorization_endpoint)
    url.searchParams.set('client_id', clientId)
    url.searchParams.set('response_type', 'code')
    url.searchParams.set('scope', scope)
    url.searchParams.set('redirect_uri', redirectUri)
    url.searchParams.set('state', state)
    url.searchParams.set('code_challenge', codeChallenge)
    url.searchParams.set('code_challenge_method', 'S256')

    window.location.href = url.toString()
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
    setSession(authSession)
    principal.value = await fetchPrincipal(authSession.accessToken)
    initialized.value = true
    await navigateTo(redirectPath)
  }

  async function fetchDiscovery(): Promise<OIDCDiscoveryDocument> {
    const issuer = String(config.public.oidcIssuerUrl || '').trim().replace(/\/+$/, '')
    if (!issuer) {
      throw new Error('missing oidc issuer')
    }
    return await $fetch<OIDCDiscoveryDocument>(`${issuer}/.well-known/openid-configuration`)
  }

  async function fetchPrincipal(accessToken: string) {
    return await $fetch<Principal>('/v1/auth/me', {
      baseURL: gatewayBase.value,
      headers: {
        Authorization: `Bearer ${accessToken}`
      }
    })
  }

  async function exchangeAuthorizationCode(input: {
    tokenEndpoint: string
    clientId: string
    code: string
    codeVerifier: string
    redirectUri: string
  }): Promise<AuthSession> {
    const form = new URLSearchParams({
      grant_type: 'authorization_code',
      client_id: input.clientId,
      code: input.code,
      code_verifier: input.codeVerifier,
      redirect_uri: input.redirectUri
    })

    const response = await $fetch<{
      access_token: string
      id_token?: string
      expires_in?: number
    }>(input.tokenEndpoint, {
      method: 'POST',
      body: form,
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      }
    })

    return {
      accessToken: response.access_token,
      idToken: response.id_token,
      expiresAt: response.expires_in ? Date.now() + response.expires_in * 1000 : undefined
    }
  }

  function setSession(nextSession: AuthSession) {
    session.value = nextSession
    if (import.meta.client) {
      localStorage.setItem(AUTH_SESSION_KEY, JSON.stringify(nextSession))
    }
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

function loadSession(): AuthSession | null {
  if (!import.meta.client) {
    return null
  }
  const raw = localStorage.getItem(AUTH_SESSION_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    localStorage.removeItem(AUTH_SESSION_KEY)
    return null
  }
}

function savePendingState(state: PendingLoginState) {
  if (!import.meta.client) {
    return
  }
  localStorage.setItem(AUTH_PENDING_KEY, JSON.stringify(state))
}

function loadPendingState(): PendingLoginState | null {
  if (!import.meta.client) {
    return null
  }
  const raw = localStorage.getItem(AUTH_PENDING_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as PendingLoginState
  } catch {
    localStorage.removeItem(AUTH_PENDING_KEY)
    return null
  }
}

function clearPendingState() {
  if (import.meta.client) {
    localStorage.removeItem(AUTH_PENDING_KEY)
  }
}

function isTauriRuntime() {
  if (!import.meta.client) {
    return false
  }
  return typeof window !== 'undefined' && ('__TAURI_INTERNALS__' in window || '__TAURI__' in window)
}

function randomString(length = 43) {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~'
  const buffer = new Uint8Array(length)
  crypto.getRandomValues(buffer)
  return Array.from(buffer, (item) => alphabet[item % alphabet.length]).join('')
}

async function pkceChallenge(verifier: string) {
  const data = new TextEncoder().encode(verifier)
  const digest = await crypto.subtle.digest('SHA-256', data)
  return base64UrlEncode(new Uint8Array(digest))
}

function base64UrlEncode(input: Uint8Array) {
  const binary = Array.from(input, (item) => String.fromCharCode(item)).join('')
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}
