<script setup lang="ts">
import type { Ref } from 'vue'
import type { CameraOption } from '~/composables/useCamera'

const props = defineProps<{
  videoRef: Ref<HTMLVideoElement | null>
  devices: CameraOption[]
  selectedDeviceId: string
  isRecognizing: boolean
  cameraLoading: boolean
  cameraError?: string
  streamError?: string
}>()

const emit = defineEmits<{
  (event: 'update:selectedDeviceId', value: string): void
  (event: 'start'): void
  (event: 'stop'): void
  (event: 'refresh'): void
}>()

const localVideo = ref<HTMLVideoElement | null>(null)

watch(localVideo, (video) => {
  props.videoRef.value = video
}, { immediate: true })

onBeforeUnmount(() => {
  if (props.videoRef.value === localVideo.value) {
    props.videoRef.value = null
  }
})

const hasDevices = computed(() => props.devices.length > 0)
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            实时识别
          </h2>
          <p class="mt-1 text-xs text-muted">
            通过网关 WebSocket 将视频帧发送到识别服务。
          </p>
        </div>
        <UBadge :color="isRecognizing ? 'success' : 'neutral'" variant="soft">
          {{ isRecognizing ? '识别中' : '空闲' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_auto_auto]">
        <USelect
          :model-value="selectedDeviceId"
          :items="devices"
          value-key="value"
          label-key="label"
          icon="i-lucide-camera"
          :disabled="cameraLoading || !hasDevices"
          placeholder="选择摄像头"
          @update:model-value="(value) => emit('update:selectedDeviceId', value as string)"
        />

        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="cameraLoading"
          label="刷新设备"
          @click="emit('refresh')"
        />

        <UButton
          v-if="!isRecognizing"
          color="primary"
          icon="i-lucide-play"
          :loading="cameraLoading"
          :disabled="!hasDevices"
          label="开始识别"
          @click="emit('start')"
        />
        <UButton
          v-else
          color="error"
          variant="outline"
          icon="i-lucide-square"
          label="停止识别"
          @click="emit('stop')"
        />
      </div>

      <UAlert
        v-if="!hasDevices"
        color="warning"
        variant="subtle"
        icon="i-lucide-camera-off"
        title="未检测到摄像头"
        description="请连接摄像头后刷新设备列表。"
      />

      <UAlert
        v-if="cameraError"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="摄像头异常"
        :description="cameraError"
      />

      <UAlert
        v-if="streamError"
        color="error"
        variant="subtle"
        icon="i-lucide-wifi-off"
        title="识别流异常"
      >
        <template #description>
          <div class="space-y-2">
            <p class="text-sm">
              {{ streamError }}
            </p>
            <UButton
              v-if="hasDevices && !isRecognizing"
              size="xs"
              color="error"
              variant="outline"
              icon="i-lucide-rotate-cw"
              label="重试连接"
              @click="emit('start')"
            />
          </div>
        </template>
      </UAlert>

      <div class="overflow-hidden rounded-lg border border-accented bg-black">
        <video
          ref="localVideo"
          autoplay
          muted
          playsinline
          class="aspect-video w-full object-cover"
        />
      </div>
    </div>
  </UCard>
</template>
