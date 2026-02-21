<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import CameraStage from '../components/CameraStage.vue'
import StatusPanel from '../components/StatusPanel.vue'
import { useCamera } from '../composables/useCamera'
import { useInferenceStream } from '../composables/useInferenceStream'

const TARGET_WIDTH = 640
const TARGET_HEIGHT = 360
const TARGET_FPS = 5
const JPEG_QUALITY = 0.8
const CAMERA_STORAGE_KEY = 'lychee-ripe.camera.deviceId'

const runtimeConfig = useRuntimeConfig()
const gatewayBase = runtimeConfig.public.gatewayBase

const toast = useToast()

const surfaceVideo = ref<HTMLVideoElement | null>(null)
const isSwitchingCamera = ref(false)

const camera = useCamera({ width: TARGET_WIDTH, height: TARGET_HEIGHT })
const stream = useInferenceStream({ gatewayBase })

const cameraStatus = computed(() => camera.status.value)
const cameraError = computed(() => camera.error.value)
const streamStatus = computed(() => stream.status.value)
const streamError = computed(() => stream.lastError.value)
const sessionSummary = computed(() => stream.sessionSummary.value)

const frameSummary = computed(() => stream.frameResult.value?.frame_summary ?? null)
const detections = computed(() => stream.frameResult.value?.detections ?? [])

const hasCameraDevices = computed(() => camera.devices.value.length > 0)

const cameraLabel = computed(() => {
  const currentId = camera.activeDeviceId.value || camera.selectedDeviceId.value
  if (!currentId) {
    return null
  }
  return camera.devices.value.find(device => device.id === currentId)?.label ?? null
})

function streamStartOptions(video: HTMLVideoElement) {
  return {
    video,
    targetWidth: TARGET_WIDTH,
    targetHeight: TARGET_HEIGHT,
    fps: TARGET_FPS,
    jpegQuality: JPEG_QUALITY,
  }
}

function persistSelectedCamera(deviceId: string) {
  if (!import.meta.client) {
    return
  }
  localStorage.setItem(CAMERA_STORAGE_KEY, deviceId)
}

const isStreaming = computed(() => streamStatus.value === 'streaming')

const canStart = computed(() => {
  return Boolean(surfaceVideo.value) &&
    hasCameraDevices.value &&
    !isSwitchingCamera.value &&
    streamStatus.value !== 'connecting' &&
    streamStatus.value !== 'streaming' &&
    streamStatus.value !== 'stopping' &&
    cameraStatus.value !== 'requesting'
})

const canStop = computed(() => {
  return !isSwitchingCamera.value && (
    streamStatus.value === 'connecting' ||
    streamStatus.value === 'streaming' ||
    streamStatus.value === 'stopping' ||
    cameraStatus.value === 'ready'
  )
})

const cameraSelectDisabled = computed(() => {
  return !surfaceVideo.value ||
    !hasCameraDevices.value ||
    cameraStatus.value === 'requesting' ||
    streamStatus.value === 'connecting' ||
    streamStatus.value === 'stopping' ||
    isSwitchingCamera.value ||
    camera.isSwitching.value
})

function handleSurfaceReady(payload: { video: HTMLVideoElement }) {
  surfaceVideo.value = payload.video
}

async function ensurePreviewOnSelectedCamera(video: HTMLVideoElement) {
  const selected = camera.selectedDeviceId.value || undefined
  if (cameraStatus.value === 'ready' && camera.activeDeviceId.value === selected) {
    return
  }
  await camera.start(video, selected)
}

