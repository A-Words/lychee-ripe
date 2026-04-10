import type { FetchError } from 'ofetch'
import type {
  Batch,
  BatchCreateApiError,
  BatchCreateFormInput,
  BatchCreateRequest,
  BatchCreateResult,
  BatchSummaryInput,
  ErrorResponse
} from '~/types/batch'
import { useAuth } from '~/composables/useAuth'

const defaultCreateError: BatchCreateApiError = {
  statusCode: 0,
  error: 'unknown_error',
  message: '建批请求失败，请稍后重试。'
}

export function useBatchCreate() {
  const auth = useAuth()

  const createBatch = async (payload: BatchCreateRequest): Promise<BatchCreateResult> => {
    const response = await auth.gatewayFetchRaw<Batch>('/v1/batches', {
      method: 'POST',
      body: payload
    })
    if (!response._data) {
      throw new Error('empty response payload')
    }

    return {
      statusCode: response.status,
      data: response._data
    }
  }

  const parseCreateError = (error: unknown): BatchCreateApiError => {
    const fetchError = error as FetchError<ErrorResponse>
    if (!fetchError || typeof fetchError !== 'object') {
      return { ...defaultCreateError }
    }

    const statusCode = fetchError.statusCode || fetchError.response?.status || 0
    const payload = fetchError.data
    const responseMessage = payload?.message || fetchError.message || defaultCreateError.message
    const mapped = mapBatchErrorMessage(statusCode, responseMessage)

    return {
      statusCode,
      error: payload?.error || defaultCreateError.error,
      message: mapped,
      requestId: payload?.request_id
    }
  }

  return {
    createBatch,
    parseCreateError
  }
}

export function buildBatchCreateRequest(
  formInput: BatchCreateFormInput,
  summary: BatchSummaryInput
): BatchCreateRequest {
  const payload: BatchCreateRequest = {
    orchard_id: formInput.orchard_id.trim(),
    orchard_name: formInput.orchard_name.trim(),
    plot_id: formInput.plot_id.trim(),
    harvested_at: formInput.harvested_at,
    summary,
    confirm_unripe: formInput.confirm_unripe
  }

  const plotName = formInput.plot_name?.trim()
  if (plotName) {
    payload.plot_name = plotName
  }

  const note = formInput.note?.trim()
  if (note) {
    payload.note = note
  }

  return payload
}

export function validateBatchSummaryInput(summary: BatchSummaryInput): string | null {
  const values = [summary.green, summary.half, summary.red, summary.young, summary.total]
  if (values.some((value) => !Number.isFinite(value) || !Number.isInteger(value))) {
    return '识别汇总存在非整数计数，请重新识别后提交。'
  }

  if (summary.total <= 0) {
    return '当前会话无有效识别结果，无法建批。'
  }

  if (summary.green < 0 || summary.half < 0 || summary.red < 0 || summary.young < 0) {
    return '识别汇总存在负数计数，请重试。'
  }

  const partsSum = summary.green + summary.half + summary.red + summary.young
  if (partsSum !== summary.total) {
    return '识别汇总与总数不一致，请重新识别。'
  }

  return null
}

export function toRFC3339FromLocal(localValue: string): string | null {
  const value = localValue.trim()
  if (!value) {
    return null
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  return date.toISOString()
}

export function mapBatchErrorMessage(statusCode: number, fallbackMessage: string): string {
  if (statusCode === 400) {
    return fallbackMessage || '请求参数非法，请检查采摘信息与汇总结果。'
  }
  if (statusCode === 401 || statusCode === 403) {
    return '当前账号无权执行建批，或登录态已失效。请重新登录后重试。'
  }
  if (statusCode === 409) {
    return fallbackMessage || '批次冲突，请重新发起建批。'
  }
  if (statusCode === 503) {
    return fallbackMessage || '服务暂不可用，请稍后重试。'
  }
  return fallbackMessage || defaultCreateError.message
}
