import { getCurrentScope, onScopeDispose, ref } from 'vue'
import type {
  FrameResult,
  SessionSummary,
  StreamEnvelope,
  StreamErrorEnvelope,
  StreamFrameEnvelope,
  StreamSummaryEnvelope,
} from '../types/infer'

export type StreamStatus = 'idle' | 'connecting' | 'streaming' | 'stopping' | 'stopped' | 'error'

export interface StartStreamOptions {
  video: HTMLVideoElement
  targetWidth?: number
  targetHeight?: number
  fps?: number
  jpegQuality?: number
}

interface UseInferenceStreamOptions {
  gatewayBase?: string
  webSocketFactory?: (url: string) => WebSocketLike
}

interface WebSocketLike {
  readonly readyState: number
  onopen: ((event: Event) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  send(data: string | ArrayBuffer): void
  close(code?: number, reason?: string): void
}

const WS_OPEN = 1
const WS_CLOSING = 2
const WS_CLOSED = 3

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isFrameEnvelope(value: unknown): value is StreamFrameEnvelope {
  return isRecord(value) && value.type === 'frame' && isRecord(value.result)
}

function isSummaryEnvelope(value: unknown): value is StreamSummaryEnvelope {
  return isRecord(value) && value.type === 'summary' && isRecord(value.summary)
}

function isErrorEnvelope(value: unknown): value is StreamErrorEnvelope {
  return isRecord(value) && value.type === 'error' && typeof value.detail === 'string'
}

function parseEnvelope(payload: unknown): StreamEnvelope | null {
  const normalized = typeof payload === 'string' ? JSON.parse(payload) : payload
  if (isFrameEnvelope(normalized) || isSummaryEnvelope(normalized) || isErrorEnvelope(normalized)) {
    return normalized
  }
  return null
}

function resolveGatewayBase(provided?: string): string {
  if (provided) {
    return provided
  }

  try {
    return useRuntimeConfig().public.gatewayBase as string
  } catch {
    return 'http://127.0.0.1:9000'
  }
}

export function toStreamWsUrl(gatewayBase: string): string {
  const base = new URL(gatewayBase)
  base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
  base.pathname = '/v1/infer/stream'
  base.search = ''
  base.hash = ''
  return base.toString()
}

async function frameToArrayBuffer(
  video: HTMLVideoElement,
  canvas: HTMLCanvasElement,
  targetWidth: number,
  targetHeight: number,
  jpegQuality: number,
): Promise<ArrayBuffer | null> {
  canvas.width = targetWidth
  canvas.height = targetHeight

  const ctx = canvas.getContext('2d')
  if (!ctx) {
    return null
  }

  ctx.drawImage(video, 0, 0, targetWidth, targetHeight)

  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob(resolve, 'image/jpeg', jpegQuality)
  })

  return blob ? blob.arrayBuffer() : null
}

