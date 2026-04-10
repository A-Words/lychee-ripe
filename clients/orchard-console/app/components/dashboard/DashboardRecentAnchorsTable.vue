<script setup lang="ts">
import { DASHBOARD_STATUS_META } from '~/constants/dashboard-status'
import type { RecentAnchorRecord } from '~/types/dashboard'
import { formatDateTime, truncateTxHash } from '~/utils/dashboard-format'
import { buildDashboardTracePath } from '~/utils/dashboard-route'

const props = defineProps<{
  records: RecentAnchorRecord[]
}>()
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-0' }">
    <template #header>
      <div class="px-4 py-4 sm:px-5">
        <h3 class="text-base font-semibold text-highlighted">
          最近链上记录
        </h3>
        <p class="mt-1 text-xs text-muted">
          默认按 created_at 倒序
        </p>
      </div>
    </template>

    <div v-if="!records.length" class="px-4 pb-5 sm:px-5">
      <UAlert
        color="neutral"
        variant="subtle"
        icon="i-lucide-database-zap"
        title="暂无链上记录"
        description="当前环境尚未产生区块链模式的已上链批次。"
      />
    </div>

    <div v-else class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead class="bg-muted/40 text-xs uppercase tracking-wider text-muted">
          <tr>
            <th class="px-4 py-3 text-left">
              溯源码
            </th>
            <th class="px-4 py-3 text-left">
              批次号
            </th>
            <th class="px-4 py-3 text-left">
              状态
            </th>
            <th class="px-4 py-3 text-left">
              上链时间
            </th>
            <th class="px-4 py-3 text-left">
              交易哈希
            </th>
            <th class="px-4 py-3 text-right">
              操作
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="item in records"
            :key="`${item.batch_id}-${item.trace_code}`"
            class="border-b border-default/60"
          >
            <td class="px-4 py-3 font-medium text-default">
              {{ item.trace_code }}
            </td>
            <td class="px-4 py-3 text-toned">
              {{ item.batch_id }}
            </td>
            <td class="px-4 py-3">
              <UBadge :color="DASHBOARD_STATUS_META[item.status].color" variant="soft">
                {{ DASHBOARD_STATUS_META[item.status].label }}
              </UBadge>
            </td>
            <td class="px-4 py-3 text-toned">
              {{ formatDateTime(item.anchored_at || item.created_at) }}
            </td>
            <td class="px-4 py-3 font-mono text-xs text-toned">
              {{ truncateTxHash(item.tx_hash) }}
            </td>
            <td class="px-4 py-3 text-right">
              <UButton
                size="xs"
                icon="i-lucide-arrow-up-right"
                :to="buildDashboardTracePath(item.trace_code)"
                label="查看溯源"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </UCard>
</template>
