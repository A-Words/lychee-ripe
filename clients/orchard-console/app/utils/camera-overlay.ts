import { RIPENESS_COLOR_MAP, RIPENESS_LABEL_MAP } from '~/constants/ripeness'
import type { Detection } from '~/types/infer'
import type { RipenessLabel } from '~/types/trace'

const MIN_OVERLAY_SIZE_PX = 2
const INFO_CARD_WIDTH_PX = 112
const INFO_CARD_HEIGHT_PX = 56
const INFO_CARD_GAP_PX = 8
const MIN_UNTRACKED_MATCH_IOU = 0.5

export type DetectionOverlayIdentitySource = 'tracked' | 'untracked'

export interface VideoCoverViewport {
  videoWidth: number
  videoHeight: number
  containerWidth: number
  containerHeight: number
}

export interface OverlayRect {
  left: number
  top: number
  width: number
  height: number
}

export type NormalizedBbox = [number, number, number, number]

export interface DetectionOverlayBox extends OverlayRect {
  renderKey: string
  hoverId: string | null
  color: string
  label: string
  confidence: number
  ripeness: RipenessLabel
  identitySource: DetectionOverlayIdentitySource
  normalizedBbox: NormalizedBbox
}

export interface OverlayInfoCardPosition {
  left: number
  top: number
}

export interface OverlayIdentityOptions {
  nextUntrackedId?: number
}

export interface ReconciledOverlayIdentityResult {
  boxes: DetectionOverlayBox[]
  nextUntrackedId: number
}

export function mapDetectionsToOverlayBoxes(
  detections: Detection[],
  viewport: VideoCoverViewport,
  _identity: OverlayIdentityOptions = {}
): DetectionOverlayBox[] {
  return detections.flatMap((detection, index) => {
    if (detection.class_name !== 'lychee') {
      return []
    }

    const rect = mapBboxToCoverViewport(detection.bbox, viewport)
    if (!rect) {
      return []
    }

    const normalizedBbox = normalizeBbox(detection.bbox, viewport.videoWidth, viewport.videoHeight)

    return [{
      renderKey: buildOverlayRenderKey(detection, index),
      hoverId: buildOverlayHoverId(detection),
      ...rect,
      color: RIPENESS_COLOR_MAP[detection.ripeness],
      label: RIPENESS_LABEL_MAP[detection.ripeness],
      confidence: detection.confidence,
      ripeness: detection.ripeness,
      identitySource: detection.track_id != null ? 'tracked' : 'untracked',
      normalizedBbox
    }]
  })
}

export function reconcileOverlayBoxIdentities(
  previousBoxes: DetectionOverlayBox[],
  currentBoxes: DetectionOverlayBox[],
  nextUntrackedId = 0
): ReconciledOverlayIdentityResult {
  const boxes = currentBoxes.map((box) => ({ ...box }))
  const previousUntracked = previousBoxes
    .map((box, index) => ({ box, index }))
    .filter((entry) => entry.box.identitySource === 'untracked')
  const currentUntracked = boxes
    .map((box, index) => ({ box, index }))
    .filter((entry) => entry.box.identitySource === 'untracked')

  const candidates = previousUntracked.flatMap((previousEntry) => {
    return currentUntracked.flatMap((currentEntry) => {
      const iou = calculateBboxIou(previousEntry.box.normalizedBbox, currentEntry.box.normalizedBbox)
      if (iou < MIN_UNTRACKED_MATCH_IOU) {
        return []
      }

      return [{
        previousEntry,
        currentEntry,
        iou
      }]
    })
  })

  candidates.sort((left, right) => right.iou - left.iou)

  const matchedPreviousIndexes = new Set<number>()
  const matchedCurrentIndexes = new Set<number>()

  for (const candidate of candidates) {
    if (
      matchedPreviousIndexes.has(candidate.previousEntry.index)
      || matchedCurrentIndexes.has(candidate.currentEntry.index)
    ) {
      continue
    }

    boxes[candidate.currentEntry.index] = {
      ...candidate.currentEntry.box,
      renderKey: candidate.previousEntry.box.renderKey,
      hoverId: candidate.previousEntry.box.hoverId
    }

    matchedPreviousIndexes.add(candidate.previousEntry.index)
    matchedCurrentIndexes.add(candidate.currentEntry.index)
  }

  for (const currentEntry of currentUntracked) {
    if (matchedCurrentIndexes.has(currentEntry.index)) {
      continue
    }

    boxes[currentEntry.index] = {
      ...currentEntry.box,
      renderKey: `untracked:${nextUntrackedId}`,
      hoverId: null
    }
    nextUntrackedId += 1
  }

  return {
    boxes,
    nextUntrackedId
  }
}

export function formatDetectionConfidence(confidence: number): string {
  const normalized = clamp(confidence, 0, 1)
  return `${(normalized * 100).toFixed(1)}%`
}

