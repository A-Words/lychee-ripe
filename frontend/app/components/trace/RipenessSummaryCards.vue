<script setup lang="ts">
import { RIPENESS_CLASSES, RIPENESS_COLOR_MAP, RIPENESS_LABEL_MAP } from '~/constants/ripeness'
import type { BatchSummary, RipenessLabel } from '~/types/trace'

const props = defineProps<{
  summary: BatchSummary
}>()

const items = computed(() =>
  RIPENESS_CLASSES.map((key) => {
    const count = props.summary[key]
    const ratio = props.summary.total > 0 ? count / props.summary.total : 0
    return {
      key,
      label: RIPENESS_LABEL_MAP[key],
      count,
      ratio,
      color: RIPENESS_COLOR_MAP[key]
    }
  })
)

function ratioText(value: number): string {
  return `${(value * 100).toFixed(1)}%`
}

function itemTextColor(key: RipenessLabel): string {
  return key === 'half' ? 'text-gray-900 dark:text-gray-100' : ''
}
</script>

<template>
  <div class="space-y-4">
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <UCard
        v-for="item in items"
        :key="item.key"
        variant="outline"
        :ui="{
          body: 'p-4'
        }"
        class="relative overflow-hidden"
      >
        <div
          class="absolute left-0 top-0 h-full w-1.5"
          :style="{ backgroundColor: item.color }"
        />
        <div class="space-y-1 pl-2">
          <p class="text-sm text-toned">
            {{ item.label }}
          </p>
          <p class="text-2xl font-semibold" :class="itemTextColor(item.key)">
            {{ item.count }}
          </p>
          <p class="text-xs text-muted">
            占比 {{ ratioText(item.ratio) }}
          </p>
        </div>
      </UCard>
    </div>

    <UCard variant="subtle" :ui="{ body: 'p-4' }">
      <div class="space-y-2">
        <div class="flex items-center justify-between text-sm">
          <span class="text-toned">未成熟占比（green + young）</span>
          <span class="font-semibold text-warning">
            {{ ratioText(summary.unripe_ratio) }}
          </span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-elevated">
          <div
            class="h-full rounded-full bg-warning transition-all"
            :style="{ width: `${Math.min(summary.unripe_ratio * 100, 100)}%` }"
          />
        </div>
        <p class="text-xs text-muted">
          未成熟数量 {{ summary.unripe_count }}，处理策略 {{ summary.unripe_handling }}
        </p>
      </div>
    </UCard>
  </div>
</template>
