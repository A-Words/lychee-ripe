import type { BatchSummaryInput } from '~/types/batch'
import type { Detection, HarvestSuggestion, RipenessRatio } from '~/types/infer'
import type { RipenessLabel } from '~/types/trace'

export interface SessionAggregateState {
  seenTrackIds: Set<number>
  counts: Record<RipenessLabel, number>
  totalUnique: number
}

export interface SessionAggregateSummary extends BatchSummaryInput {
  unripe_count: number
  unripe_ratio: number
  ripeness_ratio: RipenessRatio
  harvest_suggestion: HarvestSuggestion
}

export function createSessionAggregateState(): SessionAggregateState {
  return {
    seenTrackIds: new Set<number>(),
    counts: {
      green: 0,
      half: 0,
      red: 0,
      young: 0
    },
    totalUnique: 0
  }
}

export function applyDetectionsToSession(
  state: SessionAggregateState,
  detections: Detection[]
): SessionAggregateState {
  const next: SessionAggregateState = {
    seenTrackIds: new Set<number>(state.seenTrackIds),
    counts: {
      ...state.counts
    },
    totalUnique: state.totalUnique
  }

  for (const detection of detections) {
    if (detection.track_id !== null) {
      if (next.seenTrackIds.has(detection.track_id)) {
        continue
      }
      next.seenTrackIds.add(detection.track_id)
    }

    next.totalUnique += 1
    if (detection.ripeness in next.counts) {
      next.counts[detection.ripeness] += 1
    }
  }

  return next
}

export function buildSessionAggregateSummary(state: SessionAggregateState): SessionAggregateSummary {
  const total = state.totalUnique
  const green = state.counts.green
  const half = state.counts.half
  const red = state.counts.red
  const young = state.counts.young
  const unripeCount = green + young
  const unripeRatio = total > 0 ? unripeCount / total : 0

  const ripenessRatio: RipenessRatio = total > 0
    ? {
        green: green / total,
        half: half / total,
        red: red / total,
        young: young / total
      }
    : {
        green: 0,
        half: 0,
        red: 0,
        young: 0
      }

  return {
    total,
    green,
    half,
    red,
    young,
    unripe_count: unripeCount,
    unripe_ratio: unripeRatio,
    ripeness_ratio: ripenessRatio,
    harvest_suggestion: computeHarvestSuggestion(ripenessRatio)
  }
}

export function computeHarvestSuggestion(ratio: RipenessRatio): HarvestSuggestion {
  if (ratio.red >= 0.7 && ratio.young < 0.15) {
    return 'ready'
  }
  if ((ratio.red + ratio.half) >= 0.4) {
    return 'partially_ready'
  }
  return 'not_ready'
}

export function toBatchSummaryInput(summary: SessionAggregateSummary): BatchSummaryInput {
  return {
    total: summary.total,
    green: summary.green,
    half: summary.half,
    red: summary.red,
    young: summary.young
  }
}
