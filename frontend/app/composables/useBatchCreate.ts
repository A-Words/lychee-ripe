import { ref } from 'vue'
import type {
  BatchCreateRequest,
  BatchResponse,
  BatchSummaryInput,
  BatchStatus,
  ErrorResponse,
} from '../types/batch'

export const DEFAULT_UNRIPE_THRESHOLD = 0.15

export type BatchCreateStatus = 'idle' | 'submitting' | 'success' | 'error'

export interface BatchCreateResult {
  statusCode: 201 | 202
  batch: BatchResponse
}

export interface BatchCreateError extends Error {
  status?: number
  code?: string
  requestId?: string
}

interface UseBatchCreateOptions {
  gatewayBase?: string
  apiKey?: string
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

function toBatchCreateURL(gatewayBase: string): string {
  const url = new URL(gatewayBase)
  url.pathname = '/v1/batches'
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

function asBatchPayload(payload: unknown): BatchResponse | null {
  if (!isRecord(payload)) {
    return null
  }
  if (typeof payload.batch_id !== 'string' || typeof payload.trace_code !== 'string') {
    return null
  }
  if (typeof payload.status !== 'string') {
    return null
  }
  return payload as unknown as BatchResponse
}

export function computeUnripeMetrics(summary: BatchSummaryInput) {
  const total = Math.max(0, summary.total)
  const unripeCount = Math.max(0, summary.green) + Math.max(0, summary.young)
  const unripeRatio = total > 0 ? unripeCount / total : 0
  return {
    unripeCount,
    unripeRatio,
  }
}

export function needsUnripeConfirm(summary: BatchSummaryInput, threshold = DEFAULT_UNRIPE_THRESHOLD): boolean {
  return computeUnripeMetrics(summary).unripeRatio > threshold
}

export function getBatchStatusNotice(status: BatchStatus): { title: string; description: string; color: 'success' | 'warning' | 'error' } {
  if (status === 'anchored') {
    return {
      title: '批次已锚定',
      description: '摘要已写入链上，可直接用于公开溯源。',
      color: 'success',
    }
  }
  if (status === 'pending_anchor') {
    return {
      title: '批次已保存，待补链',
      description: '链路暂不可用，系统会在后续自动/手动补链。',
      color: 'warning',
    }
  }
  return {
    title: '批次锚定失败',
    description: '该批次处于 anchor_failed 状态，需后续人工处理。',
    color: 'error',
  }
}

export function useBatchCreate(options: UseBatchCreateOptions = {}) {
  const status = ref<BatchCreateStatus>('idle')
  const lastError = ref<BatchCreateError | null>(null)
  const lastResult = ref<BatchCreateResult | null>(null)

  const gatewayBase = resolveGatewayBase(options.gatewayBase)
  const fetchImpl = options.fetchImpl ?? fetch

  async function createBatch(input: BatchCreateRequest): Promise<BatchCreateResult> {
    status.value = 'submitting'
    lastError.value = null

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    const apiKey = options.apiKey?.trim()
    if (apiKey) {
      headers['X-API-Key'] = apiKey
    }

    let response: Response
    try {
      response = await fetchImpl(toBatchCreateURL(gatewayBase), {
        method: 'POST',
        headers,
        body: JSON.stringify(input),
      })
    } catch (err) {
      const error = new Error(err instanceof Error ? err.message : 'network error') as BatchCreateError
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
      const error = new Error(errPayload?.message ?? `request failed: ${response.status}`) as BatchCreateError
      error.status = response.status
      error.code = errPayload?.error
      error.requestId = errPayload?.request_id
      status.value = 'error'
      lastError.value = error
      throw error
    }

    const batch = asBatchPayload(payload)
    if (!batch) {
      const error = new Error('invalid batch response payload') as BatchCreateError
      error.status = response.status
      error.code = 'invalid_response'
      status.value = 'error'
      lastError.value = error
      throw error
    }

    const statusCode = response.status === 202 ? 202 : 201
    const result: BatchCreateResult = {
      statusCode,
      batch,
    }
    status.value = 'success'
    lastResult.value = result
    return result
  }

  return {
    status,
    lastError,
    lastResult,
    createBatch,
  }
}
