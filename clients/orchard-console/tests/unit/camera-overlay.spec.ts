import { describe, expect, it } from 'vitest'
import {
  formatDetectionConfidence,
  getOverlayInfoCardPosition,
  mapBboxToCoverViewport,
  mapDetectionsToOverlayBoxes,
  reconcileOverlayBoxIdentities
} from '../../app/utils/camera-overlay'
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

  it('maps hover metadata for each overlay box', () => {
    const overlays = mapDetectionsToOverlayBoxes([baseDetection], baseViewport)

    expect(overlays).toEqual([{
      renderKey: 'track:7',
      hoverId: 'track:7',
      left: 100,
      top: 80,
      width: 200,
      height: 180,
      color: '#D64545',
      label: '红果',
      confidence: 0.92,
      ripeness: 'red',
      identitySource: 'tracked',
      normalizedBbox: [0.078125, 0.1111111111111111, 0.234375, 0.3611111111111111]
    }])
  })

  it('keeps tracked overlay identity stable when bbox changes', () => {
    const overlaysA = mapDetectionsToOverlayBoxes([baseDetection], baseViewport)
    const overlaysB = mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [108, 82, 308, 262]
    }], baseViewport)

    expect(overlaysA[0]?.renderKey).toBe('track:7')
    expect(overlaysA[0]?.hoverId).toBe('track:7')
    expect(overlaysB[0]?.renderKey).toBe('track:7')
    expect(overlaysB[0]?.hoverId).toBe('track:7')
  })

  it('marks untracked detections as pending before reconciliation', () => {
    const overlays = mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null
    }], baseViewport)

    expect(overlays).toEqual([{
      renderKey: 'untracked-pending:0:red:100,80,300,260',
      hoverId: null,
      left: 100,
      top: 80,
      width: 200,
      height: 180,
      color: '#D64545',
      label: '红果',
      confidence: 0.92,
      ripeness: 'red',
      identitySource: 'untracked',
      normalizedBbox: [0.078125, 0.1111111111111111, 0.234375, 0.3611111111111111]
    }])
  })

  it('treats an omitted track_id as untracked', () => {
    const overlays = mapDetectionsToOverlayBoxes([{
      bbox: [100, 80, 300, 260],
      class_name: 'lychee',
      ripeness: 'red',
      confidence: 0.92
    } as Detection], baseViewport)

    expect(overlays).toEqual([{
      renderKey: 'untracked-pending:0:red:100,80,300,260',
      hoverId: null,
      left: 100,
      top: 80,
      width: 200,
      height: 180,
      color: '#D64545',
      label: '红果',
      confidence: 0.92,
      ripeness: 'red',
      identitySource: 'untracked',
      normalizedBbox: [0.078125, 0.1111111111111111, 0.234375, 0.3611111111111111]
    }])
  })

  it('keeps pending render keys distinct for multiple untracked detections in one frame', () => {
    const overlays = mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: undefined as number | null
    }, {
      ...baseDetection,
      bbox: [320, 120, 420, 260],
      track_id: undefined as number | null
    }], baseViewport)

    expect(overlays.map((overlay) => overlay.renderKey)).toEqual([
      'untracked-pending:0:red:100,80,300,260',
      'untracked-pending:1:red:320,120,420,260'
    ])
    expect(overlays.map((overlay) => overlay.hoverId)).toEqual([null, null])
  })

  it('reuses untracked identity across nearby frames when IoU meets the threshold', () => {
    const previous = reconcileOverlayBoxIdentities([], mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null
    }], baseViewport))
    const next = reconcileOverlayBoxIdentities(previous.boxes, mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [108, 86, 308, 266],
      track_id: null
    }], baseViewport), previous.nextUntrackedId)

    expect(previous.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.nextUntrackedId).toBe(1)
  })

  it('keeps untracked identity stable across ripeness label jitter and updates the displayed label', () => {
    const previous = reconcileOverlayBoxIdentities([], mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null,
      ripeness: 'red'
    }], baseViewport))
    const next = reconcileOverlayBoxIdentities(previous.boxes, mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null,
      ripeness: 'half'
    }], baseViewport), previous.nextUntrackedId)

    expect(previous.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.boxes[0]?.ripeness).toBe('half')
    expect(next.boxes[0]?.label).toBe('半熟')
    expect(next.boxes[0]?.color).toBe('#F5A623')
  })

  it('assigns a new identity when an untracked detection can no longer be matched', () => {
    const previous = reconcileOverlayBoxIdentities([], mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null
    }], baseViewport))
    const next = reconcileOverlayBoxIdentities(previous.boxes, mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [420, 240, 620, 420],
      track_id: null
    }], baseViewport), previous.nextUntrackedId)

    expect(previous.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.boxes[0]?.renderKey).toBe('untracked:1')
    expect(next.nextUntrackedId).toBe(2)
  })

  it('keeps untracked identity stable when the viewport changes but the raw bbox does not', () => {
    const previous = reconcileOverlayBoxIdentities([], mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [540, 180, 740, 380],
      track_id: null
    }], {
      videoWidth: 1280,
      videoHeight: 720,
      containerWidth: 1280,
      containerHeight: 720
    }))
    const next = reconcileOverlayBoxIdentities(previous.boxes, mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [540, 180, 740, 380],
      track_id: null
    }], {
      videoWidth: 1280,
      videoHeight: 720,
      containerWidth: 640,
      containerHeight: 720
    }), previous.nextUntrackedId)

    expect(previous.boxes[0]?.renderKey).toBe('untracked:0')
    expect(next.boxes[0]?.renderKey).toBe('untracked:0')
    expect(previous.boxes[0]?.left).not.toBe(next.boxes[0]?.left)
    expect(previous.boxes[0]?.normalizedBbox).toEqual(next.boxes[0]?.normalizedBbox)
  })

  it('matches multiple untracked detections one-to-one by overlap', () => {
    const previous = reconcileOverlayBoxIdentities([], mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      track_id: null
    }, {
      ...baseDetection,
      bbox: [340, 120, 500, 280],
      ripeness: 'half',
      track_id: null
    }], baseViewport))
    const next = reconcileOverlayBoxIdentities(previous.boxes, mapDetectionsToOverlayBoxes([{
      ...baseDetection,
      bbox: [112, 90, 312, 270],
      track_id: null
    }, {
      ...baseDetection,
      bbox: [348, 128, 508, 288],
      ripeness: 'half',
      track_id: null
    }], baseViewport), previous.nextUntrackedId)

    expect(previous.boxes.map((box) => box.renderKey)).toEqual(['untracked:0', 'untracked:1'])
    expect(next.boxes.map((box) => box.renderKey)).toEqual(['untracked:0', 'untracked:1'])
  })

  it('formats confidence as a percentage with one decimal place', () => {
    expect(formatDetectionConfidence(0.923)).toBe('92.3%')
  })

  it('places the hover card to the right when there is space', () => {
    const position = getOverlayInfoCardPosition({
      left: 120,
      top: 48,
      width: 56,
      height: 60
    }, {
      containerWidth: 400,
      containerHeight: 240
    })

    expect(position).toEqual({
      left: 184,
      top: 48
    })
  })

  it('keeps the hover card inside the viewport near the right edge', () => {
    const position = getOverlayInfoCardPosition({
      left: 320,
      top: 24,
      width: 50,
      height: 60
    }, {
      containerWidth: 400,
      containerHeight: 200
    })

    expect(position).toEqual({
      left: 288,
      top: 92
    })
  })
})
