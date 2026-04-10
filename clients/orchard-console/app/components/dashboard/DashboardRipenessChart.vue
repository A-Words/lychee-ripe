<script setup lang="ts">
import { RIPENESS_CLASSES, RIPENESS_LABEL_MAP } from '~/constants/ripeness'
import { buildRipenessBarOption } from '~/utils/dashboard-chart-options'
import type { RipenessDistribution } from '~/types/dashboard'

const props = defineProps<{
  ripenessDistribution: RipenessDistribution
}>()

const option = computed(() => buildRipenessBarOption(props.ripenessDistribution))
const ripenessSummary = computed(() =>
  RIPENESS_CLASSES.map((key) => RIPENESS_LABEL_MAP[key]).join(' / ')
)
const ripenessBreakdown = computed(() =>
  RIPENESS_CLASSES.map((key) => ({
    key,
    label: RIPENESS_LABEL_MAP[key],
    value: props.ripenessDistribution[key]
  }))
)
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div>
        <h3 class="text-base font-semibold text-highlighted">
          成熟度分布
        </h3>
        <p class="mt-1 text-xs text-muted">
          {{ ripenessSummary }}
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

    <div class="mt-4 flex flex-wrap gap-2 text-xs text-toned">
      <span
        v-for="item in ripenessBreakdown"
        :key="item.key"
        class="rounded-full bg-muted px-2.5 py-1"
      >
        {{ item.label }} {{ item.value }}
      </span>
    </div>
  </UCard>
</template>
