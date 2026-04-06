<script setup lang="ts">
import type { BatchStatusDistribution, DashboardTotals } from '~/types/dashboard'
import type { TraceMode } from '~/types/trace'

const props = defineProps<{
  traceMode: TraceMode
  totals: DashboardTotals
  statusDistribution: BatchStatusDistribution
}>()

const cards = computed(() => {
  const items: Array<{
    key: string
    label: string
    value: number
    icon: string
    color: 'primary' | 'success' | 'warning' | 'error'
  }> = [
    {
      key: 'batch_total',
      label: '批次总数',
      value: props.totals.batch_total,
      icon: 'i-lucide-package-open',
      color: 'primary' as const
    }
  ]

  if (props.traceMode === 'database') {
    items.push({
      key: 'stored',
      label: '已入库',
      value: props.statusDistribution.stored ?? 0,
      icon: 'i-lucide-database',
      color: 'primary' as const
    })
    return items
  }

  items.push(
    {
      key: 'anchored',
      label: '已上链',
      value: props.statusDistribution.anchored ?? 0,
      icon: 'i-lucide-shield-check',
      color: 'success' as const
    },
    {
      key: 'pending_anchor',
      label: '待补链',
      value: props.statusDistribution.pending_anchor ?? 0,
      icon: 'i-lucide-clock-alert',
      color: 'warning' as const
    },
    {
      key: 'anchor_failed',
      label: '补链失败',
      value: props.statusDistribution.anchor_failed ?? 0,
      icon: 'i-lucide-shield-x',
      color: 'error' as const
    }
  )
  return items
})
</script>

<template>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
    <UCard
      v-for="card in cards"
      :key="card.key"
      variant="outline"
      :ui="{ body: 'p-4 sm:p-5' }"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="space-y-1">
          <p class="text-xs uppercase tracking-wider text-muted">
            {{ card.label }}
          </p>
          <p class="text-2xl font-semibold text-highlighted">
            {{ card.value }}
          </p>
        </div>

        <UBadge :color="card.color" variant="soft">
          <UIcon :name="card.icon" class="size-4" />
        </UBadge>
      </div>
    </UCard>
  </div>
</template>
