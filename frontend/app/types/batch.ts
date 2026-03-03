export interface BatchSummaryInput {
  total: number
  green: number
  half: number
  red: number
  young: number
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

export interface BatchSummary extends BatchSummaryInput {
  unripe_count: number
  unripe_ratio: number
  unripe_handling: 'sorted_out'
}

export type BatchStatus = 'pending_anchor' | 'anchored' | 'anchor_failed'

export interface AnchorProof {
  tx_hash: string
  block_number: number
  chain_id: string
  contract_address: string
  anchor_hash: string
  anchored_at: string
}

export interface BatchResponse {
  batch_id: string
  trace_code: string
  status: BatchStatus
  orchard_id: string
  orchard_name: string
  plot_id: string
  plot_name: string | null
  harvested_at: string
  summary: BatchSummary
  note: string | null
  created_at: string
  anchor_proof: AnchorProof | null
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}
