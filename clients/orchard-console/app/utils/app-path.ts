const DUMMY_ORIGIN = 'https://app.local'

export function inferAppBasePath(browserPathname: string, routePath: string | string[]): string {
  const currentPath = normalizePathname(browserPathname)
  const routeCandidates = Array.isArray(routePath) ? routePath : [routePath]

  for (const candidate of routeCandidates) {
    const currentRoute = normalizeRoutePath(candidate)
    const inferred = inferBasePathFromCandidate(currentPath, currentRoute)
    if (inferred !== null) {
      return inferred
    }
  }

  return ''
}

function inferBasePathFromCandidate(currentPath: string, currentRoute: string): string | null {
  if (currentRoute === '/') {
    return currentPath === '/' ? '' : trimTrailingSlash(currentPath)
  }
  if (currentPath === currentRoute) {
    return ''
  }
  if (!currentPath.endsWith(currentRoute)) {
    return null
  }

  return trimTrailingSlash(currentPath.slice(0, currentPath.length - currentRoute.length))
}

export function buildAppPath(appBasePath: string, targetPath: string): string {
  const normalizedBase = normalizeBasePath(appBasePath)
  const normalizedTarget = String(targetPath || '').trim().replace(/^\/+/, '')
  const url = new URL(normalizedTarget, `${DUMMY_ORIGIN}${normalizedBase}/`)
  return `${url.pathname}${url.search}${url.hash}`
}

function normalizePathname(pathname: string): string {
  const trimmed = String(pathname || '').trim()
  if (!trimmed) {
    return '/'
  }
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

function normalizeRoutePath(path: string): string {
  const trimmed = String(path || '').trim()
  if (!trimmed || trimmed === '/') {
    return '/'
  }
  return trimTrailingSlash(trimmed.startsWith('/') ? trimmed : `/${trimmed}`)
}

function normalizeBasePath(path: string): string {
  const trimmed = trimTrailingSlash(String(path || '').trim())
  if (!trimmed || trimmed === '/') {
    return ''
  }
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

function trimTrailingSlash(path: string): string {
  if (!path || path === '/') {
    return path || ''
  }
  return path.replace(/\/+$/, '')
}