export function getOverlayInfoCardPosition(
  box: OverlayRect,
  viewport: Pick<VideoCoverViewport, 'containerWidth' | 'containerHeight'>
): OverlayInfoCardPosition {
  const maxLeft = Math.max(viewport.containerWidth - INFO_CARD_WIDTH_PX, 0)
  const maxTop = Math.max(viewport.containerHeight - INFO_CARD_HEIGHT_PX, 0)

  const preferredRightLeft = box.left + box.width + INFO_CARD_GAP_PX
  if (preferredRightLeft + INFO_CARD_WIDTH_PX <= viewport.containerWidth) {
    return {
      left: preferredRightLeft,
      top: clamp(box.top, 0, maxTop)
    }
  }

  const alignedLeft = clamp(box.left, 0, maxLeft)
  const preferredAboveTop = box.top - INFO_CARD_HEIGHT_PX - INFO_CARD_GAP_PX
  if (preferredAboveTop >= 0) {
    return {
      left: alignedLeft,
      top: preferredAboveTop
    }
  }

  const preferredBelowTop = box.top + box.height + INFO_CARD_GAP_PX
  if (preferredBelowTop + INFO_CARD_HEIGHT_PX <= viewport.containerHeight) {
    return {
      left: alignedLeft,
      top: preferredBelowTop
    }
  }

  return {
    left: alignedLeft,
    top: clamp(box.top, 0, maxTop)
  }
}

export function mapBboxToCoverViewport(
  bbox: Detection['bbox'],
  viewport: VideoCoverViewport
): OverlayRect | null {
  if (
    viewport.videoWidth <= 0
    || viewport.videoHeight <= 0
    || viewport.containerWidth <= 0
    || viewport.containerHeight <= 0
  ) {
    return null
  }

  const [rawX1, rawY1, rawX2, rawY2] = bbox
  const x1 = Math.min(rawX1, rawX2)
  const y1 = Math.min(rawY1, rawY2)
  const x2 = Math.max(rawX1, rawX2)
  const y2 = Math.max(rawY1, rawY2)

  const scale = Math.max(
    viewport.containerWidth / viewport.videoWidth,
    viewport.containerHeight / viewport.videoHeight
  )

  const scaledWidth = viewport.videoWidth * scale
  const scaledHeight = viewport.videoHeight * scale
  const offsetX = (scaledWidth - viewport.containerWidth) / 2
  const offsetY = (scaledHeight - viewport.containerHeight) / 2

  const mappedLeft = x1 * scale - offsetX
  const mappedTop = y1 * scale - offsetY
  const mappedRight = x2 * scale - offsetX
  const mappedBottom = y2 * scale - offsetY

  const clippedLeft = clamp(mappedLeft, 0, viewport.containerWidth)
  const clippedTop = clamp(mappedTop, 0, viewport.containerHeight)
  const clippedRight = clamp(mappedRight, 0, viewport.containerWidth)
  const clippedBottom = clamp(mappedBottom, 0, viewport.containerHeight)

  const width = clippedRight - clippedLeft
  const height = clippedBottom - clippedTop

  if (width < MIN_OVERLAY_SIZE_PX || height < MIN_OVERLAY_SIZE_PX) {
    return null
  }

  return {
    left: clippedLeft,
    top: clippedTop,
    width,
    height
  }
}

function buildOverlayRenderKey(
  detection: Detection,
  index: number
): string {
  if (detection.track_id != null) {
    return `track:${detection.track_id}`
  }

  return `untracked-pending:${index}:${detection.ripeness}:${detection.bbox.join(',')}`
}

function buildOverlayHoverId(detection: Detection): string | null {
  if (detection.track_id == null) {
    return null
  }

  return `track:${detection.track_id}`
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

function normalizeBbox(
  bbox: Detection['bbox'],
  videoWidth: number,
  videoHeight: number
): NormalizedBbox {
  const [rawX1, rawY1, rawX2, rawY2] = bbox
  const x1 = Math.min(rawX1, rawX2)
  const y1 = Math.min(rawY1, rawY2)
  const x2 = Math.max(rawX1, rawX2)
  const y2 = Math.max(rawY1, rawY2)

  return [
    clamp(x1 / videoWidth, 0, 1),
    clamp(y1 / videoHeight, 0, 1),
    clamp(x2 / videoWidth, 0, 1),
    clamp(y2 / videoHeight, 0, 1)
  ]
}

function calculateBboxIou(left: NormalizedBbox, right: NormalizedBbox): number {
  const intersectionLeft = Math.max(left[0], right[0])
  const intersectionTop = Math.max(left[1], right[1])
  const intersectionRight = Math.min(left[2], right[2])
  const intersectionBottom = Math.min(left[3], right[3])

  const intersectionWidth = Math.max(intersectionRight - intersectionLeft, 0)
  const intersectionHeight = Math.max(intersectionBottom - intersectionTop, 0)
  const intersectionArea = intersectionWidth * intersectionHeight

  if (intersectionArea <= 0) {
    return 0
  }

  const leftArea = Math.max(left[2] - left[0], 0) * Math.max(left[3] - left[1], 0)
  const rightArea = Math.max(right[2] - right[0], 0) * Math.max(right[3] - right[1], 0)
  return intersectionArea / (leftArea + rightArea - intersectionArea)
}
