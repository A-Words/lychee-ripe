export const TRACE_DEMO_CODE = 'TRC-DEMO-0001'

export interface AppNavigationItem {
  key: 'index' | 'trace' | 'dashboard'
  label: string
  to: string
  matchPrefix: string
}

export const APP_NAV_ITEMS: AppNavigationItem[] = [
  {
    key: 'index',
    label: '识别建批',
    to: '/',
    matchPrefix: '/',
  },
  {
    key: 'trace',
    label: '溯源查询',
    to: `/trace/${TRACE_DEMO_CODE}`,
    matchPrefix: '/trace',
  },
  {
    key: 'dashboard',
    label: '数据看板',
    to: '/dashboard',
    matchPrefix: '/dashboard',
  },
]

export function buildTracePath(code: string): string {
  return `/trace/${encodeURIComponent(code.trim())}`
}
