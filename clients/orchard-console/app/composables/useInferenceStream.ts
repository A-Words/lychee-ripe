import type { Ref } from 'vue'
import type { SessionSummary, StreamEnvelope, StreamFrameEnvelope } from '~/types/infer'
import {
  applyDetectionsToSession,
  buildSessionAggregateSummary,
  createSessionAggregateState
} from '~/utils/session-aggregator'
import { toWebSocketBase } from '~/utils/ws-url'

export interface UseInferenceStreamOptions {
  videoElement: Ref<HTMLVideoElement | null>
  frameIntervalMs?: number
  jpegQuality?: number
}

export function useInferenceStream(options: UseInferenceStreamOptions) {
  const gatewayBase = useGatewayBase()

  const frameIntervalMs = options.frameIntervalMs ?? 300
  const jpegQuality = options.jpegQuality ?? 0.8

  const websocket = ref<WebSocket | null>(null)
  const canvas = ref<HTMLCanvasElement | null>(null)
  const sendTimer = ref<number | null>(null)
  const isSendingFrame = ref(false)
  const manualStop = ref(false)

  const connectionState = ref<'idle' | 'connecting' | 'streaming' | 'error'>('idle')
  const streamError = ref('')
  const lastFrame = ref<StreamFrameEnvelope['result'] | null>(null)
  const serverSummary = ref<SessionSummary | null>(null)
  const aggregateState = ref(createSessionAggregateState())

  const isStreaming = computed(() => connectionState.value === 'streaming')
  const aggregateSummary = computed(() => buildSessionAggregateSummary(aggregateState.value))

  async function startStream() {
    if (!import.meta.client || isStreaming.value || connectionState.value === 'connecting') {
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
    manualStop.value = false
    resetSession()

    const wsUrl = `${toWebSocketBase(gatewayBase.value)}/v1/infer/stream`
    const socket = new WebSocket(wsUrl)

    socket.onopen = () => {
      connectionState.value = 'streaming'
      streamError.value = ''
      startFrameLoop()
    }

    socket.onmessage = (event) => {
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
        return
      }

      if (payload.type === 'error') {
        streamError.value = payload.detail || '识别流返回错误。'
      }
    }

    socket.onerror = () => {
      streamError.value = '识别流连接失败，请检查网关与模型服务状态。'
      connectionState.value = 'error'
      stopFrameLoop()
    }

    socket.onclose = () => {
      stopFrameLoop()
      websocket.value = null
      if (manualStop.value) {
        connectionState.value = 'idle'
        return
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
      manualStop.value = false
      if (connectionState.value !== 'error') {
        connectionState.value = 'idle'
      }
      return
    }

    manualStop.value = true
    try {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send('stop')
      }
    } catch {
      // Ignore send errors during shutdown.
    } finally {
      socket.close()
      websocket.value = null
      if (connectionState.value !== 'error') {
        connectionState.value = 'idle'
      }
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

      const buffer = await blob.arrayBuffer()
      socket.send(buffer)
    } catch {
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
