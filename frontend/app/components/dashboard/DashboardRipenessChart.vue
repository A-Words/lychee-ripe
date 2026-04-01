<script setup lang="ts">
import { buildRipenessBarOption } from '~/utils/dashboard-chart-options'
import type { RipenessDistribution } from '~/types/dashboard'

const props = defineProps<{
  ripenessDistribution: RipenessDistribution
}>()

const option = computed(() => buildRipenessBarOption(props.ripenessDistribution))
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div>
        <h3 class="text-base font-semibold text-highlighted">
          成熟度分布
        </h3>
        <p class="mt-1 text-xs text-muted">
          green / half / red / young
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
