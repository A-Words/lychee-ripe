<script setup lang="ts">
import { VERIFY_STATUS_META } from '~/constants/verify-status'
import type { TraceResponse } from '~/types/trace'

const props = defineProps<{
  trace: TraceResponse
}>()

const trace = computed(() => props.trace)
const verifyMeta = computed(() => VERIFY_STATUS_META[props.trace.verify_result.verify_status])
const copied = ref(false)
const verifyLead = computed(() =>
  props.trace.verify_result.verify_status === 'recorded' ? '数据库存证' : '链上可核验'
)

const locationText = computed(() => {
  const orchard = props.trace.batch.orchard_name || '未知果园'
  const plot = props.trace.batch.plot_name || '未登记地块'
  return `${orchard} / ${plot}`
})

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

async function copyTraceCode() {
  if (!navigator?.clipboard) {
    return
  }
  await navigator.clipboard.writeText(props.trace.batch.trace_code)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1800)
}
</script>

<template>
  <div class="space-y-4">
    <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="space-y-2">
          <p class="text-xs uppercase tracking-wider text-muted">
            公开溯源档案
          </p>
          <h2 class="text-xl font-semibold text-highlighted sm:text-2xl">
            {{ locationText }}
          </h2>
          <p class="text-sm text-toned">
            批次号 {{ trace.batch.batch_id }}
          </p>
        </div>

        <div class="flex items-center gap-2">
          <TraceStatusBadge :status="trace.verify_result.verify_status" />
          <UButton
            :label="copied ? '已复制' : '复制溯源码'"
            :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
            color="neutral"
            variant="soft"
            @click="copyTraceCode"
          />
        </div>
      </div>

      <div class="mt-5 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            溯源码
          </p>
          <p class="mt-1 font-semibold text-default">
            {{ trace.batch.trace_code }}
          </p>
        </UCard>
        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            采摘时间
          </p>
          <p class="mt-1 font-semibold text-default">
            {{ formatDateTime(trace.batch.harvested_at) }}
          </p>
        </UCard>
        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            批次创建时间
          </p>
          <p class="mt-1 font-semibold text-default">
            {{ formatDateTime(trace.batch.created_at) }}
          </p>
        </UCard>
        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            批次状态
          </p>
          <p class="mt-1 font-semibold text-default">
            {{ trace.batch.status }}
          </p>
        </UCard>
      </div>
    </UCard>

    <UAlert
      :color="verifyMeta.color"
      variant="subtle"
      icon="i-lucide-shield-check"
      :title="verifyMeta.label"
      :description="verifyMeta.description"
    >
      <template #description>
        <p class="text-sm">
          {{ verifyLead }}：{{ verifyMeta.description }}
        </p>
        <p class="mt-1 text-sm">
          校验说明：{{ trace.verify_result.reason }}
        </p>
      </template>
    </UAlert>

    <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
      <template #header>
        <div>
          <h3 class="text-base font-semibold text-highlighted">
            成熟度摘要
          </h3>
          <p class="mt-1 text-sm text-toned">
            共检测 {{ trace.batch.summary.total }} 颗荔枝
          </p>
        </div>
      </template>

      <TraceRipenessSummaryCards :summary="trace.batch.summary" />
    </UCard>
  </div>
</template>
