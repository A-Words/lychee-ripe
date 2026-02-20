import type { Detection, RipenessLabel } from '../types/infer'
import { RIPENESS_COLOR_MAP, RIPENESS_LABEL_MAP } from '../constants/ripeness'

interface BoxGeometry {
  x: number
  y: number
  width: number
  height: number
}

interface DrawOptions {
  context: CanvasRenderingContext2D
  detections: Detection[]
  sourceWidth: number
  sourceHeight: number
  canvasWidth: number
  canvasHeight: number
}

export function scaleBbox(
  bbox: [number, number, number, number],
  sourceWidth: number,
  sourceHeight: number,
  canvasWidth: number,
  canvasHeight: number,
): BoxGeometry {
  const [rawX1, rawY1, rawX2, rawY2] = bbox

  const clampX = (value: number) => Math.max(0, Math.min(sourceWidth - 1, value))
  const clampY = (value: number) => Math.max(0, Math.min(sourceHeight - 1, value))

  const x1 = clampX(rawX1)
  const y1 = clampY(rawY1)
  const x2 = clampX(rawX2)
  const y2 = clampY(rawY2)

  const minX = Math.min(x1, x2)
  const minY = Math.min(y1, y2)
  const width = Math.abs(x2 - x1)
  const height = Math.abs(y2 - y1)

  return {
    x: (minX / sourceWidth) * canvasWidth,
    y: (minY / sourceHeight) * canvasHeight,
    width: (width / sourceWidth) * canvasWidth,
    height: (height / sourceHeight) * canvasHeight,
  }
}

export function detectionLabel(ripeness: RipenessLabel, confidence: number) {
  return `${RIPENESS_LABEL_MAP[ripeness]} ${(confidence * 100).toFixed(2)}%`
}

export function drawOverlay({
  context,
  detections,
  sourceWidth,
  sourceHeight,
  canvasWidth,
  canvasHeight,
}: DrawOptions) {
  context.clearRect(0, 0, canvasWidth, canvasHeight)

  context.font = '12px ui-sans-serif, sans-serif'
  context.textBaseline = 'top'
  context.lineWidth = 2

  for (const det of detections) {
    const box = scaleBbox(det.bbox, sourceWidth, sourceHeight, canvasWidth, canvasHeight)
    const color = RIPENESS_COLOR_MAP[det.ripeness]
    const label = detectionLabel(det.ripeness, det.confidence)

    context.strokeStyle = color
    context.strokeRect(box.x, box.y, box.width, box.height)

    const textWidth = context.measureText(label).width
    const tagX = box.x
    const tagY = Math.max(0, box.y - 18)
    context.fillStyle = color
    context.fillRect(tagX, tagY, textWidth + 10, 18)
    context.fillStyle = '#FFFFFF'
    context.fillText(label, tagX + 5, tagY + 3)
  }
}

export function useDetectionOverlay() {
  function clear(context: CanvasRenderingContext2D, canvasWidth: number, canvasHeight: number) {
    context.clearRect(0, 0, canvasWidth, canvasHeight)
  }

  return {
    clear,
    draw: drawOverlay,
  }
}
