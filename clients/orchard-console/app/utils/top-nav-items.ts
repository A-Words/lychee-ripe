import { buildTraceLandingPathWithFrom } from '~/utils/trace-from'

export type TopNavKey = 'batch_create' | 'dashboard' | 'trace' | 'admin'

type TopNavItemDefinition = {
  key: TopNavKey
  label: string
  to: string
  icon: string
  activePrefixes: string[]
}

export type TopNavItem = {
  key: TopNavKey
  label: string
  to: string
  icon: string
  active: boolean
}

const BASE_TOP_NAV_DEFINITIONS: TopNavItemDefinition[] = [
  {
    key: 'batch_create',
    label: '识别建批',
    to: '/batch/create',
    icon: 'i-lucide-camera',
    activePrefixes: ['/batch/create']
  },
  {
    key: 'dashboard',
    label: '数据看板',
    to: '/dashboard',
    icon: 'i-lucide-chart-pie',
    activePrefixes: ['/dashboard']
  },
  {
    key: 'trace',
    label: '溯源查询',
    to: buildTraceLandingPathWithFrom('index'),
    icon: 'i-lucide-search',
    activePrefixes: ['/trace']
  },
  {
    key: 'admin',
    label: '管理后台',
    to: '/admin',
    icon: 'i-lucide-shield-check',
    activePrefixes: ['/admin']
  }
]

function isNavItemActive(path: string, prefixes: string[]): boolean {
  return prefixes.some((prefix) => path.startsWith(prefix))
}

export function buildTopNavItems(path: string, includeAdmin = false): TopNavItem[] {
  const items = includeAdmin
    ? BASE_TOP_NAV_DEFINITIONS
    : BASE_TOP_NAV_DEFINITIONS.filter((item) => item.key !== 'admin')

  return items.map((item) => ({
    key: item.key,
    label: item.label,
    to: item.to,
    icon: item.icon,
    active: isNavItemActive(path, item.activePrefixes)
  }))
}
