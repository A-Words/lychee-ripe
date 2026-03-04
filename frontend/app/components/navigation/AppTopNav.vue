<script setup lang="ts">
interface NavItem {
  key: string
  label: string
  to: string
  activePrefixes: string[]
}

const route = useRoute()
const mobileMenuOpen = ref(false)

const navItems: NavItem[] = [
  {
    key: 'batch_create',
    label: '识别建批',
    to: '/batch/create',
    activePrefixes: ['/batch/create']
  },
  {
    key: 'dashboard',
    label: '数据看板',
    to: '/dashboard',
    activePrefixes: ['/dashboard']
  },
  {
    key: 'trace',
    label: '溯源查询',
    to: '/trace',
    activePrefixes: ['/trace']
  }
]

function isActive(item: NavItem): boolean {
  return item.activePrefixes.some((prefix) => route.path.startsWith(prefix))
}

watch(() => route.fullPath, () => {
  mobileMenuOpen.value = false
})
</script>

<template>
  <header class="sticky top-0 z-50 border-b border-default bg-default/90 backdrop-blur">
    <UContainer class="flex h-14 items-center justify-between gap-3">
      <NuxtLink to="/" class="text-sm font-semibold tracking-wide text-highlighted">
        Lychee Ripe
      </NuxtLink>

      <nav class="hidden items-center gap-2 md:flex">
        <UButton
          v-for="item in navItems"
          :key="item.key"
          :to="item.to"
          :color="isActive(item) ? 'primary' : 'neutral'"
          :variant="isActive(item) ? 'solid' : 'ghost'"
          size="sm"
          :label="item.label"
        />
      </nav>

      <UButton
        class="md:hidden"
        color="neutral"
        variant="ghost"
        icon="i-lucide-menu"
        aria-label="切换导航菜单"
        @click="mobileMenuOpen = !mobileMenuOpen"
      />
    </UContainer>

    <div v-if="mobileMenuOpen" class="border-t border-default md:hidden">
      <UContainer class="py-2">
        <nav class="grid gap-2">
          <UButton
            v-for="item in navItems"
            :key="`${item.key}-mobile`"
            :to="item.to"
            :color="isActive(item) ? 'primary' : 'neutral'"
            :variant="isActive(item) ? 'solid' : 'ghost'"
            block
            class="justify-start"
            :label="item.label"
          />
        </nav>
      </UContainer>
    </div>
  </header>
</template>
