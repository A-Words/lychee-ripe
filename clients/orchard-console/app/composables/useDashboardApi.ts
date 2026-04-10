import type { FetchError } from 'ofetch'
import type {
  DashboardApiError,
  DashboardErrorResponse,
  DashboardOverviewResponse
} from '~/types/dashboard'
import { useAuth } from '~/composables/useAuth'

const defaultDashboardApiError: DashboardApiError = {
  statusCode: 0,
  error: 'unknown_error',
  message: '获取看板数据失败，请稍后重试。'
}

export function useDashboardApi() {
  const auth = useAuth()

  const getOverview = async (): Promise<DashboardOverviewResponse> =>
    await auth.gatewayFetch<DashboardOverviewResponse>('/v1/dashboard/overview')

  const parseDashboardError = (error: unknown): DashboardApiError => {
    const fetchError = error as FetchError<DashboardErrorResponse>
    if (!fetchError || typeof fetchError !== 'object') {
      return { ...defaultDashboardApiError }
    }

    const statusCode = fetchError.statusCode || fetchError.response?.status || 0
    const payload = fetchError.data
    const message = mapDashboardErrorMessage(
      statusCode,
      payload?.message || fetchError.message || defaultDashboardApiError.message
    )

    return {
      statusCode,
      error: payload?.error || defaultDashboardApiError.error,
      message,
      requestId: payload?.request_id
    }
  }

  return {
    getOverview,
    parseDashboardError
  }
}

export function mapDashboardErrorMessage(statusCode: number, fallbackMessage: string): string {
  if (statusCode === 401 || statusCode === 403) {
    return '当前账号无权访问看板，或登录态已失效。'
  }
  if (statusCode === 503) {
    return fallbackMessage || '服务暂不可用，请稍后重试。'
  }
  return fallbackMessage || defaultDashboardApiError.message
}
