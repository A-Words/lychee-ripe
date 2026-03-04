import { buildTracePath } from '~/utils/trace-route'

export function buildDashboardTracePath(traceCode: string): string {
  return buildTracePath(traceCode)
}
