import { resolveAuthGuardDecision } from '~/utils/auth-guard'

export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) {
    // This early return is only safe because the app is locked to ssr: false
    // and browser auth state lives in localStorage. If SSR is enabled later,
    // auth storage and guard strategy must be redesigned first.
    return
  }

  const auth = useAuth()
  await auth.init()

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
      path: '/login',
      query: {
        redirect: decision.redirect
      }
    })
  }

  if (decision.kind === 'dashboard') {
    return navigateTo('/dashboard')
  }
})
