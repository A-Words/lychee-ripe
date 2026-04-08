const PUBLIC_PREFIXES = ['/trace', '/login', '/auth/callback']
const PUBLIC_EXACT = ['/']

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuth()
  await auth.init()

  const isPublic = PUBLIC_EXACT.includes(to.path) || PUBLIC_PREFIXES.some((prefix) => to.path === prefix || to.path.startsWith(`${prefix}/`))
  if (isPublic) {
    return
  }

  if (!auth.isAuthenticated.value) {
    return navigateTo({
      path: '/login',
      query: {
        redirect: to.fullPath
      }
    })
  }

  if (to.path.startsWith('/admin') && !auth.isAdmin.value) {
    return navigateTo('/dashboard')
  }
})
