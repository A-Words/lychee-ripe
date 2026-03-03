import { ref } from 'vue'
import type { DashboardOverviewResponse, ErrorResponse } from '../types/query'

export type DashboardOverviewStatus = 'idle' | 'loading' | 'success' | 'error'

export interface DashboardOverviewError extends Error {
  status?: number
  code?: string
  requestId?: string
}

interface UseDashboardOverviewOptions {
  gatewayBase?: string
  fetchImpl?: typeof fetch
}

interface FetchDashboardOptions {
  apiKey?: string
}

function resolveGatewayBase(provided?: string): string {
  if (provided) {
    return provided
  }

  try {
    return useRuntimeConfig().public.gatewayBase as string
  } catch {
    return 'http://127.0.0.1:9000'
  }
}

function toDashboardOverviewURL(gatewayBase: string): string {
  const url = new URL(gatewayBase)
  url.pathname = '/v1/dashboard/overview'
  url.search = ''
  url.hash = ''
  return url.toString()
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function asErrorPayload(payload: unknown): ErrorResponse | null {
  if (!isRecord(payload)) {
    return null
  }
  if (typeof payload.error !== 'string' || typeof payload.message !== 'string') {
    return null
  }
  return payload as unknown as ErrorResponse
}

function asDashboardOverview(payload: unknown): DashboardOverviewResponse | null {
  if (!isRecord(payload)) {
    return null
  }

  if (
    !isRecord(payload.totals) ||
    !isRecord(payload.status_distribution) ||
    !isRecord(payload.ripeness_distribution) ||
    !isRecord(payload.unripe_metrics) ||
    !isRecord(payload.reconcile_stats) ||
    !Array.isArray(payload.recent_anchors)
  ) {
    return null
  }

  if (typeof payload.totals.batch_total !== 'number') {
    return null
  }

  return payload as unknown as DashboardOverviewResponse
}

export function useDashboardOverview(options: UseDashboardOverviewOptions = {}) {
  const status = ref<DashboardOverviewStatus>('idle')
  const data = ref<DashboardOverviewResponse | null>(null)
  const lastError = ref<DashboardOverviewError | null>(null)

  const gatewayBase = resolveGatewayBase(options.gatewayBase)
  const fetchImpl = options.fetchImpl ?? fetch

  async function fetchOverview(fetchOptions: FetchDashboardOptions = {}): Promise<DashboardOverviewResponse> {
    status.value = 'loading'
    lastError.value = null

    const headers: Record<string, string> = {}
    const apiKey = fetchOptions.apiKey?.trim()
    if (apiKey) {
      headers['X-API-Key'] = apiKey
    }

    let response: Response
    try {
      response = await fetchImpl(toDashboardOverviewURL(gatewayBase), {
        method: 'GET',
        headers,
      })
    } catch (err) {
      const error = new Error(err instanceof Error ? err.message : 'network error') as DashboardOverviewError
      error.code = 'network_error'
      status.value = 'error'
      lastError.value = error
      throw error
    }

    let payload: unknown = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (!response.ok) {
      const errPayload = asErrorPayload(payload)
      const error = new Error(errPayload?.message ?? `request failed: ${response.status}`) as DashboardOverviewError
      error.status = response.status
      error.code = errPayload?.error
      error.requestId = errPayload?.request_id
      status.value = 'error'
      lastError.value = error
      throw error
    }

    const nextData = asDashboardOverview(payload)
    if (!nextData) {
      const error = new Error('invalid dashboard response payload') as DashboardOverviewError
      error.code = 'invalid_response'
      error.status = response.status
      status.value = 'error'
      lastError.value = error
      throw error
    }

    data.value = nextData
    status.value = 'success'
    return nextData
  }

  return {
    status,
    data,
    lastError,
    fetchOverview,
  }
}
