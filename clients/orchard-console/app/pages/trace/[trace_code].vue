<script setup lang="ts">
import type { TraceApiError, TraceResponse, TraceViewState } from '~/types/trace'
import { buildAppPath, inferAppBasePath } from '~/utils/app-path'
import { getTraceBackTarget, getTraceFromQuery } from '~/utils/trace-from'
import { getTraceCodeFromRouteParam } from '~/utils/trace-route'

useSeoMeta({
  title: '溯源码查询结果',
  description: '查看荔枝批次摘要与链上验签状态。'
})

const route = useRoute()
const { getPublicTrace, parseTraceError } = useTraceApi()

const traceCode = computed(() => getTraceCodeFromRouteParam(route.params.trace_code as string | string[] | undefined))
const traceFrom = computed(() => getTraceFromQuery(route.query.from as string | string[] | undefined))
const appBasePath = computed(() =>
  import.meta.client ? inferAppBasePath(window.location.pathname, route.path) : ''
)
const backTarget = computed(() => {
  const target = getTraceBackTarget(traceFrom.value)
  if (!target) {
    return null
  }
  return {
    ...target,
    to: buildAppPath(appBasePath.value, target.to)
  }
})
const traceLandingPath = computed(() => buildAppPath(appBasePath.value, '/trace'))

const viewState = ref<TraceViewState>('loading')
const traceData = ref<TraceResponse | null>(null)
const traceError = ref<TraceApiError | null>(null)
const copied = ref(false)

const unavailableTitle = computed(() =>
  traceError.value?.statusCode === 503 ? '服务暂不可用' : '查询失败'
)

const unavailableDescription = computed(() => {
  if (traceError.value?.statusCode === 503) {
    return traceError.value.message || '网关服务暂时不可用，请稍后重试。'
  }
  return traceError.value?.message || '请求处理失败，请稍后重试。'
})

async function copyCurrentTraceCode() {
  if (!navigator?.clipboard || !traceCode.value) {
    return
  }
  await navigator.clipboard.writeText(traceCode.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1800)
}

async function loadTrace() {
  const code = traceCode.value
  if (!code) {
    viewState.value = 'not_found'
    traceData.value = null
    traceError.value = null
    return
  }

  viewState.value = 'loading'
  traceData.value = null
  traceError.value = null

  try {
    const data = await getPublicTrace(code)
    traceData.value = data
    viewState.value = 'success'
  } catch (error) {
    const parsed = parseTraceError(error)
    traceError.value = parsed
    viewState.value = parsed.statusCode === 404 ? 'not_found' : 'unavailable'
  }
}

watch(traceCode, () => {
  void loadTrace()
}, { immediate: true })
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="mx-auto max-w-4xl space-y-5">
      <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="space-y-1">
            <p class="text-xs uppercase tracking-wider text-muted">
              Trace Query
            </p>
            <div class="flex flex-wrap items-center gap-2">
              <h1 class="text-lg font-semibold text-highlighted sm:text-xl">
                溯源码查询结果
              </h1>
              <UBadge color="neutral" variant="soft">
                {{ traceCode || '未提供溯源码' }}
              </UBadge>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              v-if="backTarget"
              :to="backTarget.to"
              color="neutral"
              variant="outline"
              icon="i-lucide-arrow-left"
              :label="backTarget.label"
            />
            <UButton
              color="neutral"
              variant="soft"
              :label="copied ? '已复制' : '复制溯源码'"
              :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
              :disabled="!traceCode"
              @click="copyCurrentTraceCode"
            />
            <UButton
              :to="traceLandingPath"
              color="neutral"
              variant="outline"
              icon="i-lucide-search"
              label="重新查询"
            />
          </div>
        </div>
      </UCard>

      <UCard v-if="viewState === 'loading'" variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
        <div class="space-y-4">
          <USkeleton class="h-6 w-48" />
          <USkeleton class="h-20 w-full" />
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <USkeleton class="h-24 w-full" />
            <USkeleton class="h-24 w-full" />
          </div>
        </div>
      </UCard>

      <TraceResultPanel v-else-if="viewState === 'success' && traceData" :trace="traceData" />

      <UAlert
        v-else-if="viewState === 'not_found'"
        color="warning"
        variant="subtle"
        icon="i-lucide-search-x"
        title="未找到对应溯源码"
        description="请检查二维码或输入内容是否正确。"
      >
        <template #description>
          <p class="text-sm">
            请检查二维码或输入内容是否正确。
          </p>
          <p v-if="traceError?.requestId" class="mt-1 text-xs text-muted">
            请求 ID：{{ traceError.requestId }}
          </p>
        </template>
      </UAlert>

      <UCard v-else variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
        <UAlert
          color="error"
          variant="subtle"
          icon="i-lucide-server-crash"
          :title="unavailableTitle"
          :description="unavailableDescription"
        >
          <template #description>
            <p class="text-sm">
              {{ unavailableDescription }}
            </p>
            <p v-if="traceError?.requestId" class="mt-1 text-xs text-muted">
              请求 ID：{{ traceError.requestId }}
            </p>
          </template>
        </UAlert>

        <div class="mt-4">
          <UButton color="error" variant="outline" icon="i-lucide-refresh-cw" label="重试查询" @click="loadTrace" />
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
