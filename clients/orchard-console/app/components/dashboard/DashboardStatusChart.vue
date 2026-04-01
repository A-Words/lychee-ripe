<script setup lang="ts">
import { buildStatusDonutOption } from '~/utils/dashboard-chart-options'
import type { BatchStatusDistribution } from '~/types/dashboard'

const props = defineProps<{
  statusDistribution: BatchStatusDistribution
}>()

const option = computed(() => buildStatusDonutOption(props.statusDistribution))
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div>
        <h3 class="text-base font-semibold text-highlighted">
          批次状态分布
        </h3>
        <p class="mt-1 text-xs text-muted">
          anchored / pending_anchor / anchor_failed
        </p>
      </div>
    </template>

    <ClientOnly>
      <VChart :option="option" autoresize class="h-72 w-full" />
      <template #fallback>
        <USkeleton class="h-72 w-full" />
      </template>
    </ClientOnly>
  </UCard>
</template>
