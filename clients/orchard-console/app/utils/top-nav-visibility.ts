import { getTraceFromQuery, isInternalTraceFrom } from '~/utils/trace-from'

export function shouldShowTopNav(
  path: string,
  queryFrom: string | string[] | null | undefined
): boolean {
  if (path === '/login' || path.startsWith('/auth/callback')) {
    return false
  }

  const isTraceRoute = path === '/trace' || path.startsWith('/trace/')
  if (!isTraceRoute) {
    return true
  }

  const from = getTraceFromQuery(queryFrom)
  return isInternalTraceFrom(from)
}
