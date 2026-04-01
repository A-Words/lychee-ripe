import type { BatchStatus } from '~/types/trace'

type DashboardStatusColor = 'success' | 'warning' | 'error'

export interface DashboardStatusMeta {
  label: string
  color: DashboardStatusColor
  chartColor: string
}

export const DASHBOARD_STATUS_ORDER: BatchStatus[] = [
  'anchored',
  'pending_anchor',
  'anchor_failed'
]

export const DASHBOARD_STATUS_META: Record<BatchStatus, DashboardStatusMeta> = {
  anchored: {
    label: '已上链',
    color: 'success',
    chartColor: '#3D8D40'
  },
  pending_anchor: {
    label: '待补链',
    color: 'warning',
    chartColor: '#F5A623'
  },
  anchor_failed: {
    label: '补链失败',
    color: 'error',
    chartColor: '#D64545'
  }
}
