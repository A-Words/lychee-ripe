<script setup lang="ts">
import { computed } from 'vue'
import { APP_NAV_ITEMS } from '../constants/navigation'

const route = useRoute()

const currentPath = computed(() => route.path)

function isActive(path: string, matchPrefix: string): boolean {
  if (path === '/') {
    return currentPath.value === '/'
  }
  return currentPath.value === matchPrefix || currentPath.value.startsWith(`${matchPrefix}/`)
}
</script>

<template>
  <header class="lr-nav">
    <div class="lr-shell">
      <nav class="lr-nav-links">
        <NuxtLink
          v-for="item in APP_NAV_ITEMS"
          :key="item.key"
          :to="item.to"
          class="lr-nav-link"
          :class="{ 'lr-nav-link-active': isActive(item.to, item.matchPrefix) }"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>
    </div>
  </header>
</template>
