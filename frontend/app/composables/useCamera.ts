import type { Ref } from 'vue'

const LAST_CAMERA_KEY = 'lychee-ripe.last-camera-device'

export interface CameraOption {
  value: string
  label: string
}

export function useCamera(videoElement: Ref<HTMLVideoElement | null>) {
  const devices = ref<MediaDeviceInfo[]>([])
  const selectedDeviceId = ref('')
  const currentStream = ref<MediaStream | null>(null)
  const isCameraLoading = ref(false)
  const cameraError = ref('')
  const hasPermission = ref(false)

  const hasDevices = computed(() => devices.value.length > 0)
  const options = computed<CameraOption[]>(() =>
    devices.value.map((device, index) => ({
      value: device.deviceId,
      label: device.label || `摄像头 ${index + 1}`
    }))
  )

  async function refreshDevices() {
    if (!import.meta.client || !navigator.mediaDevices?.enumerateDevices) {
      cameraError.value = '当前环境不支持摄像头设备枚举。'
      devices.value = []
      return
    }

    const all = await navigator.mediaDevices.enumerateDevices()
    devices.value = all.filter((device) => device.kind === 'videoinput')

    if (!devices.value.length) {
      selectedDeviceId.value = ''
      return
    }

    const saved = getSavedDeviceId()
    const stillExists = devices.value.some((device) => device.deviceId === selectedDeviceId.value)
    if (!stillExists) {
      const preferred = devices.value.find((device) => device.deviceId === saved)
      selectedDeviceId.value = preferred?.deviceId || devices.value[0]!.deviceId
    }
  }

  async function startCamera(deviceId?: string) {
    if (!import.meta.client || !navigator.mediaDevices?.getUserMedia) {
      cameraError.value = '当前环境不支持摄像头访问。'
      return
    }

    isCameraLoading.value = true
    cameraError.value = ''
    stopCamera()

    const preferredId = deviceId || selectedDeviceId.value
    const triedFallback = { value: false }

    try {
      const stream = await openCameraStream(preferredId)
      await bindStream(stream)
      hasPermission.value = true

      await refreshDevices()
      saveDeviceId(selectedDeviceId.value)
    } catch (error) {
      const canRetryWithoutDevice = Boolean(preferredId) && !triedFallback.value
      if (canRetryWithoutDevice) {
        triedFallback.value = true
        try {
          const stream = await openCameraStream('')
          await bindStream(stream)
          hasPermission.value = true

          await refreshDevices()
          saveDeviceId(selectedDeviceId.value)
        } catch (fallbackError) {
          cameraError.value = getCameraErrorMessage(fallbackError)
        }
      } else {
        cameraError.value = getCameraErrorMessage(error)
      }
    } finally {
      isCameraLoading.value = false
    }
  }

  function stopCamera() {
    if (currentStream.value) {
      currentStream.value.getTracks().forEach((track) => track.stop())
      currentStream.value = null
    }

    if (videoElement.value) {
      videoElement.value.srcObject = null
    }
  }

  async function switchCamera(deviceId: string) {
    if (!deviceId || deviceId === selectedDeviceId.value) {
      return
    }
    selectedDeviceId.value = deviceId
    saveDeviceId(deviceId)

    if (currentStream.value) {
      await startCamera(deviceId)
    }
  }

  async function openCameraStream(deviceId: string) {
    const constraints = deviceId
      ? {
          audio: false,
          video: {
            deviceId: { exact: deviceId }
          }
        }
      : {
          audio: false,
          video: true
        }

    return await navigator.mediaDevices.getUserMedia(constraints)
  }

  async function bindStream(stream: MediaStream) {
    currentStream.value = stream
    const activeTrack = stream.getVideoTracks()[0]
    const activeDeviceId = activeTrack?.getSettings().deviceId
    if (activeDeviceId) {
      selectedDeviceId.value = activeDeviceId
    }

    const video = videoElement.value
    if (video) {
      video.srcObject = stream
      try {
        await video.play()
      } catch {
        // Browser autoplay policy may delay playback until user gesture.
      }
    }
  }

  function handleDeviceChange() {
    void refreshDevices().then(async () => {
      if (!hasDevices.value) {
        stopCamera()
        return
      }

      const activeTrack = currentStream.value?.getVideoTracks()[0]
      const activeDeviceId = activeTrack?.getSettings().deviceId
      if (activeDeviceId && devices.value.some((device) => device.deviceId === activeDeviceId)) {
        return
      }

      if (currentStream.value && selectedDeviceId.value) {
        await startCamera(selectedDeviceId.value)
      }
    })
  }

  function getSavedDeviceId(): string {
    if (!import.meta.client) {
      return ''
    }
    return localStorage.getItem(LAST_CAMERA_KEY) || ''
  }

  function saveDeviceId(deviceId: string) {
    if (!import.meta.client || !deviceId) {
      return
    }
    localStorage.setItem(LAST_CAMERA_KEY, deviceId)
  }

  onMounted(() => {
    void refreshDevices()
    navigator.mediaDevices?.addEventListener?.('devicechange', handleDeviceChange)
  })

  onBeforeUnmount(() => {
    navigator.mediaDevices?.removeEventListener?.('devicechange', handleDeviceChange)
    stopCamera()
  })

  watch(selectedDeviceId, (nextId) => {
    if (nextId) {
      saveDeviceId(nextId)
    }
  })

  return {
    options,
    hasDevices,
    hasPermission,
    selectedDeviceId,
    currentStream,
    isCameraLoading,
    cameraError,
    refreshDevices,
    startCamera,
    stopCamera,
    switchCamera
  }
}

function getCameraErrorMessage(error: unknown): string {
  const err = error as DOMException
  if (err?.name === 'NotAllowedError') {
    return '摄像头权限被拒绝，请在浏览器或系统设置中允许访问。'
  }
  if (err?.name === 'NotFoundError' || err?.name === 'DevicesNotFoundError') {
    return '未检测到可用摄像头设备。'
  }
  if (err?.name === 'NotReadableError') {
    return '摄像头正在被其他应用占用，请关闭后重试。'
  }
  if (err?.name === 'OverconstrainedError') {
    return '选定摄像头不可用，请切换设备后重试。'
  }
  return '摄像头初始化失败，请稍后重试。'
}
