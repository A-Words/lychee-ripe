import { buildTracePath } from '~/utils/trace-route'

export type TraceFromSource = 'dashboard' | 'batch_create' | 'index' | 'unknown'
export type InternalTraceFrom = Exclude<TraceFromSource, 'unknown'>

type TraceBackTarget = {
  to: string
  label: string
}

export function buildTracePathWithFrom(
  traceCode: string,
  from: InternalTraceFrom
): string {
  const path = buildTracePath(traceCode)
  return `${path}?from=${encodeURIComponent(from)}`
}

export function buildTraceLandingPathWithFrom(from: InternalTraceFrom): string {
  return `/trace?from=${encodeURIComponent(from)}`
}

export function getTraceFromQuery(value: string | string[] | null | undefined): TraceFromSource {
  const raw = Array.isArray(value) ? value[0] : value
  if (typeof raw !== 'string') {
    return 'unknown'
  }

  const normalized = raw.trim().toLowerCase()
  if (normalized === 'dashboard') {
    return 'dashboard'
  }
  if (normalized === 'batch_create') {
    return 'batch_create'
  }
  if (normalized === 'index') {
    return 'index'
  }
  return 'unknown'
}

export function isInternalTraceFrom(from: TraceFromSource): from is InternalTraceFrom {
  return from !== 'unknown'
}

export function buildTraceDetailPathFromQuery(
  traceCode: string,
  queryFrom: string | string[] | null | undefined
): string {
  const from = getTraceFromQuery(queryFrom)
  if (!isInternalTraceFrom(from)) {
    return buildTracePath(traceCode)
  }
  return buildTracePathWithFrom(traceCode, from)
}

export function getTraceBackTarget(from: TraceFromSource): TraceBackTarget | null {
  if (from === 'dashboard') {
    return {
      to: '/dashboard',
      label: '返回数据看板'
    }
  }
  if (from === 'batch_create') {
    return {
      to: '/batch/create',
      label: '返回识别建批'
    }
  }
  if (from === 'index') {
    return {
      to: '/',
      label: '返回首页'
    }
  }
  return null
}
