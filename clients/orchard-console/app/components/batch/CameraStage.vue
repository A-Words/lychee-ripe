<script setup lang="ts">
import type { CameraOption } from '~/composables/useCamera'
import type { FrameResult } from '~/types/infer'
import {
  formatDetectionConfidence,
  getOverlayInfoCardPosition,
  mapDetectionsToOverlayBoxes,
  reconcileOverlayBoxIdentities,
  type DetectionOverlayBox
} from '~/utils/camera-overlay'

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
const previewDetectionId = ref<string | null>(null)
const previewRenderKey = ref<string | null>(null)
const selectedDetectionId = ref<string | null>(null)
const selectedRenderKey = ref<string | null>(null)
const overlayBoxes = ref<DetectionOverlayBox[]>([])
let previousOverlayBoxes: DetectionOverlayBox[] = []
let nextUntrackedOverlayId = 0
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
const mappedOverlayBoxes = computed(() => {
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
const activeBox = computed(() => {
  if (!showOverlay.value) {
    return null
  }

  if (selectedDetectionId.value) {
    return overlayBoxes.value.find((box) => box.hoverId === selectedDetectionId.value) ?? null
  }

  if (selectedRenderKey.value) {
    return overlayBoxes.value.find((box) => box.renderKey === selectedRenderKey.value) ?? null
  }

  if (previewDetectionId.value) {
    return overlayBoxes.value.find((box) => box.hoverId === previewDetectionId.value) ?? null
  }

  if (previewRenderKey.value) {
    return overlayBoxes.value.find((box) => box.renderKey === previewRenderKey.value) ?? null
  }

  return null
})

watch(showOverlay, (visible) => {
  if (!visible) {
    previewDetectionId.value = null
    previewRenderKey.value = null
    selectedDetectionId.value = null
    selectedRenderKey.value = null
    overlayBoxes.value = []
    previousOverlayBoxes = []
    nextUntrackedOverlayId = 0
  }
})

watch([mappedOverlayBoxes, showOverlay], ([boxes, visible]) => {
  if (!visible) {
    return
  }

  const reconciled = reconcileOverlayBoxIdentities(previousOverlayBoxes, boxes, nextUntrackedOverlayId)
  overlayBoxes.value = reconciled.boxes
  previousOverlayBoxes = reconciled.boxes
  nextUntrackedOverlayId = reconciled.nextUntrackedId
}, { immediate: true })

watch(overlayBoxes, (boxes) => {
  if (selectedDetectionId.value) {
    const stillExists = boxes.some((box) => box.hoverId === selectedDetectionId.value)
    if (!stillExists) {
      selectedDetectionId.value = null
    }
  }

  if (selectedRenderKey.value) {
    const stillExists = boxes.some((box) => box.renderKey === selectedRenderKey.value)
    if (!stillExists) {
      selectedRenderKey.value = null
    }
  }

  if (previewDetectionId.value) {
    const stillExists = boxes.some((box) => box.hoverId === previewDetectionId.value)
    if (!stillExists) {
      previewDetectionId.value = null
    }
  }

  if (previewRenderKey.value) {
    const stillExists = boxes.some((box) => box.renderKey === previewRenderKey.value)
    if (!stillExists) {
      previewRenderKey.value = null
    }
  }
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

function overlayInfoCardStyle(box: DetectionOverlayBox) {
  const position = getOverlayInfoCardPosition(box, {
    containerWidth: viewportSize.width,
    containerHeight: viewportSize.height
  })

  return {
    left: `${position.left}px`,
    top: `${position.top}px`
  }
}

function handleBoxEnter(box: DetectionOverlayBox) {
  previewDetectionId.value = box.hoverId
  previewRenderKey.value = box.renderKey
}

function handleBoxFocus(box: DetectionOverlayBox) {
  handleBoxEnter(box)
}

function handleBoxLeave(box: DetectionOverlayBox) {
  if (box.hoverId && previewDetectionId.value === box.hoverId) {
    previewDetectionId.value = null
  }

  if (previewRenderKey.value === box.renderKey) {
    previewRenderKey.value = null
  }
}

function handleBoxBlur(box: DetectionOverlayBox) {
  handleBoxLeave(box)
}

function handleBoxClick(box: DetectionOverlayBox) {
  const isSameTrackedBox = box.hoverId && selectedDetectionId.value === box.hoverId
  const isSameFallbackBox = !box.hoverId && selectedRenderKey.value === box.renderKey

  if (isSameTrackedBox || isSameFallbackBox) {
    if (box.hoverId && previewDetectionId.value === box.hoverId) {
      previewDetectionId.value = null
    }

    if (previewRenderKey.value === box.renderKey) {
      previewRenderKey.value = null
    }

    selectedDetectionId.value = null
    selectedRenderKey.value = null
    return
  }

  selectedDetectionId.value = box.hoverId
  selectedRenderKey.value = box.renderKey
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
          <button
            v-for="box in overlayBoxes"
            :key="box.renderKey"
            type="button"
            class="pointer-events-auto absolute rounded-md border-2 bg-transparent cursor-help focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            :style="overlayStyle(box)"
            :aria-label="`${box.label}，置信度 ${formatDetectionConfidence(box.confidence)}`"
            @mouseenter="handleBoxEnter(box)"
            @mouseleave="handleBoxLeave(box)"
            @focus="handleBoxFocus(box)"
            @blur="handleBoxBlur(box)"
            @click="handleBoxClick(box)"
          />

          <div
            v-if="activeBox"
            data-testid="camera-overlay-card"
            class="pointer-events-none absolute w-28 rounded-md border border-default bg-default/92 px-2 py-1.5 text-xs text-highlighted shadow-lg backdrop-blur-sm"
            :style="overlayInfoCardStyle(activeBox)"
          >
            <p class="font-medium leading-tight">
              成熟度：{{ activeBox.label }}
            </p>
            <p class="mt-1 leading-tight text-toned">
              置信度：{{ formatDetectionConfidence(activeBox.confidence) }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </UCard>
</template>
