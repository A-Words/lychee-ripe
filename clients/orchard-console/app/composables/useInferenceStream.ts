import { computed, onBeforeUnmount, ref, shallowRef, type Ref } from 'vue'
import type { SessionSummary, StreamEnvelope, StreamFrameEnvelope } from '~/types/infer'
import {
  applyDetectionsToSession,
  buildSessionAggregateSummary,
  createSessionAggregateState
} from '~/utils/session-aggregator'
import { useAuth } from '~/composables/useAuth'

export interface UseInferenceStreamOptions {
  videoElement: Ref<HTMLVideoElement | null>
  frameIntervalMs?: number
  jpegQuality?: number
  auth?: {
    init: () => Promise<void>
    websocketUrl: (path: string) => string
  }
}

const STOP_TIMEOUT_MS = 1000

export function useInferenceStream(options: UseInferenceStreamOptions) {
  const auth = options.auth ?? useAuth()

  const frameIntervalMs = options.frameIntervalMs ?? 300
  const jpegQuality = options.jpegQuality ?? 0.8

  const websocket = shallowRef<WebSocket | null>(null)
  const canvas = shallowRef<HTMLCanvasElement | null>(null)
  const sendTimer = ref<number | null>(null)
  const isSendingFrame = ref(false)
  const manualStop = ref(false)
  let stoppingSocket: WebSocket | null = null
  let stopPromise: Promise<void> | null = null
  let resolveStopPromise: (() => void) | null = null
  let stopTimeout: number | null = null

  const connectionState = ref<'idle' | 'connecting' | 'streaming' | 'error'>('idle')
  const streamError = ref('')
  const lastFrame = ref<StreamFrameEnvelope['result'] | null>(null)
  const serverSummary = ref<SessionSummary | null>(null)
  const aggregateState = ref(createSessionAggregateState())

  const isStreaming = computed(() => connectionState.value === 'streaming')
  const aggregateSummary = computed(() => buildSessionAggregateSummary(aggregateState.value))

  async function startStream() {
    if (typeof window === 'undefined' || typeof WebSocket === 'undefined') {
      return
    }

    if (isStreaming.value || connectionState.value === 'connecting') {
      return
    }

    const video = options.videoElement.value
    if (!video) {
      streamError.value = '摄像头未就绪，无法启动识别。'
      connectionState.value = 'error'
      return
    }

    connectionState.value = 'connecting'
    streamError.value = ''
    resetManualStop()
    resetSession()

    await auth.init()
    const wsUrl = auth.websocketUrl('/v1/infer/stream')
    const socket = new WebSocket(wsUrl)

    socket.onopen = () => {
      if (websocket.value !== socket) {
        return
      }
      connectionState.value = 'streaming'
      streamError.value = ''
      startFrameLoop()
    }

    socket.onmessage = (event) => {
      if (!isTrackedSocket(socket)) {
        return
      }

      const payload = parseStreamEnvelope(event.data)
      if (!payload) {
        return
      }

      if (payload.type === 'frame') {
        lastFrame.value = payload.result
        aggregateState.value = applyDetectionsToSession(aggregateState.value, payload.result.detections)
        return
      }

      if (payload.type === 'summary') {
        serverSummary.value = payload.summary
        if (manualStop.value && stoppingSocket === socket) {
          requestSocketClose(socket)
        }
        return
      }

      if (payload.type === 'error') {
        streamError.value = payload.detail || '识别流返回错误。'
      }
    }

    socket.onerror = () => {
      if (!isTrackedSocket(socket)) {
        return
      }
      if (manualStop.value && stoppingSocket === socket) {
        return
      }
      streamError.value = '识别流连接失败，请检查网关与模型服务状态。'
      connectionState.value = 'error'
      stopFrameLoop()
    }

    socket.onclose = () => {
      if (!isTrackedSocket(socket)) {
        return
      }

      stopFrameLoop()

      if (manualStop.value && stoppingSocket === socket) {
        finishManualStop(socket)
        return
      }

      if (websocket.value === socket) {
        websocket.value = null
      }

      if (connectionState.value !== 'error') {
        streamError.value = '识别流连接中断，请重试。'
        connectionState.value = 'error'
      }
    }

    websocket.value = socket
  }

  async function stopStream() {
    stopFrameLoop()

    const socket = websocket.value
    if (!socket) {
      resetManualStop()
      if (connectionState.value !== 'error') {
        connectionState.value = 'idle'
      }
      return
    }

    if (stopPromise && stoppingSocket === socket) {
      await stopPromise
      return
    }

    manualStop.value = true
    stoppingSocket = socket
    stopPromise = new Promise<void>((resolve) => {
      resolveStopPromise = resolve
    })
    armStopTimeout(socket)

    try {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send('stop')
      } else if (socket.readyState === WebSocket.CLOSED) {
        finishManualStop(socket)
      } else if (socket.readyState === WebSocket.CONNECTING) {
        requestSocketClose(socket)
      }
    } catch {
      requestSocketClose(socket)
    }

    if (stopPromise) {
      await stopPromise
    }
  }

  function resetSession() {
    aggregateState.value = createSessionAggregateState()
    serverSummary.value = null
    lastFrame.value = null
  }

  function setErrorState(message: string) {
    streamError.value = message
    connectionState.value = 'error'
    stopFrameLoop()
  }

  function isTrackedSocket(socket: WebSocket) {
    return websocket.value === socket || stoppingSocket === socket
  }

  function armStopTimeout(socket: WebSocket) {
    clearStopTimeout()
    stopTimeout = window.setTimeout(() => {
      if (stoppingSocket !== socket) {
        return
      }

      requestSocketClose(socket)
      finishManualStop(socket)
    }, STOP_TIMEOUT_MS)
  }

  function clearStopTimeout() {
    if (stopTimeout !== null) {
      window.clearTimeout(stopTimeout)
      stopTimeout = null
    }
  }

  function requestSocketClose(socket: WebSocket) {
    if (socket.readyState === WebSocket.CLOSED || socket.readyState === WebSocket.CLOSING) {
      return
    }

    try {
      socket.close()
    } catch {
      // Ignore close errors during shutdown.
    }
  }

  function finishManualStop(socket: WebSocket) {
    clearStopTimeout()
    if (websocket.value === socket) {
      websocket.value = null
    }
    resetManualStop()
    if (connectionState.value !== 'error') {
      connectionState.value = 'idle'
    }
  }

  function resetManualStop() {
    clearStopTimeout()
    manualStop.value = false
    stoppingSocket = null

    const resolve = resolveStopPromise
    stopPromise = null
    resolveStopPromise = null
    resolve?.()
  }

  function startFrameLoop() {
    stopFrameLoop()
    sendTimer.value = window.setInterval(() => {
      void sendCurrentFrame()
    }, frameIntervalMs)
  }

  function stopFrameLoop() {
    if (sendTimer.value !== null) {
      window.clearInterval(sendTimer.value)
      sendTimer.value = null
    }
  }

  async function sendCurrentFrame() {
    if (isSendingFrame.value) {
      return
    }

    const socket = websocket.value
    const video = options.videoElement.value
    if (!socket || socket.readyState !== WebSocket.OPEN || !video || video.readyState < 2) {
      return
    }

    const width = video.videoWidth || 0
    const height = video.videoHeight || 0
    if (width <= 0 || height <= 0) {
      return
    }

    isSendingFrame.value = true
    try {
      const frameCanvas = ensureCanvas(width, height)
      const context = frameCanvas.getContext('2d')
      if (!context) {
        return
      }

      context.drawImage(video, 0, 0, width, height)
      const blob = await canvasToJpeg(frameCanvas, jpegQuality)
      if (!blob) {
        return
      }

      if (manualStop.value && stoppingSocket === socket) {
        return
      }
      if (websocket.value !== socket || socket.readyState !== WebSocket.OPEN) {
        return
      }

      const buffer = await blob.arrayBuffer()
      if (manualStop.value && stoppingSocket === socket) {
        return
      }
      if (websocket.value !== socket || socket.readyState !== WebSocket.OPEN) {
        return
      }

      socket.send(buffer)
    } catch {
      if (manualStop.value && stoppingSocket === socket) {
        return
      }
      if (websocket.value !== socket || socket.readyState !== WebSocket.OPEN) {
        return
      }
      setErrorState('推送视频帧失败，请重试识别。')
    } finally {
      isSendingFrame.value = false
    }
  }

  function ensureCanvas(width: number, height: number) {
    const current = canvas.value || document.createElement('canvas')
    if (current.width !== width) {
      current.width = width
    }
    if (current.height !== height) {
      current.height = height
    }
    canvas.value = current
    return current
  }

  onBeforeUnmount(() => {
    void stopStream()
  })

  return {
    connectionState,
    streamError,
    isStreaming,
    lastFrame,
    serverSummary,
    aggregateSummary,
    resetSession,
    startStream,
    stopStream
  }
}

function parseStreamEnvelope(raw: unknown): StreamEnvelope | null {
  if (typeof raw !== 'string') {
    return null
  }

  let payload: unknown
  try {
    payload = JSON.parse(raw)
  } catch {
    return null
  }

  if (!payload || typeof payload !== 'object' || !('type' in payload)) {
    return null
  }

  const record = payload as Record<string, unknown>
  if (record.type === 'frame') {
    return payload as StreamFrameEnvelope
  }
  if (record.type === 'summary') {
    return payload as StreamEnvelope
  }
  if (record.type === 'error') {
    return payload as StreamEnvelope
  }
  return null
}

function canvasToJpeg(canvas: HTMLCanvasElement, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) => resolve(blob),
      'image/jpeg',
      quality
    )
  })
}
