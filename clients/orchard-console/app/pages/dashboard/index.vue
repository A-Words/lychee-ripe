<script setup lang="ts">
import type {
  DashboardApiError,
  DashboardOverviewResponse,
  DashboardViewState
} from '~/types/dashboard'
import { formatDateTime } from '~/utils/dashboard-format'

useSeoMeta({
  title: '数据看板',
  description: '批次、成熟度与存证模式的聚合看板。'
})

const { getOverview, parseDashboardError } = useDashboardApi()

const viewState = ref<DashboardViewState>('loading')
const overview = ref<DashboardOverviewResponse | null>(null)
const apiError = ref<DashboardApiError | null>(null)
const loading = ref(false)
const lastRefreshedAt = ref<Date | null>(null)
const consecutiveRefreshFailures = ref(0)

const refreshIntervalMs = 30_000
const maxRefreshIntervalMs = 120_000
const refreshTimer = ref<number | null>(null)

const shouldStopAutoRefresh = computed(() => viewState.value === 'auth_blocked')
const isReadyLike = computed(() => viewState.value === 'ready' || viewState.value === 'empty')
const isEmpty = computed(() => viewState.value === 'empty')
const isShowingStaleOverview = computed(() =>
  Boolean(overview.value) && isReadyLike.value && Boolean(apiError.value)
)

const unavailableTitle = computed(() =>
  viewState.value === 'auth_blocked' ? '网关鉴权已开启' : '看板服务不可用'
)

const unavailableDescription = computed(() => {
  if (viewState.value === 'auth_blocked') {
    return apiError.value?.message || '当前环境已启用鉴权，本期页面不传 API Key。'
  }
  return apiError.value?.message || '请检查网关服务状态后重试。'
})

const lastRefreshText = computed(() =>
  lastRefreshedAt.value ? formatDateTime(lastRefreshedAt.value.toISOString()) : '--'
)

const staleOverviewMessage = computed(() =>
  apiError.value?.message || '刷新失败，当前展示的是上一次成功加载的数据。'
)

function clearRefreshTimer() {
  if (refreshTimer.value !== null) {
    window.clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
}

function scheduleAutoRefresh() {
  clearRefreshTimer()

  if (!import.meta.client || shouldStopAutoRefresh.value || document.visibilityState !== 'visible') {
    return
  }

  refreshTimer.value = window.setTimeout(() => {
    void loadOverview(true)
  }, currentRefreshDelayMs())
}

function currentRefreshDelayMs() {
  if (consecutiveRefreshFailures.value <= 0) {
    return refreshIntervalMs
  }
  return Math.min(refreshIntervalMs * 2 ** consecutiveRefreshFailures.value, maxRefreshIntervalMs)
}

async function loadOverview(manual: boolean) {
  if (loading.value) {
    return
  }

  loading.value = true
  if (!manual && !overview.value) {
    viewState.value = 'loading'
  }

  try {
    const data = await getOverview()
    overview.value = data
    apiError.value = null
    lastRefreshedAt.value = new Date()
    consecutiveRefreshFailures.value = 0
    viewState.value = data.totals.batch_total === 0 ? 'empty' : 'ready'
  } catch (error) {
    const parsed = parseDashboardError(error)
    const currentOverview = overview.value
    apiError.value = parsed

    if (parsed.statusCode === 401 || parsed.statusCode === 403) {
      overview.value = null
      viewState.value = 'auth_blocked'
    } else if (currentOverview) {
      consecutiveRefreshFailures.value += 1
      viewState.value = currentOverview.totals.batch_total === 0 ? 'empty' : 'ready'
    } else {
      overview.value = null
      consecutiveRefreshFailures.value += 1
      viewState.value = 'unavailable'
    }
  } finally {
    loading.value = false
    scheduleAutoRefresh()
  }
}

function handleManualRefresh() {
  void loadOverview(true)
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible' && !shouldStopAutoRefresh.value) {
    void loadOverview(true)
    return
  }
  clearRefreshTimer()
}

onMounted(() => {
  void loadOverview(false)
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onBeforeUnmount(() => {
  clearRefreshTimer()
  if (import.meta.client) {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }
})

watch(shouldStopAutoRefresh, () => {
  scheduleAutoRefresh()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="flex flex-wrap items-start justify-between gap-3">
        <div class="space-y-2">
          <p class="text-xs uppercase tracking-widest text-muted">
            Dashboard
          </p>
          <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
            数据看板
          </h1>
          <p class="text-sm text-toned sm:text-base">
            聚合展示批次状态、成熟度分布与当前存证模式。
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <UBadge v-if="overview" color="primary" variant="soft">
            模式：{{ overview.trace_mode === 'database' ? '数据库' : '区块链' }}
          </UBadge>
          <UBadge color="neutral" variant="soft">
            上次刷新：{{ lastRefreshText }}
          </UBadge>
          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-refresh-cw"
            :loading="loading"
            label="立即刷新"
            @click="handleManualRefresh"
          />
        </div>
      </section>

      <div v-if="viewState === 'loading'" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <USkeleton v-for="item in 4" :key="item" class="h-28 w-full" />
        </div>
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <USkeleton class="h-96 w-full" />
          <USkeleton class="h-96 w-full" />
        </div>
      </div>

      <div v-else-if="isReadyLike && overview" class="space-y-4">
        <UAlert
          v-if="isShowingStaleOverview"
          color="warning"
          variant="subtle"
          icon="i-lucide-triangle-alert"
          title="刷新失败，当前展示的是上一次成功加载的数据"
        >
          <template #description>
            <p class="text-sm">
              {{ staleOverviewMessage }}
            </p>
            <p v-if="apiError?.requestId" class="mt-1 text-xs text-muted">
              请求 ID：{{ apiError.requestId }}
            </p>
          </template>
        </UAlert>

        <UAlert
          v-if="isEmpty"
          color="neutral"
          variant="subtle"
          icon="i-lucide-database"
          title="暂无批次数据"
          description="当前数据库中无批次记录，图表已按空数据渲染。"
        />

        <DashboardOverviewCards
          :trace-mode="overview.trace_mode"
          :totals="overview.totals"
          :status-distribution="overview.status_distribution"
        />

        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <DashboardStatusChart
            :trace-mode="overview.trace_mode"
            :status-distribution="overview.status_distribution"
          />
          <DashboardRipenessChart :ripeness-distribution="overview.ripeness_distribution" />
        </div>

        <DashboardMetaPanels
          :unripe-metrics="overview.unripe_metrics"
          :reconcile-stats="overview.reconcile_stats"
        />

        <DashboardRecentAnchorsTable
          v-if="overview.trace_mode === 'blockchain'"
          :records="overview.recent_anchors"
        />
      </div>

      <UCard v-else variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
        <UAlert
          color="error"
          variant="subtle"
          icon="i-lucide-server-crash"
          :title="unavailableTitle"
        >
          <template #description>
            <p class="text-sm">
              {{ unavailableDescription }}
            </p>
            <p v-if="apiError?.requestId" class="mt-1 text-xs text-muted">
              请求 ID：{{ apiError.requestId }}
            </p>
          </template>
        </UAlert>

        <div class="mt-4">
          <UButton
            color="error"
            variant="outline"
            icon="i-lucide-refresh-cw"
            label="重试加载"
            @click="handleManualRefresh"
          />
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
