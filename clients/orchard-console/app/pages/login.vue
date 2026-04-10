<script setup lang="ts">
import { resolveAuthErrorMessage } from '~/utils/auth-error'

useSeoMeta({
  title: '登录',
  description: '使用 OIDC 登录以访问识别建批、看板与管理后台。'
})

const route = useRoute()
const auth = useAuth()
const loading = ref(false)
const actionErrorMessage = ref('')

onMounted(() => {
  void auth.init()
})

const redirectPath = computed(() => String(route.query.redirect || '/dashboard'))
const buttonLabel = computed(() => auth.mode.value === 'disabled' ? '进入系统' : '使用 OIDC 登录')
const isDisabledMode = computed(() => auth.mode.value === 'disabled')
const errorMessage = computed(() => actionErrorMessage.value || resolveAuthErrorMessage(route.query.auth_error))

async function handleLogin() {
  loading.value = true
  actionErrorMessage.value = ''
  try {
    await auth.login(redirectPath.value)
  } catch (error) {
    actionErrorMessage.value = error instanceof Error ? error.message : '登录初始化失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UContainer class="py-16">
    <div class="mx-auto max-w-lg">
      <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
        <div class="space-y-4">
          <div class="space-y-2">
            <p class="text-xs uppercase tracking-widest text-muted">
              Auth
            </p>
            <h1 class="text-2xl font-semibold text-highlighted">
              登录 Lychee Ripe
            </h1>
            <p class="text-sm text-toned">
              {{ isDisabledMode ? '当前为开发旁路模式，将直接以管理员身份进入系统。' : 'Web 端将跳转到 Gateway 托管的 OIDC 登录。' }}
            </p>
          </div>

          <UAlert
            v-if="errorMessage"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="errorMessage"
          />

          <UButton
            block
            color="primary"
            icon="i-lucide-log-in"
            :loading="loading"
            :label="buttonLabel"
            @click="handleLogin"
          />
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
