import { buildTracePath } from '~/utils/trace-route'

export type TraceFromSource = 'dashboard' | 'batch_create' | 'index' | 'unknown'

type TraceBackTarget = {
  to: string
  label: string
}

export function buildTracePathWithFrom(
  traceCode: string,
  from: Exclude<TraceFromSource, 'unknown'>
): string {
  const path = buildTracePath(traceCode)
  return `${path}?from=${encodeURIComponent(from)}`
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
