<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import { toRFC3339FromLocal } from '~/composables/useBatchCreate'
import type { BatchCreateApiError, BatchCreateFormInput } from '~/types/batch'
import type { OrchardWithPlots } from '~/types/resources'
import type { SessionAggregateSummary } from '~/utils/session-aggregator'

interface FormState {
  orchardPresetId?: string
  orchard_id: string
  orchard_name: string
  plotPresetId?: string
  plot_id: string
  plot_name: string
  harvested_at: string
  note: string
  confirm_unripe: boolean
}

const props = defineProps<{
  orchards: OrchardWithPlots[]
  summary: SessionAggregateSummary
  submitting: boolean
  isRecognizing: boolean
  requireConfirmUnripe: boolean
  apiError: BatchCreateApiError | null
}>()

const emit = defineEmits<{
  (event: 'submit', payload: BatchCreateFormInput): void
}>()

const initialOrchard = computed(() => props.orchards[0])
const initialPlot = computed(() => initialOrchard.value?.plots[0])

const state = reactive<FormState>({
  orchardPresetId: initialOrchard.value?.orchard_id,
  orchard_id: initialOrchard.value?.orchard_id || '',
  orchard_name: initialOrchard.value?.orchard_name || '',
  plotPresetId: initialPlot.value?.plot_id,
  plot_id: initialPlot.value?.plot_id || '',
  plot_name: initialPlot.value?.plot_name || '',
  harvested_at: getLocalDateTimeNow(),
  note: '',
  confirm_unripe: false
})

const formError = ref('')

const orchardItems = computed(() =>
  props.orchards.map((orchard) => ({
    label: orchard.orchard_name,
    value: orchard.orchard_id
  }))
)

const selectedPreset = computed(() =>
  props.orchards.find((orchard) => orchard.orchard_id === state.orchardPresetId)
)

const plotItems = computed(() =>
  (selectedPreset.value?.plots || []).map((plot) => ({
    label: plot.plot_name,
    value: plot.plot_id
  }))
)

const submitDisabled = computed(() =>
  props.submitting || props.isRecognizing || props.summary.total <= 0
)

watch(() => state.orchardPresetId, (value) => {
  const preset = props.orchards.find((orchard) => orchard.orchard_id === value)
  if (!preset) {
    return
  }

  state.orchard_id = preset.orchard_id
  state.orchard_name = preset.orchard_name

  const firstPlot = preset.plots[0]
  state.plotPresetId = firstPlot?.plot_id
  if (firstPlot) {
    state.plot_id = firstPlot.plot_id
    state.plot_name = firstPlot.plot_name
    return
  }

  state.plot_id = ''
  state.plot_name = ''
})

watch(() => state.plotPresetId, (value) => {
  const currentPlot = selectedPreset.value?.plots.find((plot) => plot.plot_id === value)
  if (!currentPlot) {
    return
  }
  state.plot_id = currentPlot.plot_id
  state.plot_name = currentPlot.plot_name
})

watch(() => props.requireConfirmUnripe, (required) => {
  if (!required) {
    state.confirm_unripe = false
  }
})

watch(() => props.orchards, (orchards) => {
  syncPresetState(orchards)
}, { immediate: true })

function syncPresetState(orchards: OrchardWithPlots[]) {
  if (orchards.length === 0) {
    state.orchardPresetId = undefined
    state.orchard_id = ''
    state.orchard_name = ''
    state.plotPresetId = undefined
    state.plot_id = ''
    state.plot_name = ''
    return
  }

  const orchard = orchards.find((item) => item.orchard_id === state.orchardPresetId) ?? orchards[0]
  if (!orchard) {
    state.orchardPresetId = undefined
    state.orchard_id = ''
    state.orchard_name = ''
    state.plotPresetId = undefined
    state.plot_id = ''
    state.plot_name = ''
    return
  }
  state.orchardPresetId = orchard.orchard_id
  state.orchard_id = orchard.orchard_id
  state.orchard_name = orchard.orchard_name

  const plot = orchard.plots.find((item) => item.plot_id === state.plotPresetId) ?? orchard.plots[0]
  state.plotPresetId = plot?.plot_id
  state.plot_id = plot?.plot_id || ''
  state.plot_name = plot?.plot_name || ''
}

function validate(current: FormState): FormError[] {
  const errors: FormError[] = []

  if (!current.orchard_id.trim()) {
    errors.push({ name: 'orchard_id', message: '请输入果园标识。' })
  }
  if (!current.orchard_name.trim()) {
    errors.push({ name: 'orchard_name', message: '请输入果园名称。' })
  }
  if (!current.plot_id.trim()) {
    errors.push({ name: 'plot_id', message: '请输入地块标识。' })
  }
  if (!current.harvested_at.trim()) {
    errors.push({ name: 'harvested_at', message: '请选择采摘时间。' })
  }
  if (props.summary.total <= 0) {
    errors.push({ name: 'summary', message: '当前会话无有效汇总，无法建批。' })
  }
  if (props.requireConfirmUnripe && !current.confirm_unripe) {
    errors.push({ name: 'confirm_unripe', message: '未成熟占比超阈值，请先确认。' })
  }

  return errors
}

