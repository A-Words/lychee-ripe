import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { readFileSync } from 'node:fs'
import { useCamera } from '../../app/composables/useCamera'

interface StorageLike {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
  removeItem: (key: string) => void
  clear: () => void
}

function createMemoryStorage(): StorageLike {
  const store = new Map<string, string>()
  return {
    getItem(key: string) {
      return store.has(key) ? store.get(key)! : null
    },
    setItem(key: string, value: string) {
      store.set(key, value)
    },
    removeItem(key: string) {
      store.delete(key)
    },
    clear() {
      store.clear()
    }
  }
}

function createCamera() {
  return useCamera(ref<HTMLVideoElement | null>(null))
}

function setMediaDevicesMock(mock: Partial<MediaDevices>) {
  vi.stubGlobal('navigator', { mediaDevices: mock })
}

function buildStream(deviceId: string): MediaStream {
  const track = {
    stop: vi.fn(),
    getSettings: () => ({ deviceId })
  } as unknown as MediaStreamTrack

  return {
    getTracks: () => [track],
    getVideoTracks: () => [track]
  } as unknown as MediaStream
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('camera flow', () => {
  it('keeps start button clickable even when no devices are listed', () => {
    const source = readFileSync(new URL('../../app/components/batch/CameraStage.vue', import.meta.url), 'utf8')
    expect(source).toContain(':disabled="cameraLoading"')
    expect(source).not.toContain(':disabled="!hasDevices"')
  })

  it('renders a dedicated overlay layer for realtime detection boxes', () => {
    const source = readFileSync(new URL('../../app/components/batch/CameraStage.vue', import.meta.url), 'utf8')
    expect(source).toContain('data-testid="camera-overlay"')
    expect(source).toContain('data-testid="camera-overlay-card"')
    expect(source).toContain('v-if="showOverlay"')
    expect(source).toContain(':key="box.renderKey"')
    expect(source).toContain('@mouseenter="handleBoxEnter(box)"')
    expect(source).toContain('@focus="handleBoxFocus(box)"')
    expect(source).toContain('@blur="handleBoxBlur(box)"')
    expect(source).toContain('@click="handleBoxClick(box)"')
    expect(source).toContain('<button')
    expect(source).toContain('type="button"')
    expect(source).toContain('focus-visible:outline-2')
    expect(source).toContain(':aria-label=')
    expect(source).not.toContain('{{ box.label }}')
    expect(source).not.toContain('const isSameTrackedBox = box.hoverId && hoveredDetectionId.value === box.hoverId')
    expect(source).not.toContain('@keydown="handleBoxKeydown($event, box)"')
    expect(source).not.toContain('const frameId = `${props.currentFrame.frame_index}:${props.currentFrame.timestamp_ms}`')
  })

  it('passes the latest frame into the camera stage from the batch page', () => {
    const source = readFileSync(new URL('../../app/pages/batch/create.vue', import.meta.url), 'utf8')
    expect(source).toContain(':current-frame="lastFrame"')
  })

  it('starts camera and refreshes selected device after first permission grant', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.stubGlobal('localStorage', createMemoryStorage())
    const enumerateDevices = vi
      .fn<MediaDevices['enumerateDevices']>()
      .mockResolvedValue([{ kind: 'videoinput', deviceId: 'cam-1', label: 'USB Cam' } as MediaDeviceInfo])
    const getUserMedia = vi.fn<MediaDevices['getUserMedia']>().mockResolvedValue(buildStream('cam-1'))

    setMediaDevicesMock({
      enumerateDevices,
      getUserMedia,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    })

    const camera = createCamera()
    await camera.startCamera('')

    expect(camera.cameraError.value).toBe('')
    expect(camera.hasPermission.value).toBe(true)
    expect(camera.selectedDeviceId.value).toBe('cam-1')
    expect(camera.options.value.map((item) => item.value)).toEqual(['cam-1'])
  })

  it('returns a readable error when permission is denied', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.stubGlobal('localStorage', createMemoryStorage())
    const enumerateDevices = vi.fn<MediaDevices['enumerateDevices']>().mockResolvedValue([])
    const getUserMedia = vi
      .fn<MediaDevices['getUserMedia']>()
      .mockRejectedValue(new DOMException('permission denied', 'NotAllowedError'))

    setMediaDevicesMock({
      enumerateDevices,
      getUserMedia,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    })

    const camera = createCamera()
    await camera.startCamera('cam-x')
    expect(camera.cameraError.value).toBe('摄像头权限被拒绝，请在浏览器或系统设置中允许访问。')
  })

  it('sets explicit error when enumerate devices fails', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.stubGlobal('localStorage', createMemoryStorage())
    const enumerateDevices = vi
      .fn<MediaDevices['enumerateDevices']>()
      .mockRejectedValue(new Error('enumerate failed'))

    setMediaDevicesMock({
      enumerateDevices,
      getUserMedia: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    })

    const camera = createCamera()
    await camera.refreshDevices()
    expect(camera.cameraError.value).toBe('摄像头设备枚举失败，请检查浏览器权限后重试。')
    expect(camera.options.value).toEqual([])
  })

  it('hides devices without deviceId before permission is granted', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.stubGlobal('localStorage', createMemoryStorage())
    const enumerateDevices = vi
      .fn<MediaDevices['enumerateDevices']>()
      .mockResolvedValue([{ kind: 'videoinput', deviceId: '', label: '' } as MediaDeviceInfo])

    setMediaDevicesMock({
      enumerateDevices,
      getUserMedia: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    })

    const camera = createCamera()
    await camera.refreshDevices()

    expect(camera.options.value).toEqual([])
    expect(camera.hasDevices.value).toBe(false)
  })
})
