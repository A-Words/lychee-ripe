<script setup lang="ts">
import { buildAppPath, inferAppBasePath } from '~/utils/app-path'

useSeoMeta({
  title: '登录回调',
  description: '正在恢复登录状态。'
})

const route = useRoute()
const auth = useAuth()
const errorMessage = ref('')
const appBasePath = computed(() =>
  import.meta.client ? inferAppBasePath(window.location.pathname, route.path) : ''
)
const loginPath = computed(() => buildAppPath(appBasePath.value, '/login'))

onMounted(async () => {
  try {
    const target = await auth.handleWebCallback()
    await navigateTo(buildAppPath(appBasePath.value, target || '/dashboard'))
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录回调失败'
  }
})
</script>

<template>
  <UContainer class="py-16">
    <div class="mx-auto max-w-lg">
      <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
        <div class="space-y-4">
          <h1 class="text-xl font-semibold text-highlighted">
            正在完成登录
          </h1>
          <p v-if="!errorMessage" class="text-sm text-toned">
            请稍候，正在同步当前会话并加载用户信息。
          </p>
          <UAlert
            v-else
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="errorMessage"
          />
          <UButton v-if="errorMessage" :to="loginPath" label="返回登录页" />
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
