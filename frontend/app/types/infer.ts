export type RipenessLabel = 'green' | 'half' | 'red' | 'young'

export interface Detection {
  bbox: [number, number, number, number]
  class_name: 'lychee'
  ripeness: RipenessLabel
  confidence: number
  track_id: number | null
}

export interface FrameSummary {
  total: number
  green: number
  half: number
  red: number
  young: number
}

export interface FrameResult {
  frame_index: number
  timestamp_ms: number
  detections: Detection[]
  frame_summary: FrameSummary
}

export interface RipenessRatio {
  green: number
  half: number
  red: number
  young: number
}

export interface SessionSummary {
  total_detected: number
  ripeness_ratio: RipenessRatio
  harvest_suggestion: 'not_ready' | 'partially_ready' | 'ready' | 'overripe_risk'
}

export interface StreamFrameEnvelope {
  type: 'frame'
  model_version: string
  schema_version: string
  result: FrameResult
}

export interface StreamSummaryEnvelope {
  type: 'summary'
  model_version: string
  schema_version: string
  summary: SessionSummary
}

export interface StreamErrorEnvelope {
  type: 'error'
  detail: string
}

export type StreamEnvelope = StreamFrameEnvelope | StreamSummaryEnvelope | StreamErrorEnvelope
