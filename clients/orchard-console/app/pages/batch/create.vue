<script setup lang="ts">
import type { Batch, BatchCreateApiError, BatchCreateFormInput } from '~/types/batch'
import {
  buildBatchCreateRequest,
  useBatchCreate,
  validateBatchSummaryInput
} from '~/composables/useBatchCreate'
import { useCamera } from '~/composables/useCamera'
import { useInferenceStream } from '~/composables/useInferenceStream'
import { toBatchSummaryInput } from '~/utils/session-aggregator'

useSeoMeta({
  title: '识别建批',
  description: '实时识别荔枝成熟度并创建采摘批次。'
})

const videoElement = ref<HTMLVideoElement | null>(null)

const camera = useCamera(videoElement)
const stream = useInferenceStream({
  videoElement,
  frameIntervalMs: 300,
  jpegQuality: 0.8
})
const { createBatch, parseCreateError } = useBatchCreate()
const {
  startStream,
  stopStream,
  resetSession,
  isStreaming,
  streamError,
  lastFrame,
  serverSummary,
  aggregateSummary
} = stream

const submitting = ref(false)
const submitError = ref<BatchCreateApiError | null>(null)
const createdBatch = ref<Batch | null>(null)
const createdStatusCode = ref<number>(0)

const requireConfirmUnripe = computed(() => aggregateSummary.value.unripe_ratio > 0.15)

async function handleStartRecognition() {
  submitError.value = null

  if (!camera.currentStream.value) {
    await camera.startCamera(camera.selectedDeviceId.value)
  }

  if (camera.cameraError.value) {
    return
  }

  await startStream()
}

async function handleStopRecognition() {
  await stopStream()
}

async function handleRefreshDevices() {
  await camera.refreshDevices()
}

async function handleSwitchDevice(deviceId: string) {
  await camera.switchCamera(deviceId)
}

function handleVideoElementChange(video: HTMLVideoElement | null) {
  videoElement.value = video
}

async function handleCreateBatch(formInput: BatchCreateFormInput) {
  submitError.value = null

  if (isStreaming.value) {
    await stopStream()
  }

  const summaryInput = toBatchSummaryInput(aggregateSummary.value)
  const summaryError = validateBatchSummaryInput(summaryInput)
  if (summaryError) {
    submitError.value = {
      statusCode: 400,
      error: 'invalid_summary',
      message: summaryError
    }
    return
  }

  const payload = buildBatchCreateRequest(formInput, summaryInput)

  submitting.value = true
  try {
    const result = await createBatch(payload)
    createdBatch.value = result.data
    createdStatusCode.value = result.statusCode
  } catch (error) {
    submitError.value = parseCreateError(error)
  } finally {
    submitting.value = false
  }
}

function continueCreate() {
  createdBatch.value = null
  createdStatusCode.value = 0
  submitError.value = null
  resetSession()
}

onBeforeUnmount(() => {
  void stopStream()
  camera.stopCamera()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          Batch Builder
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          识别建批页
        </h1>
        <p class="text-sm text-toned sm:text-base">
          先进行实时识别并汇总成熟度，再提交创建采摘批次并获取溯源码。
        </p>
      </section>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-[1.15fr_1fr]">
        <div class="space-y-6">
          <BatchCameraStage
            :devices="camera.options.value"
            :selected-device-id="camera.selectedDeviceId.value"
            :current-frame="lastFrame"
            :is-recognizing="isStreaming"
            :camera-loading="camera.isCameraLoading.value"
            :camera-error="camera.cameraError.value"
            :stream-error="streamError"
            @update:video-element="handleVideoElementChange"
            @update:selected-device-id="handleSwitchDevice"
            @start="handleStartRecognition"
            @stop="handleStopRecognition"
            @refresh="handleRefreshDevices"
          />

          <BatchLiveSummaryPanel
            :summary="aggregateSummary"
            :current-frame="lastFrame"
            :server-summary="serverSummary"
            :is-recognizing="isStreaming"
          />
        </div>

        <div class="space-y-6">
          <BatchCreateForm
            v-if="!createdBatch"
            :summary="aggregateSummary"
            :submitting="submitting"
            :is-recognizing="isStreaming"
            :require-confirm-unripe="requireConfirmUnripe"
            :api-error="submitError"
            @submit="handleCreateBatch"
          />

          <BatchCreateResult
            v-else
            :batch="createdBatch"
            :status-code="createdStatusCode"
            @continue="continueCreate"
          />
        </div>
      </div>
    </div>
  </UContainer>
</template>
