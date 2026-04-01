import { describe, expect, it } from 'vitest'
import { mapBboxToCoverViewport, mapDetectionsToOverlayBoxes } from '../../app/utils/camera-overlay'
import type { Detection } from '../../app/types/infer'

const baseViewport = {
  videoWidth: 1280,
  videoHeight: 720,
  containerWidth: 1280,
  containerHeight: 720
}

const baseDetection: Detection = {
  bbox: [100, 80, 300, 260],
  class_name: 'lychee',
  ripeness: 'red',
  confidence: 0.92,
  track_id: 7
}

describe('camera overlay mapping', () => {
  it('maps bbox directly when viewport matches source aspect ratio', () => {
    const rect = mapBboxToCoverViewport(baseDetection.bbox, baseViewport)

    expect(rect).toEqual({
      left: 100,
      top: 80,
      width: 200,
      height: 180
    })
  })

  it('clips overlay correctly when object-cover crops the left and right edges', () => {
    const rect = mapBboxToCoverViewport([300, 100, 340, 300], {
      videoWidth: 1280,
      videoHeight: 720,
      containerWidth: 640,
      containerHeight: 720
    })

    expect(rect).toEqual({
      left: 0,
      top: 100,
      width: 20,
      height: 200
    })
  })

  it('clips overlay correctly when object-cover crops the top and bottom edges', () => {
    const rect = mapBboxToCoverViewport([100, 0, 300, 120], {
      videoWidth: 1280,
      videoHeight: 720,
      containerWidth: 1280,
      containerHeight: 600
    })

    expect(rect).toEqual({
      left: 100,
      top: 0,
      width: 200,
      height: 60
    })
  })

  it('skips detections that are fully outside the visible cover viewport', () => {
    const overlays = mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [0, 100, 160, 300]
    }], {
      videoWidth: 1280,
      videoHeight: 720,
      containerWidth: 640,
      containerHeight: 720
    })

    expect(overlays).toEqual([])
  })

  it('returns an empty overlay list when there are no detections', () => {
    const overlays = mapDetectionsToOverlayBoxes([], baseViewport)
    expect(overlays).toEqual([])
  })
})
