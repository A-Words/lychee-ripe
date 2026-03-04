import type { BatchStatus } from '~/types/trace'

export type UnripeHandling = 'sorted_out'

export interface BatchSummaryInput {
  total: number
  green: number
  half: number
  red: number
  young: number
}

export interface BatchSummary extends BatchSummaryInput {
  unripe_count: number
  unripe_ratio: number
  unripe_handling: UnripeHandling
}

export interface BatchCreateRequest {
  orchard_id: string
  orchard_name: string
  plot_id: string
  plot_name?: string
  harvested_at: string
  summary: BatchSummaryInput
  note?: string
  confirm_unripe?: boolean
}

export interface AnchorProof {
  tx_hash: string
  block_number: number
  chain_id: string
  contract_address: string
  anchor_hash: string
  anchored_at: string
}

export interface Batch {
  batch_id: string
  trace_code: string
  status: BatchStatus
  orchard_id: string
  orchard_name: string
  plot_id: string
  plot_name?: string | null
  harvested_at: string
  summary: BatchSummary
  note?: string
  created_at: string
  anchor_proof?: AnchorProof | null
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}

export interface BatchCreateApiError {
  statusCode: number
  error: string
  message: string
  requestId?: string
}

export interface BatchCreateResult {
  statusCode: number
  data: Batch
}

export interface BatchCreateFormInput {
  orchard_id: string
  orchard_name: string
  plot_id: string
  plot_name?: string
  harvested_at: string
  note?: string
  confirm_unripe: boolean
}
