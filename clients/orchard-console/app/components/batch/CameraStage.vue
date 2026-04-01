<script setup lang="ts">
import type { CameraOption } from '~/composables/useCamera'
import type { FrameResult } from '~/types/infer'
import { mapDetectionsToOverlayBoxes } from '~/utils/camera-overlay'

const props = defineProps<{
  devices: CameraOption[]
  selectedDeviceId: string
  currentFrame: FrameResult | null
  isRecognizing: boolean
  cameraLoading: boolean
  cameraError?: string
  streamError?: string
}>()

const emit = defineEmits<{
  (event: 'update:videoElement', value: HTMLVideoElement | null): void
  (event: 'update:selectedDeviceId', value: string): void
  (event: 'start'): void
  (event: 'stop'): void
  (event: 'refresh'): void
}>()

const localVideo = ref<HTMLVideoElement | null>(null)
const stageViewport = ref<HTMLElement | null>(null)
const viewportSize = reactive({
  width: 0,
  height: 0
})
const videoSize = reactive({
  width: 0,
  height: 0
})
let resizeObserver: ResizeObserver | null = null

watch(localVideo, (video) => {
  emit('update:videoElement', video)
}, { immediate: true })

watch(localVideo, (video, previousVideo) => {
  previousVideo?.removeEventListener('loadedmetadata', syncVideoSize)
  video?.addEventListener('loadedmetadata', syncVideoSize)
  syncVideoSize()
}, { immediate: true })

watch(stageViewport, (element, previousElement) => {
  if (resizeObserver && previousElement) {
    resizeObserver.unobserve(previousElement)
  }

  if (resizeObserver && element) {
    resizeObserver.observe(element)
  }

  syncViewportSize()
}, { immediate: true })

onBeforeUnmount(() => {
  localVideo.value?.removeEventListener('loadedmetadata', syncVideoSize)
  resizeObserver?.disconnect()
  resizeObserver = null
  emit('update:videoElement', null)
})

const hasDevices = computed(() => props.devices.length > 0)
const showOverlay = computed(() => props.isRecognizing && !props.streamError && Boolean(props.currentFrame))
const overlayBoxes = computed(() => {
  if (!props.currentFrame) {
    return []
  }

  return mapDetectionsToOverlayBoxes(props.currentFrame.detections, {
    videoWidth: videoSize.width,
    videoHeight: videoSize.height,
    containerWidth: viewportSize.width,
    containerHeight: viewportSize.height
  })
})

onMounted(() => {
  if (typeof ResizeObserver === 'undefined') {
    syncViewportSize()
    return
  }

  resizeObserver = new ResizeObserver(() => {
    syncViewportSize()
  })

  if (stageViewport.value) {
    resizeObserver.observe(stageViewport.value)
  }

  syncViewportSize()
  syncVideoSize()
})

function syncViewportSize() {
  const element = stageViewport.value
  if (!element) {
    viewportSize.width = 0
    viewportSize.height = 0
    return
  }

  viewportSize.width = element.clientWidth
  viewportSize.height = element.clientHeight
}

function syncVideoSize() {
  const video = localVideo.value
  if (!video) {
    videoSize.width = 0
    videoSize.height = 0
    return
  }

  videoSize.width = video.videoWidth || 0
  videoSize.height = video.videoHeight || 0
}

function overlayStyle(box: { left: number, top: number, width: number, height: number, color: string }) {
  return {
    left: `${box.left}px`,
    top: `${box.top}px`,
    width: `${box.width}px`,
    height: `${box.height}px`,
    borderColor: box.color,
    boxShadow: `inset 0 0 0 1px ${box.color}40`
  }
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            实时识别
          </h2>
          <p class="mt-1 text-xs text-muted">
            通过网关 WebSocket 将视频帧发送到识别服务。
          </p>
        </div>
        <UBadge :color="isRecognizing ? 'success' : 'neutral'" variant="soft">
          {{ isRecognizing ? '识别中' : '空闲' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_auto_auto]">
        <USelect
          :model-value="selectedDeviceId"
          :items="devices"
          value-key="value"
          label-key="label"
          icon="i-lucide-camera"
          :disabled="cameraLoading || !hasDevices"
          placeholder="选择摄像头"
          @update:model-value="(value) => emit('update:selectedDeviceId', value as string)"
        />

        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="cameraLoading"
          label="刷新设备"
          @click="emit('refresh')"
        />

        <UButton
          v-if="!isRecognizing"
          color="primary"
          icon="i-lucide-play"
          :loading="cameraLoading"
          :disabled="cameraLoading"
          label="开始识别"
          @click="emit('start')"
        />
        <UButton
          v-else
          color="error"
          variant="outline"
          icon="i-lucide-square"
          label="停止识别"
          @click="emit('stop')"
        />
      </div>

      <UAlert
        v-if="!hasDevices"
        color="warning"
        variant="subtle"
        icon="i-lucide-camera-off"
        title="未检测到摄像头"
        description="可先点击开始识别触发授权，再刷新设备列表。"
      />

      <UAlert
        v-if="cameraError"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="摄像头异常"
        :description="cameraError"
      />

      <UAlert
        v-if="streamError"
        color="error"
        variant="subtle"
        icon="i-lucide-wifi-off"
        title="识别流异常"
      >
        <template #description>
          <div class="space-y-2">
            <p class="text-sm">
              {{ streamError }}
            </p>
            <UButton
              v-if="hasDevices && !isRecognizing"
              size="xs"
              color="error"
              variant="outline"
              icon="i-lucide-rotate-cw"
              label="重试连接"
              @click="emit('start')"
            />
          </div>
        </template>
      </UAlert>

      <div
        ref="stageViewport"
        data-testid="camera-stage-viewport"
        class="relative overflow-hidden rounded-lg border border-accented bg-black"
      >
        <video
          ref="localVideo"
          autoplay
          muted
          playsinline
          class="aspect-video w-full object-cover"
        />

        <div
          v-if="showOverlay"
          data-testid="camera-overlay"
          class="pointer-events-none absolute inset-0"
        >
          <div
            v-for="box in overlayBoxes"
            :key="box.key"
            class="absolute rounded-md border-2"
            :style="overlayStyle(box)"
          />
        </div>
      </div>
    </div>
  </UCard>
</template>
