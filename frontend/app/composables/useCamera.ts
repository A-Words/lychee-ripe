import { getCurrentScope, onScopeDispose, ref } from 'vue'

type CameraStatus = 'idle' | 'requesting' | 'ready' | 'error'

interface CameraOptions {
  width?: number
  height?: number
}

export interface CameraDeviceOption {
  id: string
  label: string
}

export function useCamera(options: CameraOptions = {}) {
  const status = ref<CameraStatus>('idle')
  const error = ref<string | null>(null)
  const stream = ref<MediaStream | null>(null)
  const devices = ref<CameraDeviceOption[]>([])
  const selectedDeviceId = ref<string>('')
  const activeDeviceId = ref<string | null>(null)
  const activeDeviceMissing = ref(false)
  const isEnumerating = ref(false)
  const isSwitching = ref(false)

  const deviceChangeHandler = () => {
    void refreshDevices()
  }

  function hasCameraApi() {
    return typeof navigator !== 'undefined' &&
      Boolean(navigator.mediaDevices?.getUserMedia) &&
      Boolean(navigator.mediaDevices?.enumerateDevices)
  }

  function makeFallbackLabel(index: number) {
    return `Camera ${index + 1}`
  }

  function reconcileSelection(nextDevices: CameraDeviceOption[]) {
    if (!nextDevices.length) {
      selectedDeviceId.value = ''
      activeDeviceMissing.value = activeDeviceId.value !== null
      return
    }

    const selectedExists = nextDevices.some(device => device.id === selectedDeviceId.value)
    if (!selectedExists) {
      const fallbackDevice = nextDevices[0]
      if (fallbackDevice) {
        selectedDeviceId.value = fallbackDevice.id
      }
    }

    if (!activeDeviceId.value) {
      activeDeviceMissing.value = false
      return
    }

    activeDeviceMissing.value = !nextDevices.some(device => device.id === activeDeviceId.value)
  }

  async function refreshDevices() {
    if (!hasCameraApi()) {
      devices.value = []
      selectedDeviceId.value = ''
      activeDeviceMissing.value = activeDeviceId.value !== null
      return devices.value
    }

    isEnumerating.value = true
    try {
      const raw = await navigator.mediaDevices.enumerateDevices()
      const videoInputs = raw
        .filter(device => device.kind === 'videoinput')
        .map((device, index) => ({
          id: device.deviceId,
          label: device.label || makeFallbackLabel(index),
        }))

      devices.value = videoInputs
      reconcileSelection(videoInputs)
      return videoInputs
    } finally {
      isEnumerating.value = false
    }
  }

  function selectDevice(deviceId: string) {
    selectedDeviceId.value = deviceId
  }

  async function requestStream(deviceId?: string) {
    if (!hasCameraApi()) {
      throw new Error('Current environment does not support camera APIs')
    }

    const hasTargetDevice = typeof deviceId === 'string' && deviceId.trim() !== ''

    const preferred = {
      video: {
        width: options.width ? { ideal: options.width } : undefined,
        height: options.height ? { ideal: options.height } : undefined,
        deviceId: hasTargetDevice ? { exact: deviceId } : undefined,
      },
      audio: false,
    } satisfies MediaStreamConstraints

    if (hasTargetDevice) {
      return navigator.mediaDevices.getUserMedia(preferred)
    }

    try {
      return await navigator.mediaDevices.getUserMedia(preferred)
    } catch {
      return navigator.mediaDevices.getUserMedia({ video: true, audio: false })
    }
  }

  async function start(videoEl: HTMLVideoElement, preferredDeviceId?: string) {
    status.value = 'requesting'
    error.value = null

    try {
      stop()
      await refreshDevices()

      const initialTarget = preferredDeviceId?.trim() || selectedDeviceId.value || undefined
      let targetDeviceId = initialTarget

      try {
        stream.value = await requestStream(targetDeviceId)
      } catch (requestError) {
        const fallback = devices.value.find(device => device.id !== targetDeviceId)?.id
        if (!fallback) {
          throw requestError
        }
        targetDeviceId = fallback
        selectedDeviceId.value = fallback
        stream.value = await requestStream(fallback)
      }

      if (!stream.value) {
        throw new Error('No camera stream available')
      }

      videoEl.srcObject = stream.value
      videoEl.muted = true
      videoEl.playsInline = true
      await videoEl.play()

      const track = stream.value.getVideoTracks()[0]
      const resolvedDeviceId = track?.getSettings().deviceId
      activeDeviceId.value = resolvedDeviceId || targetDeviceId || null
      if (activeDeviceId.value) {
        selectedDeviceId.value = activeDeviceId.value
      }

      activeDeviceMissing.value = false
      await refreshDevices()
      status.value = 'ready'
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to access camera'
      error.value = message
      status.value = 'error'
      throw err
    }
  }

  async function switchDevice(videoEl: HTMLVideoElement, deviceId: string) {
    isSwitching.value = true
    try {
      selectDevice(deviceId)
      await start(videoEl, deviceId)
    } finally {
      isSwitching.value = false
    }
  }

  function stop() {
    if (stream.value) {
      for (const track of stream.value.getTracks()) {
        track.stop()
      }
      stream.value = null
    }
    activeDeviceId.value = null
    activeDeviceMissing.value = false

    if (status.value !== 'error') {
      status.value = 'idle'
    }
  }

  if (hasCameraApi()) {
    void refreshDevices()
    navigator.mediaDevices.addEventListener('devicechange', deviceChangeHandler)
  }

  if (getCurrentScope()) {
    onScopeDispose(() => {
      stop()
      if (hasCameraApi()) {
        navigator.mediaDevices.removeEventListener('devicechange', deviceChangeHandler)
      }
    })
  }

  return {
    status,
    error,
    stream,
    devices,
    selectedDeviceId,
    activeDeviceId,
    activeDeviceMissing,
    isEnumerating,
    isSwitching,
    refreshDevices,
    selectDevice,
    start,
    switchDevice,
    stop,
  }
}
