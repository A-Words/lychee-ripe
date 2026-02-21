<script setup lang="ts">
import { computed } from 'vue'
import type { FrameSummary, SessionSummary } from '../types/infer'

interface Props {
  streamStatus: string
  cameraStatus: string
  cameraLabel: string | null
  frameSummary: FrameSummary | null
  sessionSummary: SessionSummary | null
  lastError: string | null
}

const props = defineProps<Props>()

const streamColor = computed(() => {
  if (props.streamStatus === 'streaming') {
    return 'success'
  }
  if (props.streamStatus === 'error') {
    return 'error'
  }
  if (props.streamStatus === 'connecting' || props.streamStatus === 'stopping') {
    return 'warning'
  }
  return 'neutral'
})

const cameraColor = computed(() => {
  if (props.cameraStatus === 'ready') {
    return 'success'
  }
  if (props.cameraStatus === 'error') {
    return 'error'
  }
  if (props.cameraStatus === 'requesting') {
    return 'warning'
  }
  return 'neutral'
})
</script>

<template>
  <UCard class="lr-panel">
    <template #header>
      <div class="flex items-center justify-between gap-2">
        <h2 class="text-base font-semibold">Session Status</h2>
      </div>
    </template>

    <div class="space-y-3 text-sm">
      <div class="flex items-center justify-between gap-2">
        <span>Stream</span>
        <UBadge :color="streamColor" variant="subtle">{{ streamStatus }}</UBadge>
      </div>
      <div class="flex items-center justify-between gap-2">
        <span>Camera</span>
        <UBadge :color="cameraColor" variant="subtle">{{ cameraStatus }}</UBadge>
      </div>
      <div class="flex items-center justify-between gap-2">
        <span>Selected Camera</span>
        <span class="truncate text-right text-neutral-600">{{ cameraLabel ?? 'N/A' }}</span>
      </div>
    </div>

    <hr class="my-4 border-black/10">

    <div class="space-y-2 text-sm">
      <h3 class="font-semibold">Current Frame</h3>
      <template v-if="frameSummary">
        <div class="grid grid-cols-2 gap-2">
          <span>Total</span><span class="text-right">{{ frameSummary.total }}</span>
          <span>Green</span><span class="text-right">{{ frameSummary.green }}</span>
          <span>Half</span><span class="text-right">{{ frameSummary.half }}</span>
          <span>Red</span><span class="text-right">{{ frameSummary.red }}</span>
          <span>Young</span><span class="text-right">{{ frameSummary.young }}</span>
        </div>
      </template>
      <p v-else class="text-neutral-500">No frame result yet.</p>
    </div>

    <hr class="my-4 border-black/10">

    <div class="space-y-2 text-sm">
      <h3 class="font-semibold">Session Summary</h3>
      <template v-if="sessionSummary">
        <div class="grid grid-cols-2 gap-2">
          <span>Total Detected</span><span class="text-right">{{ sessionSummary.total_detected }}</span>
          <span>Green Ratio</span><span class="text-right">{{ (sessionSummary.ripeness_ratio.green * 100).toFixed(1) }}%</span>
          <span>Half Ratio</span><span class="text-right">{{ (sessionSummary.ripeness_ratio.half * 100).toFixed(1) }}%</span>
          <span>Red Ratio</span><span class="text-right">{{ (sessionSummary.ripeness_ratio.red * 100).toFixed(1) }}%</span>
          <span>Young Ratio</span><span class="text-right">{{ (sessionSummary.ripeness_ratio.young * 100).toFixed(1) }}%</span>
          <span>Harvest Suggestion</span><span class="text-right">{{ sessionSummary.harvest_suggestion }}</span>
        </div>
      </template>
      <p v-else class="text-neutral-500">No summary yet.</p>
    </div>

    <UAlert
      v-if="lastError"
      class="mt-4"
      color="error"
      variant="soft"
      title="Stream warning"
      :description="lastError"
    />
  </UCard>
</template>
