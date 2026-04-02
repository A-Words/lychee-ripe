import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { Detection, FrameResult } from '../../app/types/infer'
import CameraStage from '../../app/components/batch/CameraStage.vue'

const baseDetection: Detection = {
  bbox: [540, 180, 740, 380],
  class_name: 'lychee',
  ripeness: 'red',
  confidence: 0.92,
  track_id: 7
}

const stubs = {
  UCard: defineComponent({
    template: '<div><slot name="header" /><slot /></div>'
  }),
  UAlert: defineComponent({
    template: '<div><slot /><slot name="description" /></div>'
  }),
  UBadge: defineComponent({
    template: '<div><slot /></div>'
  }),
  UButton: defineComponent({
    props: {
      label: {
        type: String,
        default: ''
      }
    },
    emits: ['click'],
    template: '<button type="button" @click="$emit(\'click\', $event)">{{ label }}<slot /></button>'
  }),
  USelect: defineComponent({
    template: '<select />'
  })
}

const resizeObservers: ResizeObserverMock[] = []

class ResizeObserverMock {
  callback: ResizeObserverCallback

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
    resizeObservers.push(this)
  }

  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
}

beforeEach(() => {
  resizeObservers.length = 0
  vi.stubGlobal('ResizeObserver', ResizeObserverMock)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('camera stage overlay interactions', () => {
  it('shows and hides the info card on hover', async () => {
    const wrapper = await mountCameraStage()
    const box = getOverlayButton(wrapper)

    expect(findOverlayCard(wrapper).exists()).toBe(false)

    await box.trigger('mouseenter')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(true)
    expect(findOverlayCard(wrapper).text()).toContain('成熟度：红果')

    await box.trigger('mouseleave')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(false)
  })

  it('shows and hides the info card on focus', async () => {
    const wrapper = await mountCameraStage()
    const box = getOverlayButton(wrapper)

    await box.trigger('focus')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(true)

    await box.trigger('blur')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(false)
  })

  it('dismisses the active card when clicking the same box again while still hovered', async () => {
    const wrapper = await mountCameraStage()
    const box = getOverlayButton(wrapper)

    await box.trigger('mouseenter')
    await box.trigger('click')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(true)

    await box.trigger('click')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(false)
  })

  it('keeps tracked selection active across streamed frames', async () => {
    const wrapper = await mountCameraStage({
      currentFrame: buildFrame([baseDetection], 1)
    })

    await getOverlayButton(wrapper).trigger('click')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(true)

    await wrapper.setProps({
      currentFrame: buildFrame([{
        ...baseDetection,
        bbox: [548, 188, 748, 388]
      }], 2)
    })
    await flushUi()

    expect(findOverlayCard(wrapper).exists()).toBe(true)
    expect(findOverlayCard(wrapper).text()).toContain('成熟度：红果')
  })

  it('keeps untracked selection active across label jitter and viewport resize', async () => {
    const wrapper = await mountCameraStage({
      currentFrame: buildFrame([{
        ...baseDetection,
        track_id: null,
        ripeness: 'red'
      }], 1)
    })

    await getOverlayButton(wrapper).trigger('click')
    await flushUi()
    expect(findOverlayCard(wrapper).text()).toContain('成熟度：红果')

    setViewportSize(getViewportElement(wrapper), 960, 720)
    triggerResize()

    await wrapper.setProps({
      currentFrame: buildFrame([{
        ...baseDetection,
        track_id: null,
        ripeness: 'half'
      }], 2)
    })
    await flushUi()

    expect(findOverlayCard(wrapper).exists()).toBe(true)
    expect(findOverlayCard(wrapper).text()).toContain('成熟度：半熟')
    expect(findOverlayCard(wrapper).text()).toContain('置信度：92.0%')
  })

  it('clears the untracked card when the detection can no longer be matched', async () => {
    const wrapper = await mountCameraStage({
      currentFrame: buildFrame([{
        ...baseDetection,
        track_id: null
      }], 1)
    })

    await getOverlayButton(wrapper).trigger('click')
    await flushUi()
    expect(findOverlayCard(wrapper).exists()).toBe(true)

    await wrapper.setProps({
      currentFrame: buildFrame([{
        ...baseDetection,
        bbox: [80, 80, 220, 220],
        track_id: null
      }], 2)
    })
    await flushUi()

    expect(findOverlayCard(wrapper).exists()).toBe(false)
  })
})

async function mountCameraStage(options: {
  currentFrame?: FrameResult | null
} = {}) {
  const wrapper = await mountSuspended(CameraStage, {
    props: {
      devices: [],
      selectedDeviceId: '',
      currentFrame: options.currentFrame ?? buildFrame([baseDetection], 1),
      isRecognizing: true,
      cameraLoading: false,
      cameraError: '',
      streamError: ''
    },
    global: {
      stubs
    }
  })

  const viewport = getViewportElement(wrapper)
  const video = wrapper.get('video').element as HTMLVideoElement

  setViewportSize(viewport, 1280, 720)
  setVideoSize(video, 1280, 720)
  video.dispatchEvent(new Event('loadedmetadata'))
  triggerResize()
  await flushUi()

  return wrapper
}

function buildFrame(detections: Detection[], frameIndex: number): FrameResult {
  return {
    frame_index: frameIndex,
    timestamp_ms: frameIndex * 1000,
    detections,
    frame_summary: {
      total: detections.length,
      green: detections.filter((item) => item.ripeness === 'green').length,
      half: detections.filter((item) => item.ripeness === 'half').length,
      red: detections.filter((item) => item.ripeness === 'red').length,
      young: detections.filter((item) => item.ripeness === 'young').length
    }
  }
}

function getViewportElement(wrapper: Awaited<ReturnType<typeof mountSuspended>>) {
  return wrapper.get('[data-testid="camera-stage-viewport"]').element as HTMLElement
}

function getOverlayButton(wrapper: Awaited<ReturnType<typeof mountSuspended>>) {
  return wrapper.get('[data-testid="camera-overlay"] button')
}

function findOverlayCard(wrapper: Awaited<ReturnType<typeof mountSuspended>>) {
  return wrapper.find('[data-testid="camera-overlay-card"]')
}

function setViewportSize(element: HTMLElement, width: number, height: number) {
  Object.defineProperty(element, 'clientWidth', {
    configurable: true,
    get: () => width
  })
  Object.defineProperty(element, 'clientHeight', {
    configurable: true,
    get: () => height
  })
}

function setVideoSize(element: HTMLVideoElement, width: number, height: number) {
  Object.defineProperty(element, 'videoWidth', {
    configurable: true,
    get: () => width
  })
  Object.defineProperty(element, 'videoHeight', {
    configurable: true,
    get: () => height
  })
}

function triggerResize() {
  for (const observer of resizeObservers) {
    observer.callback([], observer as unknown as ResizeObserver)
  }
}

async function flushUi() {
  await nextTick()
  await nextTick()
}
