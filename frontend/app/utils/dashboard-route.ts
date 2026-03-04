import { buildTracePathWithFrom } from '~/utils/trace-from'

export function buildDashboardTracePath(traceCode: string): string {
  return buildTracePathWithFrom(traceCode, 'dashboard')
}
