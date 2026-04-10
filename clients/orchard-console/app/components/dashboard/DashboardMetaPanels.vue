<script setup lang="ts">
import type { ReconcileStats, UnripeMetrics } from '~/types/dashboard'
import { getUnripeHandlingLabel } from '~/constants/unripe-handling'
import { formatDateTime, formatPercent } from '~/utils/dashboard-format'

const props = defineProps<{
  unripeMetrics: UnripeMetrics
  reconcileStats?: ReconcileStats | null
}>()

const unripePercent = computed(() => formatPercent(props.unripeMetrics.unripe_batch_ratio))
const thresholdPercent = computed(() => formatPercent(props.unripeMetrics.threshold))
const handlingLabel = computed(() => getUnripeHandlingLabel(props.unripeMetrics.unripe_handling))
const progressWidth = computed(() =>
  Math.min(Math.max(props.unripeMetrics.unripe_batch_ratio * 100, 0), 100)
)
</script>

<template>
  <div class="grid grid-cols-1 gap-4" :class="props.reconcileStats ? 'lg:grid-cols-2' : ''">
    <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
      <template #header>
        <div>
          <h3 class="text-base font-semibold text-highlighted">
            未成熟指标
          </h3>
          <p class="mt-1 text-xs text-muted">
            阈值 {{ thresholdPercent }}，处理策略 {{ handlingLabel }}
          </p>
        </div>
      </template>

      <div class="space-y-3">
        <div class="flex items-center justify-between text-sm">
          <span class="text-toned">未成熟批次占比</span>
          <span class="font-semibold text-warning">
            {{ unripePercent }}
          </span>
        </div>

        <div class="h-2 overflow-hidden rounded-full bg-elevated">
          <div
            class="h-full rounded-full bg-warning transition-all"
            :style="{ width: `${progressWidth}%` }"
          />
        </div>

        <p class="text-sm text-default">
          未成熟批次数：{{ unripeMetrics.unripe_batch_count }}
        </p>
      </div>
    </UCard>

    <UCard v-if="props.reconcileStats" variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
      <template #header>
        <div>
          <h3 class="text-base font-semibold text-highlighted">
            补链统计
          </h3>
          <p class="mt-1 text-xs text-muted">
            最近一次补链：{{ formatDateTime(reconcileStats?.last_reconcile_at ?? null) }}
          </p>
        </div>
      </template>

      <div class="grid grid-cols-3 gap-2 text-sm">
        <UCard variant="subtle" :ui="{ body: 'px-3 py-3' }">
          <p class="text-xs text-muted">
            待处理
          </p>
          <p class="mt-1 text-lg font-semibold text-default">
            {{ reconcileStats?.pending_count ?? 0 }}
          </p>
        </UCard>
        <UCard variant="subtle" :ui="{ body: 'px-3 py-3' }">
          <p class="text-xs text-muted">
            重试总数
          </p>
          <p class="mt-1 text-lg font-semibold text-default">
            {{ reconcileStats?.retried_total ?? 0 }}
          </p>
        </UCard>
        <UCard variant="subtle" :ui="{ body: 'px-3 py-3' }">
          <p class="text-xs text-muted">
            失败总数
          </p>
          <p class="mt-1 text-lg font-semibold text-default">
            {{ reconcileStats?.failed_total ?? 0 }}
          </p>
        </UCard>
      </div>
    </UCard>
  </div>
</template>
