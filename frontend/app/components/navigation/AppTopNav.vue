<script setup lang="ts">
import { buildTopNavItems } from '~/utils/top-nav-items'

const route = useRoute()

const navItems = computed(() => buildTopNavItems(route.path))
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
    </template>
  </UHeader>
</template>
