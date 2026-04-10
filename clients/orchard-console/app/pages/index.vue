<script setup lang="ts">
import { buildAppPath, inferAppBasePath } from '~/utils/app-path'

const route = useRoute()
const appBasePath = computed(() =>
  import.meta.client ? inferAppBasePath(window.location.pathname, route.path) : ''
)

const batchCreatePath = computed(() => buildAppPath(appBasePath.value, '/batch/create'))
const dashboardPath = computed(() => buildAppPath(appBasePath.value, '/dashboard'))
const tracePath = computed(() => buildAppPath(appBasePath.value, '/trace'))
</script>

<template>
  <UContainer class="py-10 sm:py-16">
    <div class="space-y-4">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          Lychee Ripe
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          区块链荔枝溯源与识别系统
        </h1>
      </section>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
          <div class="space-y-3">
            <h2 class="text-xl font-semibold text-highlighted">
              识别建批
            </h2>
            <p class="text-sm text-toned sm:text-base">
              调用实时识别流，形成会话汇总并提交采摘批次。
            </p>
            <div class="pt-2">
              <UButton :to="batchCreatePath" icon="i-lucide-camera" label="进入识别建批页" />
            </div>
          </div>
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
          <div class="space-y-3">
            <h2 class="text-xl font-semibold text-highlighted">
              数据看板
            </h2>
            <p class="text-sm text-toned sm:text-base">
              聚合查看批次状态、成熟度分布与当前存证模式。
            </p>
            <div class="pt-2">
              <UButton :to="dashboardPath" icon="i-lucide-chart-pie" label="进入数据看板" />
            </div>
          </div>
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
          <div class="space-y-3">
            <h2 class="text-xl font-semibold text-highlighted">
              公众溯源查询
            </h2>
            <p class="text-sm text-toned sm:text-base">
              通过溯源码查询荔枝批次摘要、数据库存证或链上校验结果。
            </p>
            <div class="pt-2">
              <UButton :to="tracePath" icon="i-lucide-arrow-right" label="进入查询页" />
            </div>
          </div>
        </UCard>
      </div>
    </div>
  </UContainer>
</template>
