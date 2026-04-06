<script setup lang="ts">
import type { Batch } from '~/types/batch'
import type { BatchStatus } from '~/types/trace'
import { buildTracePathWithFrom } from '~/utils/trace-from'

const props = defineProps<{
  batch: Batch
  statusCode: number
}>()

const emit = defineEmits<{
  (event: 'continue'): void
}>()

const copied = ref(false)

const statusMeta = computed(() => getStatusMeta(props.batch.status))
const tracePath = computed(() => buildTracePathWithFrom(props.batch.trace_code, 'batch_create'))
const traceUrl = computed(() => {
  if (!import.meta.client) {
    return tracePath.value
  }
  return new URL(tracePath.value, window.location.origin).toString()
})

async function copyTraceLink() {
  if (!navigator?.clipboard) {
    return
  }
  await navigator.clipboard.writeText(traceUrl.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1800)
}

function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date)
}

function getStatusMeta(status: BatchStatus): { color: 'primary' | 'success' | 'warning' | 'error', label: string } {
  if (status === 'stored') {
    return { color: 'primary', label: '已入库' }
  }
  if (status === 'anchored') {
    return { color: 'success', label: '上链成功' }
  }
  if (status === 'pending_anchor') {
    return { color: 'warning', label: '已保存待补链' }
  }
  return { color: 'error', label: '补链失败' }
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            建批结果
          </h2>
          <p class="mt-1 text-xs text-muted">
            HTTP {{ statusCode }} · batch_id {{ batch.batch_id }}
          </p>
        </div>
        <UBadge :color="statusMeta.color" variant="soft">
          {{ statusMeta.label }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        :color="statusMeta.color"
        variant="subtle"
        icon="i-lucide-circle-check-big"
        :title="statusMeta.label"
      >
        <template #description>
          <p class="text-sm">
            溯源码：{{ batch.trace_code }}
          </p>
          <p class="mt-1 text-xs text-muted">
            {{ batch.trace_mode === 'database' ? '数据库存证，可直接查询。' : '区块链模式批次，可公开验真。' }}
          </p>
          <p class="mt-1 text-xs text-muted">
            创建时间：{{ formatDateTime(batch.created_at) }}
          </p>
        </template>
      </UAlert>

      <div class="flex flex-wrap gap-2">
        <UButton
          color="neutral"
          variant="soft"
          :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
          :label="copied ? '已复制' : '复制溯源链接'"
          @click="copyTraceLink"
        />
        <UButton
          :to="tracePath"
          icon="i-lucide-arrow-up-right"
          label="查看溯源页"
        />
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-rotate-ccw"
          label="继续建批"
          @click="emit('continue')"
        />
      </div>

      <UCard v-if="batch.anchor_proof" variant="subtle" :ui="{ body: 'px-4 py-3' }">
        <p class="text-xs text-muted">
          链上交易哈希
        </p>
        <p class="mt-1 break-all text-sm font-medium text-default">
          {{ batch.anchor_proof.tx_hash }}
        </p>
      </UCard>
    </div>
  </UCard>
</template>
