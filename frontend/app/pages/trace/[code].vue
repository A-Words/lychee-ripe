<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { usePublicTrace } from '../../composables/usePublicTrace'
import { buildTracePath } from '../../constants/navigation'
import type { VerifyStatus } from '../../types/query'

const route = useRoute()
const router = useRouter()

const traceCode = computed(() => String(route.params.code ?? '').trim())
const editableCode = ref(traceCode.value)

const traceQuery = usePublicTrace()

watch(
  traceCode,
  (nextCode) => {
    editableCode.value = nextCode
    if (!nextCode) {
      return
    }
    void traceQuery.fetchTrace(nextCode)
  },
  { immediate: true },
)

const traceData = computed(() => traceQuery.data.value)
const fetchStatus = computed(() => traceQuery.status.value)
const fetchError = computed(() => traceQuery.lastError.value)

const verifyAlert = computed(() => {
  const status = traceData.value?.verify_result.verify_status
  if (!status) {
    return null
  }
  return verifyStatusMeta(status)
})

function verifyStatusMeta(status: VerifyStatus): { color: 'success' | 'warning' | 'error'; title: string } {
  if (status === 'pass') {
    return { color: 'success', title: '验签通过（pass）' }
  }
  if (status === 'pending') {
    return { color: 'warning', title: '待验签（pending）' }
  }
  return { color: 'error', title: '验签失败（fail）' }
}

function gotoTraceCode() {
  const normalizedCode = editableCode.value.trim().toUpperCase()
  if (!normalizedCode) {
    return
  }
  void router.push(buildTracePath(normalizedCode))
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
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">公开溯源查询</h1>
          <p class="text-sm text-neutral-600">
            输入 trace_code 后查询公开溯源与验签结果。
          </p>
        </div>

        <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          <UInput
            v-model="editableCode"
            placeholder="输入 trace code"
            class="sm:min-w-64"
          />
          <UButton
            icon="i-lucide-search"
            :loading="fetchStatus === 'loading'"
            @click="gotoTraceCode"
          >
            查询
          </UButton>
        </div>
      </div>
    </UCard>

    <UAlert
      v-if="fetchStatus === 'loading'"
      color="neutral"
      variant="soft"
      title="查询中"
      description="正在获取溯源数据..."
    />

    <UAlert
      v-else-if="fetchError"
      :color="fetchError.status === 404 ? 'warning' : 'error'"
      variant="soft"
      :title="fetchError.status === 404 ? '溯源码不存在' : '溯源查询失败'"
      :description="fetchError.message"
    />

    <template v-else-if="traceData">
      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">批次摘要</h2>
        </template>

        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <p class="text-xs text-neutral-500">batch_id</p>
            <p class="font-medium break-all">{{ traceData.batch.batch_id }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">trace_code</p>
            <p class="font-medium">{{ traceData.batch.trace_code }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">status</p>
            <p class="font-medium">{{ traceData.batch.status }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">orchard_name</p>
            <p class="font-medium">{{ traceData.batch.orchard_name }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">plot_name</p>
            <p class="font-medium">{{ traceData.batch.plot_name }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">harvested_at</p>
            <p class="font-medium">{{ formatDateTime(traceData.batch.harvested_at) }}</p>
          </div>
          <div>
            <p class="text-xs text-neutral-500">created_at</p>
            <p class="font-medium">{{ formatDateTime(traceData.batch.created_at) }}</p>
          </div>
        </div>

        <div class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <UBadge color="neutral" variant="soft">total {{ traceData.batch.summary.total }}</UBadge>
          <UBadge color="warning" variant="soft">green {{ traceData.batch.summary.green }}</UBadge>
          <UBadge color="neutral" variant="soft">half {{ traceData.batch.summary.half }}</UBadge>
          <UBadge color="neutral" variant="soft">red {{ traceData.batch.summary.red }}</UBadge>
          <UBadge color="warning" variant="soft">young {{ traceData.batch.summary.young }}</UBadge>
          <UBadge color="neutral" variant="soft">unripe_count {{ traceData.batch.summary.unripe_count }}</UBadge>
          <UBadge color="neutral" variant="soft">unripe_ratio {{ traceData.batch.summary.unripe_ratio }}</UBadge>
          <UBadge color="neutral" variant="soft">{{ traceData.batch.summary.unripe_handling }}</UBadge>
        </div>
      </UCard>

      <UCard class="lr-panel">
        <template #header>
          <h2 class="text-base font-semibold">验签结果</h2>
        </template>

        <UAlert
          v-if="verifyAlert"
          :color="verifyAlert.color"
          variant="soft"
          :title="verifyAlert.title"
          :description="traceData.verify_result.reason"
        />
      </UCard>
    </template>
  </div>
</template>
