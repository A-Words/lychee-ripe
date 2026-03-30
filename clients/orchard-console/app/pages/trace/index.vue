<script setup lang="ts">
useSeoMeta({
  title: '荔枝溯源查询',
  description: '请输入溯源码，查询荔枝批次摘要与链上验签结果。'
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="mx-auto max-w-3xl space-y-6">
      <section class="space-y-3">
        <p class="text-xs uppercase tracking-widest text-muted">
          Public Trace Portal
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-4xl">
          荔枝区块链溯源查询
        </h1>
        <p class="text-sm text-toned sm:text-base">
          页面面向公众开放，支持二维码落地链接与手动输入两种查询方式。
        </p>
      </section>

      <TraceLandingForm />
    </div>
  </UContainer>
</template>
