import { getCurrentScope, onScopeDispose, ref } from 'vue'

type CameraStatus = 'idle' | 'requesting' | 'ready' | 'error'

interface CameraOptions {
  width?: number
  height?: number
}

export function useCamera(options: CameraOptions = {}) {
  const status = ref<CameraStatus>('idle')
  const error = ref<string | null>(null)
  const stream = ref<MediaStream | null>(null)

  async function requestStream() {
    if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
      throw new Error('Current environment does not support camera APIs')
    }

    const preferred = {
      video: {
        width: options.width ? { ideal: options.width } : undefined,
        height: options.height ? { ideal: options.height } : undefined,
      },
      audio: false,
    } satisfies MediaStreamConstraints

    try {
      return await navigator.mediaDevices.getUserMedia(preferred)
    } catch {
      return navigator.mediaDevices.getUserMedia({ video: true, audio: false })
    }
  }

  async function start(videoEl: HTMLVideoElement) {
    status.value = 'requesting'
    error.value = null

    try {
      stop()
      stream.value = await requestStream()
      videoEl.srcObject = stream.value
      videoEl.muted = true
      videoEl.playsInline = true
      await videoEl.play()
      status.value = 'ready'
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to access camera'
      error.value = message
      status.value = 'error'
      throw err
    }
  }

  function stop() {
    if (stream.value) {
      for (const track of stream.value.getTracks()) {
        track.stop()
      }
      stream.value = null
    }

    if (status.value !== 'error') {
      status.value = 'idle'
    }
  }

  if (getCurrentScope()) {
    onScopeDispose(stop)
  }

  return {
    status,
    error,
    stream,
    start,
    stop,
  }
}
