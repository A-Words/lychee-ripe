<script setup lang="ts">
import AppTopNav from '~/components/navigation/AppTopNav.vue'
import { shouldShowTopNav } from '~/utils/top-nav-visibility'

const route = useRoute()
const showTopNav = computed(() =>
  shouldShowTopNav(route.path, route.query.from as string | string[] | undefined)
)
</script>

<template>
  <UApp>
    <AppTopNav v-if="showTopNav" />
    <NuxtPage />
  </UApp>
</template>
