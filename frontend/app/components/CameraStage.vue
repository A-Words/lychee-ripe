<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Detection } from '../types/infer'
import { useDetectionOverlay } from '../composables/useDetectionOverlay'

interface Props {
  detections: Detection[]
  sourceWidth: number
  sourceHeight: number
}

const props = withDefaults(defineProps<Props>(), {
  detections: () => [],
  sourceWidth: 640,
  sourceHeight: 360,
})

const emit = defineEmits<{
  (e: 'surface-ready', payload: { video: HTMLVideoElement; overlay: HTMLCanvasElement }): void
}>()

const wrapperEl = ref<HTMLDivElement | null>(null)
const videoEl = ref<HTMLVideoElement | null>(null)
const overlayEl = ref<HTMLCanvasElement | null>(null)

const { clear, draw } = useDetectionOverlay()

let resizeObserver: ResizeObserver | null = null

function syncCanvasSize() {
  if (!videoEl.value || !overlayEl.value) {
    return
  }

  const rect = videoEl.value.getBoundingClientRect()
  const width = Math.max(1, Math.round(rect.width))
  const height = Math.max(1, Math.round(rect.height))

  if (overlayEl.value.width !== width || overlayEl.value.height !== height) {
    overlayEl.value.width = width
    overlayEl.value.height = height
  }
}

function renderOverlay() {
  if (!overlayEl.value) {
    return
  }

  const ctx = overlayEl.value.getContext('2d')
  if (!ctx) {
    return
  }

  if (!props.detections.length) {
    clear(ctx, overlayEl.value.width, overlayEl.value.height)
    return
  }

  draw({
    context: ctx,
    detections: props.detections,
    sourceWidth: props.sourceWidth,
    sourceHeight: props.sourceHeight,
    canvasWidth: overlayEl.value.width,
    canvasHeight: overlayEl.value.height,
  })
}

onMounted(() => {
  syncCanvasSize()
  renderOverlay()

  if (wrapperEl.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      syncCanvasSize()
      renderOverlay()
    })
    resizeObserver.observe(wrapperEl.value)
  }

  if (videoEl.value && overlayEl.value) {
    emit('surface-ready', { video: videoEl.value, overlay: overlayEl.value })
  }
})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

watch(
  () => [props.detections, props.sourceWidth, props.sourceHeight] as const,
  () => {
    syncCanvasSize()
    renderOverlay()
  },
  { deep: true },
)
</script>

<template>
  <div ref="wrapperEl" class="relative aspect-video w-full overflow-hidden rounded-xl border border-black/10 bg-black">
    <video
      ref="videoEl"
      class="h-full w-full object-cover"
      autoplay
      muted
      playsinline
    />
    <canvas ref="overlayEl" class="pointer-events-none absolute inset-0 h-full w-full" />
  </div>
</template>
