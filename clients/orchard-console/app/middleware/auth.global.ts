import { resolveAuthGuardDecision } from '~/utils/auth-guard'
import { buildAppPath, inferAppBasePath } from '~/utils/app-path'

export default defineNuxtRouteMiddleware(async (to, from) => {
  if (import.meta.server) {
    // This early return is only safe because the app is locked to ssr: false
    // and browser auth is restored on the client from gateway cookies plus
    // non-sensitive cached principal state. If SSR is enabled later, auth
    // storage and guard strategy must be redesigned first.
    return
  }

  const auth = useAuth()
  await auth.init()
  const appBasePath = inferAppBasePath(window.location.pathname, [from.path, to.path])

  const decision = resolveAuthGuardDecision({
    path: to.path,
    fullPath: to.fullPath,
    isServer: false,
    isAuthenticated: auth.isAuthenticated.value,
    isAdmin: auth.isAdmin.value
  })

  if (decision.kind === 'allow') {
    return
  }

  if (decision.kind === 'login') {
    return navigateTo({
      path: buildAppPath(appBasePath, '/login'),
      query: {
        redirect: decision.redirect
      }
    })
  }

  if (decision.kind === 'dashboard') {
    return navigateTo(buildAppPath(appBasePath, '/dashboard'))
  }
})