async function startRecognition() {
  if (!surfaceVideo.value || isSwitchingCamera.value) {
    return
  }

  try {
    await ensurePreviewOnSelectedCamera(surfaceVideo.value)
    await stream.start(streamStartOptions(surfaceVideo.value))
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

async function handleCameraChange(deviceId: string, reason: 'user' | 'system' = 'user') {
  if (!surfaceVideo.value || !deviceId || isSwitchingCamera.value) {
    return
  }

  const previousSelected = camera.selectedDeviceId.value
  const previousActive = camera.activeDeviceId.value
  const wasStreaming = isStreaming.value

  if (reason === 'user' &&
    previousSelected === deviceId &&
    previousActive === deviceId &&
    (cameraStatus.value === 'ready' || wasStreaming)) {
    return
  }

  isSwitchingCamera.value = true
  camera.selectDevice(deviceId)
  persistSelectedCamera(deviceId)

  try {
    if (wasStreaming) {
      await stream.stop()
    }

    await camera.switchDevice(surfaceVideo.value, deviceId)

    if (wasStreaming) {
      await stream.start(streamStartOptions(surfaceVideo.value))
    }

    if (reason === 'user') {
      toast.add({
        color: 'success',
        title: 'Camera switched',
        description: `Now using ${cameraLabel.value ?? 'selected camera'}.`,
      })
    }
  } catch (err) {
    const description = err instanceof Error ? err.message : 'Camera switch failed'
    toast.add({
      color: 'error',
      title: 'Camera switch failed',
      description,
    })

    const rollbackId = previousActive || previousSelected
    if (!rollbackId || rollbackId === deviceId) {
      if (wasStreaming) {
        await stream.stop()
      }
      isSwitchingCamera.value = false
      return
    }

    camera.selectDevice(rollbackId)
    persistSelectedCamera(rollbackId)

    try {
      await camera.switchDevice(surfaceVideo.value, rollbackId)
      if (wasStreaming) {
        await stream.start(streamStartOptions(surfaceVideo.value))
      }
      toast.add({
        color: 'warning',
        title: 'Switched back',
        description: `Recovered to ${cameraLabel.value ?? 'previous camera'}.`,
      })
    } catch (rollbackErr) {
      if (wasStreaming) {
        await stream.stop()
      }
      const rollbackMessage = rollbackErr instanceof Error ? rollbackErr.message : 'Rollback failed'
      toast.add({
        color: 'error',
        title: 'Camera recovery failed',
        description: rollbackMessage,
      })
    }
  } finally {
    isSwitchingCamera.value = false
  }
}

function onCameraSelected(value: unknown) {
  if (typeof value !== 'string' || !value) {
    return
  }
  void handleCameraChange(value, 'user')
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

watch(
  () => camera.activeDeviceMissing.value,
  (missing) => {
    if (!missing || isSwitchingCamera.value) {
      return
    }

    if (!surfaceVideo.value || !camera.selectedDeviceId.value) {
      void stopRecognition()
      toast.add({
        color: 'error',
        title: 'Camera disconnected',
        description: 'No fallback camera is available.',
      })
      return
    }

    toast.add({
      color: 'warning',
      title: 'Camera changed',
      description: 'Current camera is unavailable, switching to fallback.',
    })
    void handleCameraChange(camera.selectedDeviceId.value, 'system')
  },
)

onMounted(async () => {
  await camera.refreshDevices()

  if (import.meta.client) {
    const saved = localStorage.getItem(CAMERA_STORAGE_KEY)
    if (saved) {
      camera.selectDevice(saved)
    }
  }

  await camera.refreshDevices()

  if (!camera.selectedDeviceId.value && camera.devices.value.length > 0) {
    const firstDevice = camera.devices.value[0]
    if (firstDevice) {
      camera.selectDevice(firstDevice.id)
      persistSelectedCamera(firstDevice.id)
    }
  }
})

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

        <div class="flex flex-wrap items-center gap-2">
          <USelect
            :model-value="camera.selectedDeviceId.value || undefined"
            :items="camera.devices.value"
            value-key="id"
            label-key="label"
            placeholder="Select camera"
            class="min-w-60"
            :disabled="cameraSelectDisabled"
            @update:model-value="onCameraSelected"
          />

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
          v-if="!hasCameraDevices"
          color="warning"
          variant="soft"
          title="No camera detected"
          description="Plug in a camera and allow browser permissions."
        />

        <UAlert
          v-else-if="cameraStatus === 'error'"
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
          description="Select a camera then click Start to begin detection."
        />
      </div>

      <StatusPanel
        :stream-status="streamStatus"
        :camera-status="cameraStatus"
        :camera-label="cameraLabel"
        :frame-summary="frameSummary"
        :session-summary="sessionSummary"
        :last-error="streamError"
      />
    </div>
  </div>
</template>
