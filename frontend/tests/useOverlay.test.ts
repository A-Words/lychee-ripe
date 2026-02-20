import { describe, expect, it } from 'vitest'
import { detectionLabel, drawOverlay, scaleBbox } from '../app/composables/useDetectionOverlay'
import type { Detection } from '../app/types/infer'

describe('useOverlay', () => {
  it('scales bbox into canvas coordinates', () => {
    const box = scaleBbox([10, 20, 110, 120], 200, 200, 100, 50)
    expect(box).toEqual({ x: 5, y: 5, width: 50, height: 25 })
  })

  it('clamps out-of-range bbox', () => {
    const box = scaleBbox([-10, -20, 210, 220], 200, 200, 100, 50)
    expect(box.x).toBe(0)
    expect(box.y).toBe(0)
    expect(box.width).toBeCloseTo(99.5)
    expect(box.height).toBeCloseTo(49.75)
  })

  it('draws rectangle and label', () => {
    const calls: string[] = []
    const ctx = {
      font: '',
      textBaseline: 'top',
      lineWidth: 1,
      strokeStyle: '#000',
      fillStyle: '#000',
      clearRect: () => calls.push('clearRect'),
      strokeRect: () => calls.push('strokeRect'),
      fillRect: () => calls.push('fillRect'),
      fillText: () => calls.push('fillText'),
      measureText: () => ({ width: 40 }),
    } as unknown as CanvasRenderingContext2D

    const detections: Detection[] = [
      {
        bbox: [10, 10, 40, 50],
        class_name: 'lychee',
        ripeness: 'red',
        confidence: 0.95,
        track_id: null,
      },
    ]

    drawOverlay({
      context: ctx,
      detections,
      sourceWidth: 100,
      sourceHeight: 100,
      canvasWidth: 100,
      canvasHeight: 100,
    })

    expect(calls).toContain('clearRect')
    expect(calls).toContain('strokeRect')
    expect(calls).toContain('fillRect')
    expect(calls).toContain('fillText')
  })

  it('formats detection label', () => {
    expect(detectionLabel('half', 0.5)).toBe('Half 50.00%')
  })
})
