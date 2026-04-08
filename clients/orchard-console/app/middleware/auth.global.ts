import { resolveAuthGuardDecision } from '~/utils/auth-guard'

export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) {
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
