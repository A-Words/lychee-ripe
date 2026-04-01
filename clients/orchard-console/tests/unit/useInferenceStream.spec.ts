import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useInferenceStream } from '../../app/composables/useInferenceStream'

interface Deferred<T> {
  promise: Promise<T>
  resolve: (value: T) => void
}

interface BlobLike {
  arrayBuffer: () => Promise<ArrayBuffer>
}

class FakeWebSocket {
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSING = 2
  static readonly CLOSED = 3
  static instances: FakeWebSocket[] = []

  readonly sent: Array<string | ArrayBufferLike | ArrayBuffer> = []
  readyState = FakeWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  closeCalls = 0

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this)
  }

  send(data: string | ArrayBufferLike | ArrayBuffer) {
    if (this.readyState !== FakeWebSocket.OPEN) {
      throw new Error('socket is not open')
    }

    this.sent.push(data)
  }

  close() {
    this.closeCalls += 1
    if (this.readyState !== FakeWebSocket.CLOSED) {
      this.readyState = FakeWebSocket.CLOSING
    }
  }

  open() {
    this.readyState = FakeWebSocket.OPEN
    this.onopen?.({} as Event)
  }

  emitMessage(payload: unknown) {
    this.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent)
  }

  emitError() {
    this.onerror?.({} as Event)
  }

  serverClose() {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.({} as CloseEvent)
  }
}

function createDeferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve
  })

  return { promise, resolve }
}

function createSummaryEnvelope() {
  return {
    type: 'summary' as const,
    model_version: '1.0.0',
    schema_version: 'v1',
    summary: {
      total_detected: 3,
      ripeness_ratio: {
        green: 0,
        half: 1 / 3,
        red: 2 / 3,
        young: 0
      },
      harvest_suggestion: 'ready' as const
    }
  }
}

function createFakeVideo(): HTMLVideoElement {
  return {
    readyState: 2,
    videoWidth: 640,
    videoHeight: 480
  } as HTMLVideoElement
}

describe('useInferenceStream', () => {
  let nextBlob: BlobLike | null
  let drawImage: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(console, 'warn').mockImplementation(() => {})

    nextBlob = null
    drawImage = vi.fn()

    vi.stubGlobal('useRuntimeConfig', () => ({
      public: {
        gatewayBase: 'http://127.0.0.1:9000'
      }
    }))

    vi.stubGlobal('window', globalThis)
    vi.stubGlobal('WebSocket', FakeWebSocket as unknown as typeof WebSocket)
    vi.stubGlobal('document', {
      createElement: vi.fn(() => ({
        width: 0,
        height: 0,
        getContext: vi.fn(() => ({
          drawImage
        })),
        toBlob: (callback: (blob: Blob | null) => void) => {
          callback(nextBlob as Blob | null)
        }
      }))
    })
  })

  afterEach(() => {
    FakeWebSocket.instances = []
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('waits for the final summary before finishing a manual stop', async () => {
    const stream = useInferenceStream({
      videoElement: ref(createFakeVideo()),
      frameIntervalMs: 50
    })

    await stream.startStream()
    const socket = FakeWebSocket.instances[0]!
    socket.open()

    const stopPromise = stream.stopStream()
    expect(socket.sent).toEqual(['stop'])
    expect(stream.serverSummary.value).toBeNull()
    expect(stream.connectionState.value).toBe('streaming')

    const summary = createSummaryEnvelope()
    socket.emitMessage(summary)

    expect(stream.serverSummary.value).toEqual(summary.summary)
    expect(socket.closeCalls).toBe(1)
    expect(stream.connectionState.value).toBe('streaming')

    socket.serverClose()
    await stopPromise

    expect(stream.connectionState.value).toBe('idle')
    expect(stream.streamError.value).toBe('')
  })

  it('drops an encoded frame when stop is requested mid-send', async () => {
    const stream = useInferenceStream({
      videoElement: ref(createFakeVideo()),
      frameIntervalMs: 10
    })

    await stream.startStream()
    const socket = FakeWebSocket.instances[0]!
    socket.open()

    const frameBuffer = createDeferred<ArrayBuffer>()
    nextBlob = {
      arrayBuffer: vi.fn(() => frameBuffer.promise)
    }

    vi.advanceTimersByTime(10)
    await Promise.resolve()

    const stopPromise = stream.stopStream()
    expect(socket.sent).toEqual(['stop'])

    frameBuffer.resolve(new ArrayBuffer(8))
    await Promise.resolve()

    expect(socket.sent).toEqual(['stop'])
    expect(stream.connectionState.value).toBe('streaming')
    expect(stream.streamError.value).toBe('')

    socket.emitMessage(createSummaryEnvelope())
    socket.serverClose()
    await stopPromise

    expect(stream.connectionState.value).toBe('idle')
    expect(stream.streamError.value).toBe('')
  })

  it('still reports an unexpected close as a stream error', async () => {
    const stream = useInferenceStream({
      videoElement: ref(createFakeVideo()),
      frameIntervalMs: 50
    })

    await stream.startStream()
    const socket = FakeWebSocket.instances[0]!
    socket.open()

    socket.serverClose()

    expect(stream.connectionState.value).toBe('error')
    expect(stream.streamError.value).toBe('识别流连接中断，请重试。')
  })
})
