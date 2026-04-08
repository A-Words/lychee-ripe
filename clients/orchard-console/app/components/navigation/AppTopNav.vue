<script setup lang="ts">
import { buildTopNavItems } from '~/utils/top-nav-items'

const route = useRoute()
const auth = useAuth()

onMounted(() => {
  void auth.init()
})

const navItems = computed(() => buildTopNavItems(route.path, auth.isAdmin.value))
const principalName = computed(() => auth.principal.value?.display_name || '')

const authActionLabel = computed(() => auth.isAuthenticated.value ? '退出登录' : '登录')

async function handleAuthAction() {
  if (auth.isAuthenticated.value) {
    await auth.logout()
    return
  }
  await auth.login(route.fullPath)
}
</script>

<template>
  <UHeader
    title="Lychee Ripe"
    to="/"
    mode="slideover"
    :toggle="{ color: 'neutral', variant: 'ghost' }"
  >
    <UNavigationMenu
      :items="navItems"
      orientation="horizontal"
      color="neutral"
      variant="pill"
      highlight
      :ui="{ root: 'w-full' }"
    />

    <div class="flex items-center gap-2">
      <UBadge v-if="auth.principal" color="neutral" variant="soft">
        {{ principalName }}
      </UBadge>
      <UButton
        color="neutral"
        variant="ghost"
        icon="i-lucide-log-in"
        :label="authActionLabel"
        @click="handleAuthAction"
      />
    </div>

    <template #body>
      <UNavigationMenu
        :items="navItems"
        orientation="vertical"
        color="neutral"
        variant="pill"
        :ui="{
          root: 'w-full',
          list: 'w-full',
          link: 'justify-start'
        }"
      />
      <div class="mt-4">
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-log-in"
          block
          :label="authActionLabel"
          @click="handleAuthAction"
        />
      </div>
    </template>
  </UHeader>
</template>
