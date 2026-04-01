import { describe, expect, it } from 'vitest'
import {
  applyDetectionsToSession,
  buildSessionAggregateSummary,
  createSessionAggregateState
} from '../../app/utils/session-aggregator'
import type { Detection } from '../../app/types/infer'

function detection(ripeness: Detection['ripeness'], trackId: number | null): Detection {
  return {
    bbox: [0, 0, 10, 10],
    class_name: 'lychee',
    ripeness,
    confidence: 0.9,
    track_id: trackId
  }
}

describe('session aggregator', () => {
  it('deduplicates detections by track_id', () => {
    let state = createSessionAggregateState()
    state = applyDetectionsToSession(state, [
      detection('red', 1),
      detection('half', 2),
      detection('red', 1)
    ])

    const summary = buildSessionAggregateSummary(state)
    expect(summary.total).toBe(2)
    expect(summary.red).toBe(1)
    expect(summary.half).toBe(1)
  })

  it('counts detections without track_id as new entries', () => {
    let state = createSessionAggregateState()
    state = applyDetectionsToSession(state, [
      detection('green', null),
      detection('young', null),
      detection('green', null)
    ])

    const summary = buildSessionAggregateSummary(state)
    expect(summary.total).toBe(3)
    expect(summary.green).toBe(2)
    expect(summary.young).toBe(1)
    expect(summary.unripe_ratio).toBe(1)
  })

  it('uses backend-compatible harvest suggestion rules', () => {
    let state = createSessionAggregateState()
    state = applyDetectionsToSession(state, [
      detection('red', 1),
      detection('red', 2),
      detection('red', 3),
      detection('red', 4),
      detection('half', 5)
    ])

    const summary = buildSessionAggregateSummary(state)
    expect(summary.harvest_suggestion).toBe('ready')
  })
})
