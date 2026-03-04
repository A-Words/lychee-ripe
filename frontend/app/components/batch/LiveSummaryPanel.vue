<script setup lang="ts">
import { HARVEST_SUGGESTION_META } from '~/constants/harvest-suggestion'
import type { FrameResult, SessionSummary } from '~/types/infer'
import type { SessionAggregateSummary } from '~/utils/session-aggregator'

const props = defineProps<{
  summary: SessionAggregateSummary
  currentFrame: FrameResult | null
  serverSummary: SessionSummary | null
  isRecognizing: boolean
}>()

const suggestionMeta = computed(() => HARVEST_SUGGESTION_META[props.summary.harvest_suggestion])

const batchLikeSummary = computed(() => ({
  total: props.summary.total,
  green: props.summary.green,
  half: props.summary.half,
  red: props.summary.red,
  young: props.summary.young,
  unripe_count: props.summary.unripe_count,
  unripe_ratio: props.summary.unripe_ratio,
  unripe_handling: 'sorted_out' as const
}))

function ratioText(value: number): string {
  return `${(value * 100).toFixed(1)}%`
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            会话汇总
          </h2>
          <p class="mt-1 text-xs text-muted">
            当前会话累计检测 {{ summary.total }} 颗荔枝
          </p>
        </div>
        <UBadge :color="isRecognizing ? 'success' : 'neutral'" variant="soft">
          {{ isRecognizing ? '实时更新中' : '已暂停更新' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        :color="suggestionMeta.color"
        variant="subtle"
        icon="i-lucide-leaf"
        :title="suggestionMeta.label"
      >
        <template #description>
          <p class="text-sm">
            {{ suggestionMeta.description }}
          </p>
          <p class="mt-1 text-sm text-muted">
            未成熟占比 {{ ratioText(summary.unripe_ratio) }}
          </p>
        </template>
      </UAlert>

      <RipenessSummaryCards :summary="batchLikeSummary" />

      <UCard
        v-if="currentFrame"
        variant="subtle"
        :ui="{ body: 'px-4 py-3' }"
      >
        <p class="text-xs text-muted">
          当前帧汇总（frame #{{ currentFrame.frame_index }})
        </p>
        <div class="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-sm sm:grid-cols-5">
          <p>总数：{{ currentFrame.frame_summary.total }}</p>
          <p>青果：{{ currentFrame.frame_summary.green }}</p>
          <p>半熟：{{ currentFrame.frame_summary.half }}</p>
          <p>红果：{{ currentFrame.frame_summary.red }}</p>
          <p>嫩果：{{ currentFrame.frame_summary.young }}</p>
        </div>
      </UCard>

      <UCard
        v-if="serverSummary"
        variant="subtle"
        :ui="{ body: 'px-4 py-3' }"
      >
        <p class="text-xs text-muted">
          服务端会话总量（summary 事件）
        </p>
        <p class="mt-1 text-sm text-default">
          total_detected = {{ serverSummary.total_detected }}
        </p>
      </UCard>
    </div>
  </UCard>
</template>
