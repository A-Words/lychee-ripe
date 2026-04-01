export function normalizeTraceCode(input: string): string {
  return input.trim().toUpperCase()
}

export function buildTracePath(traceCode: string): string {
  const normalized = normalizeTraceCode(traceCode)
  return `/trace/${encodeURIComponent(normalized)}`
}

export function getTraceCodeFromRouteParam(param: string | string[] | undefined): string {
  if (Array.isArray(param)) {
    return normalizeTraceCode(param[0] || '')
  }
  return normalizeTraceCode(param || '')
}
