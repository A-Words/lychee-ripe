import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useCamera } from '../app/composables/useCamera'

interface DeviceState {
  kind: MediaDeviceKind
  deviceId: string
  label: string
}

function nextTick() {
  return Promise.resolve()
}

function createMediaStream(deviceId: string) {
  const track = {
    stop: vi.fn(),
    getSettings: () => ({ deviceId }),
  }

  return {
    getTracks: () => [track],
    getVideoTracks: () => [track],
  } as unknown as MediaStream
}

function createMediaDevicesMock(initialDevices: DeviceState[]) {
  let devices = [...initialDevices]
  const listeners = new Map<string, EventListener[]>()

  const enumerateDevices = vi.fn(async () => devices)
  const getUserMedia = vi.fn(async (constraints: MediaStreamConstraints) => {
    if (!constraints.video || constraints.video === true) {
      const fallback = devices[0]?.deviceId || 'default-camera'
      return createMediaStream(fallback)
    }

    const requestedDeviceId = (constraints.video as MediaTrackConstraints).deviceId as
      | { exact?: string }
      | undefined
    const exactId = requestedDeviceId?.exact

    if (exactId && !devices.some(device => device.deviceId === exactId)) {
      throw new Error('device not found')
    }

    return createMediaStream(exactId || devices[0]?.deviceId || 'default-camera')
  })

  const addEventListener = vi.fn((type: string, listener: EventListener) => {
    const list = listeners.get(type) ?? []
    list.push(listener)
    listeners.set(type, list)
  })

  const removeEventListener = vi.fn((type: string, listener: EventListener) => {
    const list = listeners.get(type) ?? []
    listeners.set(type, list.filter(item => item !== listener))
  })

  function setDevices(nextDevices: DeviceState[]) {
    devices = [...nextDevices]
  }

  function emit(type: string) {
    const event = new Event(type)
    for (const listener of listeners.get(type) ?? []) {
      listener(event)
    }
  }

  return {
    mediaDevices: {
      enumerateDevices,
      getUserMedia,
      addEventListener,
      removeEventListener,
    },
    setDevices,
    emit,
    enumerateDevices,
    getUserMedia,
  }
}

describe('useCamera', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('enumerates video devices and uses fallback labels', async () => {
    const mock = createMediaDevicesMock([
      { kind: 'videoinput', deviceId: 'cam-1', label: '' },
      { kind: 'videoinput', deviceId: 'cam-2', label: 'USB Camera' },
      { kind: 'audioinput', deviceId: 'mic-1', label: 'Microphone' },
    ])
    vi.stubGlobal('navigator', { mediaDevices: mock.mediaDevices })

    const camera = useCamera({ width: 640, height: 360 })
    await camera.refreshDevices()

    expect(camera.devices.value).toEqual([
      { id: 'cam-1', label: 'Camera 1' },
      { id: 'cam-2', label: 'USB Camera' },
    ])
    expect(camera.selectedDeviceId.value).toBe('cam-1')
  })

  it('uses exact deviceId constraint when selected', async () => {
    const mock = createMediaDevicesMock([
      { kind: 'videoinput', deviceId: 'cam-1', label: 'Front' },
      { kind: 'videoinput', deviceId: 'cam-2', label: 'Back' },
    ])
    vi.stubGlobal('navigator', { mediaDevices: mock.mediaDevices })

    const camera = useCamera()
    await camera.refreshDevices()
    camera.selectDevice('cam-2')

    const video = {
      srcObject: null,
      muted: false,
      playsInline: false,
      play: vi.fn(async () => {}),
    } as unknown as HTMLVideoElement

    await camera.start(video, camera.selectedDeviceId.value)

    const constraint = mock.getUserMedia.mock.calls[0][0] as MediaStreamConstraints
    expect((constraint.video as MediaTrackConstraints).deviceId).toEqual({ exact: 'cam-2' })
    expect(camera.activeDeviceId.value).toBe('cam-2')
  })

  it('falls back to first available camera when selected device is stale', async () => {
    const mock = createMediaDevicesMock([
      { kind: 'videoinput', deviceId: 'cam-1', label: 'Front' },
      { kind: 'videoinput', deviceId: 'cam-2', label: 'Back' },
    ])
    vi.stubGlobal('navigator', { mediaDevices: mock.mediaDevices })

    const camera = useCamera()
    await camera.refreshDevices()
    camera.selectDevice('cam-missing')
    await camera.refreshDevices()

    expect(camera.selectedDeviceId.value).toBe('cam-1')
  })

  it('refreshes device list on devicechange event', async () => {
    const mock = createMediaDevicesMock([
      { kind: 'videoinput', deviceId: 'cam-1', label: 'Front' },
    ])
    vi.stubGlobal('navigator', { mediaDevices: mock.mediaDevices })

    const camera = useCamera()
    await camera.refreshDevices()
    expect(camera.devices.value).toHaveLength(1)

    mock.setDevices([
      { kind: 'videoinput', deviceId: 'cam-1', label: 'Front' },
      { kind: 'videoinput', deviceId: 'cam-2', label: 'Back' },
    ])
    mock.emit('devicechange')
    await nextTick()

    expect(camera.devices.value).toHaveLength(2)
  })

  it('marks active device as missing and picks fallback after unplug', async () => {
    const mock = createMediaDevicesMock([
      { kind: 'videoinput', deviceId: 'cam-1', label: 'Front' },
      { kind: 'videoinput', deviceId: 'cam-2', label: 'Back' },
    ])
    vi.stubGlobal('navigator', { mediaDevices: mock.mediaDevices })

    const camera = useCamera()
    await camera.refreshDevices()
    camera.selectDevice('cam-1')

    const video = {
      srcObject: null,
      muted: false,
      playsInline: false,
      play: vi.fn(async () => {}),
    } as unknown as HTMLVideoElement

    await camera.start(video, 'cam-1')
    expect(camera.activeDeviceId.value).toBe('cam-1')

    mock.setDevices([
      { kind: 'videoinput', deviceId: 'cam-2', label: 'Back' },
    ])
    mock.emit('devicechange')
    await nextTick()

    expect(camera.activeDeviceMissing.value).toBe(true)
    expect(camera.selectedDeviceId.value).toBe('cam-2')
  })
})
