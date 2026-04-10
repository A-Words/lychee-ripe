export type AuthGuardDecision =
  | { kind: 'allow' }
  | { kind: 'login', redirect: string }
  | { kind: 'dashboard' }

type AuthGuardInput = {
  path: string
  fullPath: string
  isServer: boolean
  isAuthenticated: boolean
  isAdmin: boolean
}

const PUBLIC_PREFIXES = ['/trace', '/login', '/auth/callback']
const PUBLIC_EXACT = ['/']

export function resolveAuthGuardDecision(input: AuthGuardInput): AuthGuardDecision {
  if (input.isServer) {
    return { kind: 'allow' }
  }

  const isPublic =
    PUBLIC_EXACT.includes(input.path)
    || PUBLIC_PREFIXES.some((prefix) => input.path === prefix || input.path.startsWith(`${prefix}/`))
  if (isPublic) {
    return { kind: 'allow' }
  }

  if (!input.isAuthenticated) {
    return {
      kind: 'login',
      redirect: input.fullPath
    }
  }

  if (input.path.startsWith('/admin') && !input.isAdmin) {
    return { kind: 'dashboard' }
  }

  return { kind: 'allow' }
}
