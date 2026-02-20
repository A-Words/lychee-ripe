<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import CameraStage from '../components/CameraStage.vue'
import StatusPanel from '../components/StatusPanel.vue'
import { useCamera } from '../composables/useCamera'
import { useInferenceStream } from '../composables/useInferenceStream'

const TARGET_WIDTH = 640
const TARGET_HEIGHT = 360
const TARGET_FPS = 5
const JPEG_QUALITY = 0.8

const runtimeConfig = useRuntimeConfig()
const gatewayBase = runtimeConfig.public.gatewayBase

const toast = useToast()

const surfaceVideo = ref<HTMLVideoElement | null>(null)

const camera = useCamera({ width: TARGET_WIDTH, height: TARGET_HEIGHT })
const stream = useInferenceStream({ gatewayBase })

const cameraStatus = computed(() => camera.status.value)
const cameraError = computed(() => camera.error.value)
const streamStatus = computed(() => stream.status.value)
const streamError = computed(() => stream.lastError.value)
const sessionSummary = computed(() => stream.sessionSummary.value)

const frameSummary = computed(() => stream.frameResult.value?.frame_summary ?? null)
const detections = computed(() => stream.frameResult.value?.detections ?? [])

const canStart = computed(() => {
  return Boolean(surfaceVideo.value) &&
    streamStatus.value !== 'connecting' &&
    streamStatus.value !== 'streaming' &&
    streamStatus.value !== 'stopping' &&
    cameraStatus.value !== 'requesting'
})

const canStop = computed(() => {
  return streamStatus.value === 'connecting' ||
    streamStatus.value === 'streaming' ||
    streamStatus.value === 'stopping'
})

function handleSurfaceReady(payload: { video: HTMLVideoElement }) {
  surfaceVideo.value = payload.video
}

async function startRecognition() {
  if (!surfaceVideo.value) {
    return
  }

  try {
    await camera.start(surfaceVideo.value)
    await stream.start({
      video: surfaceVideo.value,
      targetWidth: TARGET_WIDTH,
      targetHeight: TARGET_HEIGHT,
      fps: TARGET_FPS,
      jpegQuality: JPEG_QUALITY,
    })
  } catch (err) {
    const description = err instanceof Error ? err.message : 'Failed to start stream'
    toast.add({
      color: 'error',
      title: 'Start failed',
      description,
    })
  }
}

async function stopRecognition() {
  await stream.stop()
  camera.stop()
}

watch(
  () => streamError.value,
  (message, prev) => {
    if (!message || message === prev) {
      return
    }
    toast.add({
      color: 'warning',
      title: 'Stream message',
      description: message,
    })
  },
)

onBeforeUnmount(() => {
  void stopRecognition()
})
</script>

<template>
  <div class="lr-shell space-y-4">
    <UCard class="lr-panel">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">Lychee Ripeness Live Monitor</h1>
          <p class="text-sm text-neutral-600">
            Gateway: <code>{{ gatewayBase }}</code>
          </p>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            icon="i-lucide-play"
            :disabled="!canStart"
            @click="startRecognition"
          >
            Start
          </UButton>
          <UButton
            icon="i-lucide-square"
            color="neutral"
            variant="soft"
            :disabled="!canStop"
            @click="stopRecognition"
          >
            Stop
          </UButton>
        </div>
      </div>
    </UCard>

    <div class="grid gap-4 lg:grid-cols-[2fr_1fr]">
      <div class="space-y-3">
        <CameraStage
          :detections="detections"
          :source-width="TARGET_WIDTH"
          :source-height="TARGET_HEIGHT"
          @surface-ready="handleSurfaceReady"
        />

        <UAlert
          v-if="cameraStatus === 'error'"
          color="error"
          variant="soft"
          title="Camera unavailable"
          :description="cameraError ?? 'Failed to open camera'"
        />

        <UAlert
          v-else-if="cameraStatus === 'idle'"
          color="neutral"
          variant="soft"
          title="Camera is idle"
          description="Click Start to request permission and begin detection."
        />
      </div>

      <StatusPanel
        :stream-status="streamStatus"
        :camera-status="cameraStatus"
        :frame-summary="frameSummary"
        :session-summary="sessionSummary"
        :last-error="streamError"
      />
    </div>
  </div>
</template>
