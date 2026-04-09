export function buildURLUnderBase(base: string, targetPath: string): URL {
  const normalizedBase = ensureBaseTrailingSlash(base)
  const relativeTarget = String(targetPath || '')
    .trim()
    .replace(/^\/+/, "")
  return new URL(relativeTarget, normalizedBase)
}

function ensureBaseTrailingSlash(base: string): string {
  const trimmed = String(base || '').trim()
  return trimmed.endsWith('/') ? trimmed : `${trimmed}/`
}
