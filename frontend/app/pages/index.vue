<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import CameraStage from '../components/CameraStage.vue'
import StatusPanel from '../components/StatusPanel.vue'
import {
  computeUnripeMetrics,
  DEFAULT_UNRIPE_THRESHOLD,
  getBatchStatusNotice,
  needsUnripeConfirm,
  useBatchCreate,
} from '../composables/useBatchCreate'
import { useCamera } from '../composables/useCamera'
import { useInferenceStream } from '../composables/useInferenceStream'
import { buildTracePath } from '../constants/navigation'
import type { BatchSummaryInput } from '../types/batch'

const TARGET_WIDTH = 640
const TARGET_HEIGHT = 360
const TARGET_FPS = 5
const JPEG_QUALITY = 0.8
const CAMERA_STORAGE_KEY = 'lychee-ripe.camera.deviceId'

interface PlotPreset {
  id: string
  name: string
}

interface OrchardPreset {
  id: string
  name: string
  plots: PlotPreset[]
}

const ORCHARD_PRESETS: OrchardPreset[] = [
  {
    id: 'orchard_zc_xc',
    name: '增城仙村果园',
    plots: [
      { id: 'plot_a01', name: 'A-01' },
      { id: 'plot_a07', name: 'A-07' },
    ],
  },
  {
    id: 'orchard_mm_gg',
    name: '茂名高州果园',
    plots: [
      { id: 'plot_b03', name: 'B-03' },
      { id: 'plot_b09', name: 'B-09' },
    ],
  },
]

const runtimeConfig = useRuntimeConfig()
const gatewayBase = runtimeConfig.public.gatewayBase

const toast = useToast()

const surfaceVideo = ref<HTMLVideoElement | null>(null)
const isSwitchingCamera = ref(false)

const camera = useCamera({ width: TARGET_WIDTH, height: TARGET_HEIGHT })
const stream = useInferenceStream({ gatewayBase })
const batchCreator = useBatchCreate({ gatewayBase })

const initialOrchard = ORCHARD_PRESETS[0]
const initialPlot = initialOrchard?.plots[0]

const createForm = reactive({
  orchardPresetId: initialOrchard?.id ?? '',
  orchardId: initialOrchard?.id ?? '',
  orchardName: initialOrchard?.name ?? '',
  plotPresetId: initialPlot?.id ?? '',
  plotId: initialPlot?.id ?? '',
  plotName: initialPlot?.name ?? '',
  harvestedAtLocal: toDateTimeLocal(new Date()),
  note: '',
  confirmUnripe: false,
})

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

const selectedOrchardPreset = computed(() =>
  ORCHARD_PRESETS.find(item => item.id === createForm.orchardPresetId) ?? null,
)

const availablePlotPresets = computed(() => selectedOrchardPreset.value?.plots ?? [])

const orchardPresetOptions = ORCHARD_PRESETS.map(orchard => ({
  id: orchard.id,
  label: `${orchard.name} (${orchard.id})`,
}))

const plotPresetOptions = computed(() =>
  availablePlotPresets.value.map(plot => ({
    id: plot.id,
    label: `${plot.name} (${plot.id})`,
  })),
)

const liveSummary = computed<BatchSummaryInput>(() => {
  if (!frameSummary.value) {
    return {
      total: 0,
      green: 0,
      half: 0,
      red: 0,
      young: 0,
    }
  }
  return {
    total: frameSummary.value.total,
    green: frameSummary.value.green,
    half: frameSummary.value.half,
    red: frameSummary.value.red,
    young: frameSummary.value.young,
  }
})

const unripeMetrics = computed(() => computeUnripeMetrics(liveSummary.value))
const requiresUnripeConfirm = computed(() =>
  needsUnripeConfirm(liveSummary.value, DEFAULT_UNRIPE_THRESHOLD),
)
const unripeRatioPercent = computed(() => `${(unripeMetrics.value.unripeRatio * 100).toFixed(1)}%`)

const summaryItems = computed(() => [
  { key: 'total', label: 'total', value: liveSummary.value.total, highlight: false },
  { key: 'green', label: 'green', value: liveSummary.value.green, highlight: true },
  { key: 'half', label: 'half', value: liveSummary.value.half, highlight: false },
  { key: 'red', label: 'red', value: liveSummary.value.red, highlight: false },
  { key: 'young', label: 'young', value: liveSummary.value.young, highlight: true },
])

const createBatchStatus = computed(() => batchCreator.status.value)
const latestBatchResult = computed(() => batchCreator.lastResult.value)
const latestBatch = computed(() => latestBatchResult.value?.batch ?? null)
const latestBatchNotice = computed(() =>
  latestBatch.value ? getBatchStatusNotice(latestBatch.value.status) : null,
)
const latestBatchError = computed(() => batchCreator.lastError.value?.message ?? null)

