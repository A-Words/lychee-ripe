import { ref } from 'vue'
import type { ErrorResponse, TraceResponse } from '../types/query'

export type PublicTraceStatus = 'idle' | 'loading' | 'success' | 'error'

export interface PublicTraceError extends Error {
  status?: number
  code?: string
  requestId?: string
}

interface UsePublicTraceOptions {
  gatewayBase?: string
  fetchImpl?: typeof fetch
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

function toTraceURL(gatewayBase: string, traceCode: string): string {
  const url = new URL(gatewayBase)
  url.pathname = `/v1/trace/${encodeURIComponent(traceCode)}`
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

function asTraceResponse(payload: unknown): TraceResponse | null {
  if (!isRecord(payload) || !isRecord(payload.batch) || !isRecord(payload.verify_result)) {
    return null
  }
  if (typeof payload.batch.batch_id !== 'string' || typeof payload.batch.trace_code !== 'string') {
    return null
  }
  if (typeof payload.verify_result.verify_status !== 'string' || typeof payload.verify_result.reason !== 'string') {
    return null
  }
  return payload as unknown as TraceResponse
}

export function usePublicTrace(options: UsePublicTraceOptions = {}) {
  const status = ref<PublicTraceStatus>('idle')
  const data = ref<TraceResponse | null>(null)
  const lastError = ref<PublicTraceError | null>(null)

  const gatewayBase = resolveGatewayBase(options.gatewayBase)
  const fetchImpl = options.fetchImpl ?? fetch

  async function fetchTrace(traceCodeRaw: string): Promise<TraceResponse> {
    const traceCode = traceCodeRaw.trim()
    if (!traceCode) {
      const error = new Error('trace code is required') as PublicTraceError
      error.code = 'invalid_trace_code'
      status.value = 'error'
      lastError.value = error
      throw error
    }

    status.value = 'loading'
    lastError.value = null

    let response: Response
    try {
      response = await fetchImpl(toTraceURL(gatewayBase, traceCode), {
        method: 'GET',
      })
    } catch (err) {
      const error = new Error(err instanceof Error ? err.message : 'network error') as PublicTraceError
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
      const error = new Error(errPayload?.message ?? `request failed: ${response.status}`) as PublicTraceError
      error.status = response.status
      error.code = errPayload?.error
      error.requestId = errPayload?.request_id
      status.value = 'error'
      lastError.value = error
      throw error
    }

    const nextData = asTraceResponse(payload)
    if (!nextData) {
      const error = new Error('invalid trace response payload') as PublicTraceError
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
    fetchTrace,
  }
}
