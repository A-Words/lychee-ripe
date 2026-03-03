<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useDashboardOverview } from '../composables/useDashboardOverview'

const DASHBOARD_API_KEY_STORAGE_KEY = 'lychee-ripe.dashboard.apiKey'

const dashboardQuery = useDashboardOverview()
const dashboardApiKey = ref('')
const refreshedAt = ref<Date | null>(null)

const overview = computed(() => dashboardQuery.data.value)
const fetchStatus = computed(() => dashboardQuery.status.value)
const fetchError = computed(() => dashboardQuery.lastError.value)

const refreshedAtText = computed(() => {
  if (!refreshedAt.value) {
    return '--'
  }
  return formatDateTime(refreshedAt.value.toISOString())
})

watch(
  () => dashboardApiKey.value,
  (nextKey) => {
    if (!import.meta.client) {
      return
    }
    localStorage.setItem(DASHBOARD_API_KEY_STORAGE_KEY, nextKey.trim())
  },
)

onMounted(() => {
  if (import.meta.client) {
    dashboardApiKey.value = localStorage.getItem(DASHBOARD_API_KEY_STORAGE_KEY) ?? ''
  }
  void refreshOverview()
})

async function refreshOverview() {
  try {
    await dashboardQuery.fetchOverview({ apiKey: dashboardApiKey.value })
    refreshedAt.value = new Date()
  } catch {
    // rendered by fetchError alert
  }
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat('zh-CN', {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}
</script>

<template>
  <div class="lr-shell space-y-4">
    <UCard class="lr-panel">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">数据看板</h1>
          <p class="text-sm text-neutral-600">
            最近刷新：{{ refreshedAtText }}
          </p>
        </div>
        <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          <UInput
            v-model="dashboardApiKey"
            type="password"
            placeholder="X-API-Key（如已启用鉴权）"
            class="sm:min-w-72"
          />
          <UButton
            icon="i-lucide-refresh-cw"
            :loading="fetchStatus === 'loading'"
            @click="refreshOverview"
          >
            刷新
          </UButton>
        </div>
      </div>
    </UCard>

    <UAlert
      v-if="fetchStatus === 'loading'"
      color="neutral"
      variant="soft"
      title="看板加载中"
      description="正在请求 /v1/dashboard/overview ..."
    />

    <UAlert
      v-else-if="fetchError"
      :color="fetchError.status === 401 ? 'warning' : 'error'"
      variant="soft"
      :title="fetchError.status === 401 ? '鉴权失败' : '看板请求失败'"
      :description="fetchError.message"
    />

    <template v-else-if="overview">
      <div class="grid gap-4 lg:grid-cols-2">
        <UCard class="lr-panel">
          <template #header>
            <h2 class="text-base font-semibold">批次总览</h2>
          </template>
          <div class="space-y-2">
            <p>batch_total: <strong>{{ overview.totals.batch_total }}</strong></p>
            <p>anchored: <strong>{{ overview.status_distribution.anchored }}</strong></p>
            <p>pending_anchor: <strong>{{ overview.status_distribution.pending_anchor }}</strong></p>
            <p>anchor_failed: <strong>{{ overview.status_distribution.anchor_failed }}</strong></p>
          </div>
        </UCard>

        <UCard class="lr-panel">
          <template #header>
            <h2 class="text-base font-semibold">成熟度分布</h2>
          </template>
          <div class="space-y-2">
            <p>green: <strong>{{ overview.ripeness_distribution.green }}</strong></p>
            <p>half: <strong>{{ overview.ripeness_distribution.half }}</strong></p>
            <p>red: <strong>{{ overview.ripeness_distribution.red }}</strong></p>
            <p>young: <strong>{{ overview.ripeness_distribution.young }}</strong></p>
          </div>
        </UCard>

        <UCard class="lr-panel">
          <template #header>
            <h2 class="text-base font-semibold">未成熟指标</h2>
          </template>
          <div class="space-y-2">
            <p>unripe_batch_count: <strong>{{ overview.unripe_metrics.unripe_batch_count }}</strong></p>
            <p>unripe_batch_ratio: <strong>{{ overview.unripe_metrics.unripe_batch_ratio }}</strong></p>
            <p>threshold: <strong>{{ overview.unripe_metrics.threshold }}</strong></p>
            <p>unripe_handling: <strong>{{ overview.unripe_metrics.unripe_handling }}</strong></p>
          </div>
        </UCard>

        <UCard class="lr-panel">
          <template #header>
            <h2 class="text-base font-semibold">补链统计</h2>
          </template>
          <div class="space-y-2">
            <p>pending_count: <strong>{{ overview.reconcile_stats.pending_count }}</strong></p>
            <p>retried_total: <strong>{{ overview.reconcile_stats.retried_total }}</strong></p>
            <p>failed_total: <strong>{{ overview.reconcile_stats.failed_total }}</strong></p>
            <p>
              last_reconcile_at:
              <strong>
                {{
                  overview.reconcile_stats.last_reconcile_at
                    ? formatDateTime(overview.reconcile_stats.last_reconcile_at)
                    : '-'
                }}
              </strong>
            </p>
          </div>
        </UCard>
      </div>

      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">最近上链记录</h2>
        </template>
        <div
          v-if="overview.recent_anchors.length === 0"
          class="text-sm text-neutral-500"
        >
          暂无最近上链记录。
        </div>
        <div
          v-else
          class="space-y-3"
        >
          <div
            v-for="anchor in overview.recent_anchors"
            :key="anchor.batch_id"
            class="rounded-md border border-neutral-200 p-3"
          >
            <p class="text-sm">batch_id: <strong>{{ anchor.batch_id }}</strong></p>
            <p class="text-sm">trace_code: <strong>{{ anchor.trace_code }}</strong></p>
            <p class="text-sm">status: <strong>{{ anchor.status }}</strong></p>
            <p class="text-sm break-all">tx_hash: <strong>{{ anchor.tx_hash ?? '-' }}</strong></p>
            <p class="text-sm">
              anchored_at:
              <strong>{{ anchor.anchored_at ? formatDateTime(anchor.anchored_at) : '-' }}</strong>
            </p>
            <p class="text-sm">
              created_at:
              <strong>{{ formatDateTime(anchor.created_at) }}</strong>
            </p>
          </div>
        </div>
      </UCard>
    </template>
  </div>
</template>
