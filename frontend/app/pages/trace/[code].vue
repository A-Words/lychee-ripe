<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { buildTracePath } from '../../constants/navigation'

const route = useRoute()
const router = useRouter()

const traceCode = computed(() => String(route.params.code ?? '').trim())
const editableCode = ref(traceCode.value)

watch(
  traceCode,
  (nextCode) => {
    editableCode.value = nextCode
  },
  { immediate: true },
)

const batchPreview = computed(() => ({
  batchId: 'batch_01J3NQ2JAYPH1M8QZ0QDZBY8BK',
  traceCode: traceCode.value || 'TRC-UNKNOWN',
  orchardName: '增城仙村示例果园',
  plotName: 'A-07',
  harvestedAt: '2026-03-01T08:30:00Z',
  summary: {
    total: 124,
    green: 12,
    half: 26,
    red: 70,
    young: 16,
    unripeCount: 28,
    unripeRatio: 0.2258,
    unripeHandling: 'sorted_out',
  },
}))

const verifyExamples = [
  {
    status: 'pass',
    reason: 'anchor_hash matches on-chain record',
    color: 'success' as const,
  },
  {
    status: 'pending',
    reason: 'batch is not anchored yet',
    color: 'warning' as const,
  },
  {
    status: 'fail',
    reason: 'anchor_hash does not match on-chain record',
    color: 'error' as const,
  },
]

function gotoTraceCode() {
  const normalizedCode = editableCode.value.trim().toUpperCase()
  if (!normalizedCode) {
    return
  }
  void router.push(buildTracePath(normalizedCode))
}
</script>

<template>
  <div class="lr-shell space-y-4">
    <UCard class="lr-panel">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">公开溯源查询（静态示例）</h1>
          <p class="text-sm text-neutral-600">
            当前 trace_code：<code>{{ batchPreview.traceCode }}</code>
          </p>
        </div>

        <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          <UInput
            v-model="editableCode"
            placeholder="输入示例 trace code"
            class="sm:min-w-64"
          />
          <UButton icon="i-lucide-search" @click="gotoTraceCode">
            查询示例
          </UButton>
        </div>
      </div>
    </UCard>

    <UCard class="lr-panel">
      <template #header>
        <h2 class="text-base font-semibold">批次摘要（示例）</h2>
      </template>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <p class="text-xs text-neutral-500">批次编号</p>
          <p class="font-medium">{{ batchPreview.batchId }}</p>
        </div>
        <div>
          <p class="text-xs text-neutral-500">果园</p>
          <p class="font-medium">{{ batchPreview.orchardName }}</p>
        </div>
        <div>
          <p class="text-xs text-neutral-500">地块</p>
          <p class="font-medium">{{ batchPreview.plotName }}</p>
        </div>
        <div>
          <p class="text-xs text-neutral-500">采摘时间</p>
          <p class="font-medium">{{ batchPreview.harvestedAt }}</p>
        </div>
      </div>

      <div class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <UBadge color="neutral" variant="soft">total {{ batchPreview.summary.total }}</UBadge>
        <UBadge color="neutral" variant="soft">green {{ batchPreview.summary.green }}</UBadge>
        <UBadge color="neutral" variant="soft">half {{ batchPreview.summary.half }}</UBadge>
        <UBadge color="neutral" variant="soft">red {{ batchPreview.summary.red }}</UBadge>
        <UBadge color="neutral" variant="soft">young {{ batchPreview.summary.young }}</UBadge>
        <UBadge color="neutral" variant="soft">unripe_count {{ batchPreview.summary.unripeCount }}</UBadge>
        <UBadge color="neutral" variant="soft">unripe_ratio {{ batchPreview.summary.unripeRatio }}</UBadge>
        <UBadge color="neutral" variant="soft">{{ batchPreview.summary.unripeHandling }}</UBadge>
      </div>
    </UCard>

    <UCard class="lr-panel">
      <template #header>
        <h2 class="text-base font-semibold">验签结果三态（示例）</h2>
      </template>

      <div class="space-y-2">
        <UAlert
          v-for="example in verifyExamples"
          :key="example.status"
          :color="example.color"
          variant="soft"
          :title="`verify_status = ${example.status}`"
          :description="example.reason"
        />
      </div>
    </UCard>
  </div>
</template>
