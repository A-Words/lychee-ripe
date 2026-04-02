<script setup lang="ts">
import { buildTraceDetailPathFromQuery } from '~/utils/trace-from'
import { normalizeTraceCode } from '~/utils/trace-route'

const inputCode = ref('')
const errorMessage = ref('')
const pending = ref(false)
const route = useRoute()

async function submit() {
  if (pending.value) {
    return
  }

  const normalized = normalizeTraceCode(inputCode.value)
  if (!normalized) {
    errorMessage.value = '请输入溯源码。'
    return
  }

  pending.value = true
  errorMessage.value = ''
  try {
    await navigateTo(buildTraceDetailPathFromQuery(normalized, route.query.from as string | string[] | undefined))
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
    <form class="space-y-4" @submit.prevent="submit">
      <div class="space-y-2">
        <label class="block text-sm font-medium text-highlighted" for="trace-code-input">
          输入溯源码
        </label>
        <UInput
          id="trace-code-input"
          v-model="inputCode"
          :disabled="pending"
          size="xl"
          icon="i-lucide-qr-code"
          placeholder="例如：TRC-9A7X-11QF"
          autocomplete="off"
        />
        <p class="text-xs text-muted">
          扫码链接将直接打开 `/trace/{trace_code}`，此处用于手动查询。
        </p>
      </div>

      <UAlert
        v-if="errorMessage"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        :description="errorMessage"
      />

      <div class="flex flex-wrap items-center gap-3">
        <UButton
          type="submit"
          :disabled="pending"
          :loading="pending"
          size="lg"
          icon="i-lucide-search"
          label="查询溯源信息"
        />
        <p class="text-xs text-muted">
          公开查询，无需登录。
        </p>
      </div>
    </form>
  </UCard>
</template>
