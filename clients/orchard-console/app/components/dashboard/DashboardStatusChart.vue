<script setup lang="ts">
import { DASHBOARD_STATUS_META, getDashboardStatusOrder } from '~/constants/dashboard-status'
import { buildStatusDonutOption } from '~/utils/dashboard-chart-options'
import type { BatchStatusDistribution } from '~/types/dashboard'
import type { TraceMode } from '~/types/trace'

const props = defineProps<{
  traceMode: TraceMode
  statusDistribution: BatchStatusDistribution
}>()

const option = computed(() => buildStatusDonutOption(props.traceMode, props.statusDistribution))
const statusSummary = computed(() =>
  getDashboardStatusOrder(props.traceMode)
    .map((key) => DASHBOARD_STATUS_META[key].label)
    .join(' / ')
)
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div>
        <h3 class="text-base font-semibold text-highlighted">
          批次状态分布
        </h3>
        <p class="mt-1 text-xs text-muted">
          {{ statusSummary }}
        </p>
      </div>
    </template>

    <ClientOnly>
      <div class="h-72 w-full">
        <VChart :option="option" autoresize class="h-full w-full" />
      </div>
      <template #fallback>
        <USkeleton class="h-72 w-full" />
      </template>
    </ClientOnly>
  </UCard>
</template>