const tracePath = computed(() =>
  latestBatch.value ? buildTracePath(latestBatch.value.trace_code) : '',
)

const traceURL = computed(() => {
  if (!latestBatch.value) {
    return ''
  }
  if (!import.meta.client) {
    return tracePath.value
  }
  return new URL(tracePath.value, window.location.origin).toString()
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

function toDateTimeLocal(value: Date): string {
  const local = new Date(value.getTime() - value.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function localDateTimeToRFC3339(value: string): string | null {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }
  return parsed.toISOString()
}

const isStreaming = computed(() => streamStatus.value === 'streaming')
const isSubmittingBatch = computed(() => createBatchStatus.value === 'submitting')

const hasCreateFields = computed(() =>
  createForm.orchardId.trim() !== '' &&
  createForm.orchardName.trim() !== '' &&
  createForm.plotId.trim() !== '' &&
  createForm.harvestedAtLocal.trim() !== '',
)

const canSubmitBatch = computed(() => {
  if (isSubmittingBatch.value) {
    return false
  }
  if (liveSummary.value.total <= 0 || !hasCreateFields.value) {
    return false
  }
  if (!localDateTimeToRFC3339(createForm.harvestedAtLocal)) {
    return false
  }
  if (requiresUnripeConfirm.value && !createForm.confirmUnripe) {
    return false
  }
  return true
})

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

function onOrchardPresetSelected(value: unknown) {
  if (typeof value !== 'string') {
    return
  }
  createForm.orchardPresetId = value
}

function onPlotPresetSelected(value: unknown) {
  if (typeof value !== 'string') {
    return
  }
  createForm.plotPresetId = value
}

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

async function submitBatchCreate() {
  if (liveSummary.value.total <= 0) {
    toast.add({
      color: 'warning',
      title: '无法建批',
      description: '当前没有有效识别汇总，请先开始识别。',
    })
    return
  }

  if (requiresUnripeConfirm.value && !createForm.confirmUnripe) {
    toast.add({
      color: 'warning',
      title: '需要二次确认',
      description: '未成熟占比超过阈值，请勾选确认后再提交。',
    })
    return
  }

  const harvestedAt = localDateTimeToRFC3339(createForm.harvestedAtLocal)
  if (!harvestedAt) {
    toast.add({
      color: 'error',
      title: '时间格式错误',
      description: '请填写有效的采摘时间。',
    })
    return
  }

  try {
    const result = await batchCreator.createBatch({
      orchard_id: createForm.orchardId.trim(),
      orchard_name: createForm.orchardName.trim(),
      plot_id: createForm.plotId.trim(),
      plot_name: createForm.plotName.trim() || undefined,
      harvested_at: harvestedAt,
      summary: liveSummary.value,
      note: createForm.note.trim() || undefined,
      confirm_unripe: createForm.confirmUnripe,
    })

    const statusText = result.batch.status === 'anchored' ? '已锚定' : '待补链'
    toast.add({
      color: result.batch.status === 'anchored' ? 'success' : 'warning',
      title: '批次创建成功',
      description: `${result.batch.trace_code} (${statusText})`,
    })
  } catch (err) {
    const description = err instanceof Error ? err.message : 'create batch failed'
    toast.add({
      color: 'error',
      title: '批次创建失败',
      description,
    })
  }
}

async function copyTraceLink() {
  if (!import.meta.client || !traceURL.value) {
    return
  }
  if (!navigator.clipboard) {
    toast.add({
      color: 'warning',
      title: '复制不可用',
      description: '当前环境不支持剪贴板接口。',
    })
    return
  }

  try {
    await navigator.clipboard.writeText(traceURL.value)
    toast.add({
      color: 'success',
      title: '已复制溯源链接',
      description: traceURL.value,
    })
  } catch (err) {
    const description = err instanceof Error ? err.message : 'copy failed'
    toast.add({
      color: 'error',
      title: '复制失败',
      description,
    })
  }
}

watch(
  () => createForm.orchardPresetId,
  (orchardID) => {
    const preset = ORCHARD_PRESETS.find(item => item.id === orchardID)
    if (!preset) {
      return
    }
    createForm.orchardId = preset.id
    createForm.orchardName = preset.name

    const firstPlot = preset.plots[0]
    if (firstPlot) {
      createForm.plotPresetId = firstPlot.id
      createForm.plotId = firstPlot.id
      createForm.plotName = firstPlot.name
    } else {
      createForm.plotPresetId = ''
      createForm.plotId = ''
      createForm.plotName = ''
    }
  },
)

watch(
  () => createForm.plotPresetId,
  (plotID) => {
    const preset = availablePlotPresets.value.find(item => item.id === plotID)
    if (!preset) {
      return
    }
    createForm.plotId = preset.id
    createForm.plotName = preset.name
  },
)

watch(
  () => requiresUnripeConfirm.value,
  (next) => {
    if (!next) {
      createForm.confirmUnripe = false
    }
  },
)

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

      <div class="space-y-3">
        <StatusPanel
          :stream-status="streamStatus"
          :camera-status="cameraStatus"
          :camera-label="cameraLabel"
          :frame-summary="frameSummary"
          :session-summary="sessionSummary"
          :last-error="streamError"
        />

        <UCard class="lr-panel">
          <template #header>
            <div class="space-y-1">
              <h2 class="text-base font-semibold">识别建批</h2>
              <p class="text-xs text-neutral-500">未成熟阈值：{{ DEFAULT_UNRIPE_THRESHOLD }}</p>
            </div>
          </template>

          <div class="space-y-3">
            <div class="grid gap-2 sm:grid-cols-2">
              <USelect
                :model-value="createForm.orchardPresetId || undefined"
                :items="orchardPresetOptions"
                value-key="id"
                label-key="label"
                placeholder="选择果园预设"
                @update:model-value="onOrchardPresetSelected"
              />
              <USelect
                :model-value="createForm.plotPresetId || undefined"
                :items="plotPresetOptions"
                value-key="id"
                label-key="label"
                placeholder="选择地块预设"
                @update:model-value="onPlotPresetSelected"
              />
            </div>

            <div class="grid gap-2 sm:grid-cols-2">
              <UInput v-model="createForm.orchardId" placeholder="orchard_id" />
              <UInput v-model="createForm.orchardName" placeholder="orchard_name" />
              <UInput v-model="createForm.plotId" placeholder="plot_id" />
              <UInput v-model="createForm.plotName" placeholder="plot_name (optional)" />
              <UInput v-model="createForm.harvestedAtLocal" type="datetime-local" />
            </div>

            <UTextarea
              v-model="createForm.note"
              :rows="2"
              placeholder="note (optional)"
            />

            <div class="space-y-2 rounded-md border border-neutral-200 p-3">
              <p class="text-sm font-medium">实时 summary</p>
              <div class="grid gap-2 sm:grid-cols-2">
                <div
                  v-for="item in summaryItems"
                  :key="item.key"
                  class="flex items-center justify-between rounded-md border px-2 py-1"
                  :class="item.highlight ? 'border-amber-300 bg-amber-50' : 'border-neutral-200 bg-white'"
                >
                  <span class="text-sm">{{ item.label }}</span>
                  <span class="font-semibold">{{ item.value }}</span>
                </div>
              </div>
              <p class="text-xs text-neutral-600">
                unripe_count={{ unripeMetrics.unripeCount }},
                unripe_ratio={{ unripeRatioPercent }},
                unripe_handling=sorted_out
              </p>
            </div>

            <UAlert
              v-if="requiresUnripeConfirm"
              color="warning"
              variant="soft"
              title="未成熟占比超阈值"
              :description="`当前 unripe_ratio=${unripeRatioPercent}，超过 ${DEFAULT_UNRIPE_THRESHOLD}，提交前需二次确认。`"
            />

            <label class="flex items-center gap-2 text-sm">
              <input
                v-model="createForm.confirmUnripe"
                type="checkbox"
                class="h-4 w-4"
              >
              <span>我确认该批次未成熟果比例偏高，仍继续创建</span>
            </label>

            <UButton
              icon="i-lucide-package-plus"
              :loading="isSubmittingBatch"
              :disabled="!canSubmitBatch"
              @click="submitBatchCreate"
            >
              创建批次
            </UButton>

            <UAlert
              v-if="latestBatchError"
              color="error"
              variant="soft"
              title="创建失败"
              :description="latestBatchError"
            />

            <UAlert
              v-if="latestBatchNotice"
              :color="latestBatchNotice.color"
              variant="soft"
              :title="latestBatchNotice.title"
              :description="latestBatchNotice.description"
            />

            <div
              v-if="latestBatch"
              class="space-y-2 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm"
            >
              <p>batch_id: <strong>{{ latestBatch.batch_id }}</strong></p>
              <p>trace_code: <strong>{{ latestBatch.trace_code }}</strong></p>
              <p>status: <strong>{{ latestBatch.status }}</strong></p>
              <p>http_status: <strong>{{ latestBatchResult?.statusCode }}</strong></p>
              <div class="flex flex-wrap items-center gap-2">
                <NuxtLink
                  :to="tracePath"
                  class="text-emerald-700 underline"
                >
                  打开溯源页
                </NuxtLink>
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-copy"
                  @click="copyTraceLink"
                >
                  复制溯源链接
                </UButton>
              </div>
            </div>
          </div>
        </UCard>
      </div>
    </div>
  </div>
</template>