function onSubmit(event: FormSubmitEvent<FormState>) {
  formError.value = ''

  const harvestedAt = toRFC3339FromLocal(event.data.harvested_at)
  if (!harvestedAt) {
    formError.value = '采摘时间格式不正确，请重新选择。'
    return
  }

  emit('submit', {
    orchard_id: event.data.orchard_id.trim(),
    orchard_name: event.data.orchard_name.trim(),
    plot_id: event.data.plot_id.trim(),
    plot_name: event.data.plot_name.trim() || undefined,
    harvested_at: harvestedAt,
    note: event.data.note.trim() || undefined,
    confirm_unripe: event.data.confirm_unripe
  })
}

function getLocalDateTimeNow(): string {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')

  const year = now.getFullYear()
  const month = pad(now.getMonth() + 1)
  const day = pad(now.getDate())
  const hour = pad(now.getHours())
  const minute = pad(now.getMinutes())
  return `${year}-${month}-${day}T${hour}:${minute}`
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div>
        <h2 class="text-base font-semibold text-highlighted">
          建批信息
        </h2>
        <p class="mt-1 text-xs text-muted">
          汇总数据来自当前识别会话，提交后触发锚定流程。
        </p>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        v-if="isRecognizing"
        color="warning"
        variant="subtle"
        icon="i-lucide-pause-circle"
        title="请先停止识别"
        description="识别停止后汇总值将冻结，可用于建批提交。"
      />

      <UAlert
        v-if="apiError"
        color="error"
        variant="subtle"
        icon="i-lucide-alert-circle"
        title="建批失败"
      >
        <template #description>
          <p class="text-sm">
            {{ apiError.message }}
          </p>
          <p v-if="apiError.requestId" class="mt-1 text-xs text-muted">
            请求 ID：{{ apiError.requestId }}
          </p>
        </template>
      </UAlert>

      <UAlert
        v-if="formError"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        :description="formError"
      />

      <UForm :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <UFormField label="预置果园" name="orchardPresetId">
            <USelect
              v-model="state.orchardPresetId"
              :items="orchardItems"
              value-key="value"
              label-key="label"
              icon="i-lucide-trees"
              placeholder="选择预置果园"
            />
          </UFormField>

          <UFormField label="预置地块" name="plotPresetId">
            <USelect
              v-model="state.plotPresetId"
              :items="plotItems"
              value-key="value"
              label-key="label"
              icon="i-lucide-map"
              placeholder="选择预置地块"
            />
          </UFormField>

          <UFormField label="果园标识" name="orchard_id" required>
            <UInput v-model="state.orchard_id" icon="i-lucide-fingerprint" />
          </UFormField>

          <UFormField label="果园名称" name="orchard_name" required>
            <UInput v-model="state.orchard_name" icon="i-lucide-trees" />
          </UFormField>

          <UFormField label="地块标识" name="plot_id" required>
            <UInput v-model="state.plot_id" icon="i-lucide-map-pinned" />
          </UFormField>

          <UFormField label="地块名称（可选）" name="plot_name">
            <UInput v-model="state.plot_name" icon="i-lucide-map" />
          </UFormField>

          <UFormField label="采摘时间" name="harvested_at" required>
            <UInput v-model="state.harvested_at" type="datetime-local" icon="i-lucide-calendar-clock" />
          </UFormField>
        </div>

        <UFormField label="备注（可选）" name="note">
          <UTextarea v-model="state.note" :rows="3" placeholder="例如：首批果园采摘批次" />
        </UFormField>

        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-5">
            <p>总数：<span class="font-semibold">{{ summary.total }}</span></p>
            <p>青果：<span class="font-semibold">{{ summary.green }}</span></p>
            <p>半熟：<span class="font-semibold">{{ summary.half }}</span></p>
            <p>红果：<span class="font-semibold">{{ summary.red }}</span></p>
            <p>嫩果：<span class="font-semibold">{{ summary.young }}</span></p>
          </div>
        </UCard>

        <UFormField
          v-if="requireConfirmUnripe"
          name="confirm_unripe"
          description="未成熟占比超过 15%，提交前需进行确认。"
        >
          <UCheckbox
            v-model="state.confirm_unripe"
            label="我已确认未成熟占比较高，允许继续建批。"
          />
        </UFormField>

        <div class="flex flex-wrap items-center gap-3">
          <UButton
            type="submit"
            icon="i-lucide-package-plus"
            :loading="submitting"
            :disabled="submitDisabled"
            label="创建采摘批次"
          />
          <p class="text-xs text-muted">
            返回状态 201=上链成功，202=已保存待补链。
          </p>
        </div>
      </UForm>
    </div>
  </UCard>
</template>