export function useInferenceStream(options: UseInferenceStreamOptions = {}) {
  const status = ref<StreamStatus>('idle')
  const lastError = ref<string | null>(null)
  const frameResult = ref<FrameResult | null>(null)
  const sessionSummary = ref<SessionSummary | null>(null)
  const modelVersion = ref<string | null>(null)
  const schemaVersion = ref<string | null>(null)

  const gatewayBase = resolveGatewayBase(options.gatewayBase)

  let ws: WebSocketLike | null = null
  let sendTimer: ReturnType<typeof setInterval> | null = null
  let stoppingDeadline: ReturnType<typeof setTimeout> | null = null
  let frameInFlight = false
  let frameQueued = false

  async function flushFrame(
    video: HTMLVideoElement,
    canvas: HTMLCanvasElement,
    width: number,
    height: number,
    jpegQuality: number,
  ) {
    if (!ws || ws.readyState !== WS_OPEN) {
      return
    }
    if (video.readyState < 2) {
      return
    }

    const frameBuffer = await frameToArrayBuffer(video, canvas, width, height, jpegQuality)
    if (!frameBuffer) {
      return
    }

    if (!ws || ws.readyState !== WS_OPEN) {
      return
    }
    ws.send(frameBuffer)
  }

  function clearTimers() {
    if (sendTimer) {
      clearInterval(sendTimer)
      sendTimer = null
    }
    if (stoppingDeadline) {
      clearTimeout(stoppingDeadline)
      stoppingDeadline = null
    }
    frameInFlight = false
    frameQueued = false
  }

  function cleanupSocket() {
    if (!ws) {
      return
    }

    if (ws.readyState !== WS_CLOSING && ws.readyState !== WS_CLOSED) {
      ws.close()
    }
    ws = null
  }

  function handleServerMessage(payload: unknown): StreamEnvelope | null {
    let envelope: StreamEnvelope | null = null
    try {
      envelope = parseEnvelope(payload)
    } catch {
      lastError.value = 'Failed to parse stream message'
      return null
    }

    if (!envelope) {
      return null
    }

    if (envelope.type === 'frame') {
      modelVersion.value = envelope.model_version
      schemaVersion.value = envelope.schema_version
      frameResult.value = envelope.result
      return envelope
    }

    if (envelope.type === 'summary') {
      modelVersion.value = envelope.model_version
      schemaVersion.value = envelope.schema_version
      sessionSummary.value = envelope.summary
      return envelope
    }

    lastError.value = envelope.detail
    return envelope
  }

  async function start(startOptions: StartStreamOptions) {
    if (status.value === 'connecting' || status.value === 'streaming') {
      return
    }

    status.value = 'connecting'
    lastError.value = null
    frameResult.value = null
    sessionSummary.value = null
    modelVersion.value = null
    schemaVersion.value = null

    const streamUrl = toStreamWsUrl(gatewayBase)
    ws = (options.webSocketFactory ?? ((url: string) => new WebSocket(url) as unknown as WebSocketLike))(streamUrl)

    const openPromise = new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('WebSocket connect timeout')), 5000)
      if (!ws) {
        clearTimeout(timeout)
        reject(new Error('WebSocket is not available'))
        return
      }

      ws.onopen = () => {
        clearTimeout(timeout)
        resolve()
      }

      ws.onerror = () => {
        clearTimeout(timeout)
        reject(new Error('WebSocket connection failed'))
      }
    })

    try {
      await openPromise
      status.value = 'streaming'
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to connect stream'
      lastError.value = message
      status.value = 'error'
      cleanupSocket()
      throw err
    }

    if (!ws) {
      status.value = 'error'
      throw new Error('WebSocket was closed unexpectedly')
    }

    ws.onmessage = (event) => {
      const envelope = handleServerMessage(event.data)
      if (envelope?.type === 'summary' && status.value === 'stopping') {
        cleanupSocket()
        status.value = 'stopped'
      }
    }

    ws.onerror = () => {
      if (status.value !== 'stopped') {
        status.value = 'error'
        lastError.value = lastError.value ?? 'Stream connection interrupted'
      }
    }

    ws.onclose = () => {
      clearTimers()
      if (status.value !== 'error') {
        status.value = 'stopped'
      }
      ws = null
    }

    const fps = startOptions.fps ?? 5
    const targetWidth = startOptions.targetWidth ?? 640
    const targetHeight = startOptions.targetHeight ?? 360
    const jpegQuality = startOptions.jpegQuality ?? 0.8
    const intervalMs = Math.max(50, Math.floor(1000 / fps))
    const frameCanvas = document.createElement('canvas')

    sendTimer = setInterval(() => {
      if (frameInFlight) {
        frameQueued = true
        return
      }

      frameInFlight = true
      void (async () => {
        try {
          do {
            frameQueued = false
            await flushFrame(
              startOptions.video,
              frameCanvas,
              targetWidth,
              targetHeight,
              jpegQuality,
            )
          } while (frameQueued && status.value === 'streaming')
        } finally {
          frameInFlight = false
        }
      })()
    }, intervalMs)
  }

  async function stop() {
    if (status.value === 'idle' || status.value === 'stopped') {
      status.value = 'stopped'
      return
    }

    status.value = 'stopping'
    clearTimers()

    if (ws && ws.readyState === WS_OPEN) {
      ws.send('eos')
      stoppingDeadline = setTimeout(() => {
        cleanupSocket()
        if (status.value !== 'error') {
          status.value = 'stopped'
        }
      }, 1500)
      return
    }

    cleanupSocket()
    status.value = 'stopped'
  }

  if (getCurrentScope()) {
    onScopeDispose(() => {
      void stop()
    })
  }

  return {
    status,
    lastError,
    frameResult,
    sessionSummary,
    modelVersion,
    schemaVersion,
    start,
    stop,
    handleServerMessage,
  }
}
