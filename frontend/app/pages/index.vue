<template>
  <UContainer class="py-10 sm:py-16">
    <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
      <div class="space-y-3">
        <p class="text-xs uppercase tracking-widest text-muted">
          Lychee Ripe
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          公众溯源查询
        </h1>
        <p class="text-sm text-toned sm:text-base">
          通过溯源码查询荔枝批次摘要与链上验签结论。
        </p>
        <div class="pt-2">
          <UButton to="/trace" icon="i-lucide-arrow-right" label="进入查询页" />
        </div>
      </div>
    </UCard>
  </UContainer>
</template>
