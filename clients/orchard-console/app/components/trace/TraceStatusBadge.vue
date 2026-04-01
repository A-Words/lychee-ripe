<script setup lang="ts">
import { VERIFY_STATUS_META } from '~/constants/verify-status'
import type { VerifyStatus } from '~/types/trace'

const props = defineProps<{
  status: VerifyStatus
}>()

const meta = computed(() => VERIFY_STATUS_META[props.status])
</script>

<template>
  <UBadge :color="meta.color" variant="subtle" size="lg" class="font-semibold">
    {{ meta.label }}
  </UBadge>
</template>
