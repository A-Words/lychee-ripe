import type { FetchError } from 'ofetch'
import type { ErrorResponse, TraceApiError, TraceResponse } from '~/types/trace'
import { normalizeTraceCode } from '~/utils/trace-route'

const defaultTraceApiError: TraceApiError = {
  statusCode: 0,
  error: 'unknown_error',
  message: '请求失败，请稍后重试。'
}

export function useTraceApi() {
  const gatewayBase = useGatewayBase()

  const getPublicTrace = async (traceCode: string): Promise<TraceResponse> => {
    const code = normalizeTraceCode(traceCode)
    return await $fetch<TraceResponse>(`/v1/trace/${encodeURIComponent(code)}`, {
      baseURL: gatewayBase.value
    })
  }

  const parseTraceError = (error: unknown): TraceApiError => {
    const fetchError = error as FetchError<ErrorResponse>
    if (!fetchError || typeof fetchError !== 'object') {
      return { ...defaultTraceApiError }
    }

    const statusCode = fetchError.statusCode || fetchError.response?.status || 0
    const payload = fetchError.data

    return {
      statusCode,
      error: payload?.error || defaultTraceApiError.error,
      message: payload?.message || fetchError.message || defaultTraceApiError.message,
      requestId: payload?.request_id
    }
  }

  return {
    gatewayBase,
    getPublicTrace,
    parseTraceError
  }
}
