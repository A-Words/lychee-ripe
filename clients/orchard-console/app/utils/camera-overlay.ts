import { RIPENESS_COLOR_MAP } from '~/constants/ripeness'
import type { Detection } from '~/types/infer'

const MIN_OVERLAY_SIZE_PX = 2

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

export interface DetectionOverlayBox extends OverlayRect {
  key: string
  color: string
}

export function mapDetectionsToOverlayBoxes(
  detections: Detection[],
  viewport: VideoCoverViewport
): DetectionOverlayBox[] {
  return detections.flatMap((detection, index) => {
    if (detection.class_name !== 'lychee') {
      return []
    }

    const rect = mapBboxToCoverViewport(detection.bbox, viewport)
    if (!rect) {
      return []
    }

    return [{
      key: buildOverlayKey(detection, index),
      ...rect,
      color: RIPENESS_COLOR_MAP[detection.ripeness]
    }]
  })
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

function buildOverlayKey(detection: Detection, index: number): string {
  const trackPart = detection.track_id ?? `frame-${index}`
  return `${trackPart}:${detection.ripeness}:${detection.bbox.join(',')}`
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
