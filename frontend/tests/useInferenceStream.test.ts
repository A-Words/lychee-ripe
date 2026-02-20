import { describe, expect, it } from 'vitest'
import { toStreamWsUrl, useInferenceStream } from '../app/composables/useInferenceStream'

describe('useInferenceStream', () => {
  it('dispatches frame envelope', () => {
    const stream = useInferenceStream({ gatewayBase: 'http://127.0.0.1:9000' })

    stream.handleServerMessage({
      type: 'frame',
      model_version: 'v1',
      schema_version: 's1',
      result: {
        frame_index: 3,
        timestamp_ms: 120,
        detections: [],
        frame_summary: { total: 1, green: 0, half: 1, red: 0, young: 0 },
      },
    })

    expect(stream.frameResult.value?.frame_index).toBe(3)
    expect(stream.frameResult.value?.frame_summary.half).toBe(1)
    expect(stream.modelVersion.value).toBe('v1')
  })

  it('dispatches summary envelope', () => {
    const stream = useInferenceStream({ gatewayBase: 'http://127.0.0.1:9000' })

    stream.handleServerMessage({
      type: 'summary',
      model_version: 'v1',
      schema_version: 's1',
      summary: {
        total_detected: 6,
        ripeness_ratio: { green: 0.2, half: 0.1, red: 0.7, young: 0 },
        harvest_suggestion: 'ready',
      },
    })

    expect(stream.sessionSummary.value?.total_detected).toBe(6)
    expect(stream.sessionSummary.value?.harvest_suggestion).toBe('ready')
  })

  it('dispatches error envelope', () => {
    const stream = useInferenceStream({ gatewayBase: 'http://127.0.0.1:9000' })

    stream.handleServerMessage({ type: 'error', detail: 'empty frame' })

    expect(stream.lastError.value).toBe('empty frame')
  })

  it('builds ws url from gateway base', () => {
    expect(toStreamWsUrl('http://127.0.0.1:9000')).toBe('ws://127.0.0.1:9000/v1/infer/stream')
    expect(toStreamWsUrl('https://example.com')).toBe('wss://example.com/v1/infer/stream')
  })
})
