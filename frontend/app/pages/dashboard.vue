<script setup lang="ts">
import { computed, ref } from 'vue'

const refreshedAt = ref(new Date())

const totals = {
  batchTotal: 186,
}

const statusDistribution = {
  anchored: 142,
  pendingAnchor: 31,
  anchorFailed: 13,
}

const ripenessDistribution = {
  green: 328,
  half: 497,
  red: 1206,
  young: 211,
}

const unripeMetrics = {
  unripeBatchCount: 39,
  unripeBatchRatio: 0.2097,
  threshold: 0.15,
  unripeHandling: 'sorted_out',
}

const recentAnchors = [
  {
    batchId: 'batch_01J3NQ2JAYPH1M8QZ0QDZBY8BK',
    traceCode: 'TRC-9A7X-11QF',
    txHash: '0x8a9b4f2e713f11a2ce8f51a4d13b27505849621d2df612f7396db7d2e498cb78',
    anchoredAt: '2026-03-02T09:30:12Z',
  },
  {
    batchId: 'batch_01J3NQ5G5F0M5MBR9V8XW4NQ6M',
    traceCode: 'TRC-7K2N-3MPL',
    txHash: '0x9d3a01f15a42a5111d59c5adf6b40b6bd8824fdb8235e48d83d52b25f0c36ceb',
    anchoredAt: '2026-03-02T09:17:08Z',
  },
]

const reconcileStats = {
  pendingCount: 31,
  retriedTotal: 58,
  failedTotal: 13,
  lastReconcileAt: '2026-03-02T09:35:00Z',
}

const refreshedAtText = computed(() =>
  new Intl.DateTimeFormat('zh-CN', {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(refreshedAt.value),
)

function refreshStaticBoard() {
  refreshedAt.value = new Date()
}
</script>

<template>
  <div class="lr-shell space-y-4">
    <UCard class="lr-panel">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">数据看板（静态示例）</h1>
          <p class="text-sm text-neutral-600">
            最近刷新：{{ refreshedAtText }}
          </p>
        </div>
        <UButton icon="i-lucide-refresh-cw" @click="refreshStaticBoard">
          刷新示例
        </UButton>
      </div>
    </UCard>

    <div class="grid gap-4 lg:grid-cols-2">
      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">批次总览</h2>
        </template>
        <div class="space-y-2">
          <p>batch_total: <strong>{{ totals.batchTotal }}</strong></p>
          <p>anchored: <strong>{{ statusDistribution.anchored }}</strong></p>
          <p>pending_anchor: <strong>{{ statusDistribution.pendingAnchor }}</strong></p>
          <p>anchor_failed: <strong>{{ statusDistribution.anchorFailed }}</strong></p>
        </div>
      </UCard>

      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">成熟度分布</h2>
        </template>
        <div class="space-y-2">
          <p>green: <strong>{{ ripenessDistribution.green }}</strong></p>
          <p>half: <strong>{{ ripenessDistribution.half }}</strong></p>
          <p>red: <strong>{{ ripenessDistribution.red }}</strong></p>
          <p>young: <strong>{{ ripenessDistribution.young }}</strong></p>
        </div>
      </UCard>

      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">未成熟拦截指标</h2>
        </template>
        <div class="space-y-2">
          <p>unripe_batch_count: <strong>{{ unripeMetrics.unripeBatchCount }}</strong></p>
          <p>unripe_batch_ratio: <strong>{{ unripeMetrics.unripeBatchRatio }}</strong></p>
          <p>threshold: <strong>{{ unripeMetrics.threshold }}</strong></p>
          <p>unripe_handling: <strong>{{ unripeMetrics.unripeHandling }}</strong></p>
        </div>
      </UCard>

      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">补链统计</h2>
        </template>
        <div class="space-y-2">
          <p>pending_count: <strong>{{ reconcileStats.pendingCount }}</strong></p>
          <p>retried_total: <strong>{{ reconcileStats.retriedTotal }}</strong></p>
          <p>failed_total: <strong>{{ reconcileStats.failedTotal }}</strong></p>
          <p>last_reconcile_at: <strong>{{ reconcileStats.lastReconcileAt }}</strong></p>
        </div>
      </UCard>
    </div>

    <UCard class="lr-panel">
      <template #header>
        <h2 class="text-base font-semibold">最近上链记录（静态示例）</h2>
      </template>
      <div class="space-y-3">
        <div
          v-for="anchor in recentAnchors"
          :key="anchor.batchId"
          class="rounded-md border border-neutral-200 p-3"
        >
          <p class="text-sm">batch_id: <strong>{{ anchor.batchId }}</strong></p>
          <p class="text-sm">trace_code: <strong>{{ anchor.traceCode }}</strong></p>
          <p class="text-sm break-all">tx_hash: <strong>{{ anchor.txHash }}</strong></p>
          <p class="text-sm">anchored_at: <strong>{{ anchor.anchoredAt }}</strong></p>
        </div>
      </div>
    </UCard>
  </div>
</template>
