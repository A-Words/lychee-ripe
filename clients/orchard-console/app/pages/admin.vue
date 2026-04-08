<script setup lang="ts">
import type { UserRole, UserStatus } from '~/types/auth'
import type { Orchard, Plot, ResourceStatus, UserRecord } from '~/types/resources'

useSeoMeta({
  title: '管理后台',
  description: '维护果园、地块和用户角色。'
})

const auth = useAuth()
const api = useAdminApi()

const loading = ref(false)
const errorMessage = ref('')
const orchards = ref<Orchard[]>([])
const plots = ref<Plot[]>([])
const users = ref<UserRecord[]>([])

const orchardForm = reactive({
  orchard_id: '',
  orchard_name: ''
})

const plotForm = reactive({
  plot_id: '',
  orchard_id: '',
  plot_name: ''
})

const userForm = reactive({
  email: '',
  display_name: '',
  role: 'operator' as UserRole,
  status: 'active' as UserStatus
})

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    await auth.init()
    const [orchardItems, plotItems, userItems] = await Promise.all([
      api.listOrchards(true),
      api.listPlots(undefined, true),
      api.listUsers()
    ])
    orchards.value = orchardItems
    plots.value = plotItems
    users.value = userItems
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载管理数据失败'
  } finally {
    loading.value = false
  }
}

async function handleCreateOrchard() {
  await api.createOrchard({
    orchard_id: orchardForm.orchard_id,
    orchard_name: orchardForm.orchard_name
  })
  orchardForm.orchard_id = ''
  orchardForm.orchard_name = ''
  await loadAll()
}

async function handleCreatePlot() {
  await api.createPlot({
    plot_id: plotForm.plot_id,
    orchard_id: plotForm.orchard_id,
    plot_name: plotForm.plot_name
  })
  plotForm.plot_id = ''
  plotForm.orchard_id = ''
  plotForm.plot_name = ''
  await loadAll()
}

async function handleCreateUser() {
  await api.createUser({
    email: userForm.email,
    display_name: userForm.display_name,
    role: userForm.role,
    status: userForm.status
  })
  userForm.email = ''
  userForm.display_name = ''
  userForm.role = 'operator'
  userForm.status = 'active'
  await loadAll()
}

async function toggleOrchardStatus(item: Orchard) {
  const nextStatus: ResourceStatus = item.status === 'active' ? 'archived' : 'active'
  await api.updateOrchard(item.orchard_id, {
    orchard_name: item.orchard_name,
    status: nextStatus
  })
  await loadAll()
}

async function togglePlotStatus(item: Plot) {
  const nextStatus: ResourceStatus = item.status === 'active' ? 'archived' : 'active'
  await api.updatePlot(item.plot_id, {
    orchard_id: item.orchard_id,
    plot_name: item.plot_name,
    status: nextStatus
  })
  await loadAll()
}

async function toggleUserStatus(item: UserRecord) {
  const nextStatus: UserStatus = item.status === 'active' ? 'disabled' : 'active'
  await api.updateUser(item.id, {
    email: item.email,
    display_name: item.display_name,
    role: item.role,
    status: nextStatus
  })
  await loadAll()
}

onMounted(() => {
  void loadAll()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          Admin
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          管理后台
        </h1>
        <p class="text-sm text-toned sm:text-base">
          维护果园、地块与系统用户角色。认证关闭时此页面默认以管理员身份可用。
        </p>
      </section>

      <UAlert
        v-if="errorMessage"
        color="error"
        variant="subtle"
        icon="i-lucide-alert-circle"
        :description="errorMessage"
      />

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <UCard variant="outline" :ui="{ body: 'p-5 space-y-4' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              新增果园
            </h2>
          </template>
          <UInput v-model="orchardForm.orchard_id" placeholder="orchard-id" />
          <UInput v-model="orchardForm.orchard_name" placeholder="果园名称" />
          <UButton block label="创建果园" :loading="loading" @click="handleCreateOrchard" />
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-5 space-y-4' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              新增地块
            </h2>
          </template>
          <UInput v-model="plotForm.plot_id" placeholder="plot-id" />
          <UInput v-model="plotForm.orchard_id" placeholder="所属果园 ID" />
          <UInput v-model="plotForm.plot_name" placeholder="地块名称" />
          <UButton block label="创建地块" :loading="loading" @click="handleCreatePlot" />
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-5 space-y-4' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              预创建用户
            </h2>
          </template>
          <UInput v-model="userForm.email" placeholder="user@example.com" />
          <UInput v-model="userForm.display_name" placeholder="显示名" />
          <USelect
            v-model="userForm.role"
            :items="[{ label: '管理员', value: 'admin' }, { label: '普通用户', value: 'operator' }]"
            value-key="value"
            label-key="label"
          />
          <USelect
            v-model="userForm.status"
            :items="[{ label: '启用', value: 'active' }, { label: '停用', value: 'disabled' }]"
            value-key="value"
            label-key="label"
          />
          <UButton block label="创建用户" :loading="loading" @click="handleCreateUser" />
        </UCard>
      </div>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <UCard variant="outline" :ui="{ body: 'p-5 space-y-3' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              果园列表
            </h2>
          </template>
          <div v-for="item in orchards" :key="item.orchard_id" class="flex items-center justify-between gap-3 rounded border border-default p-3">
            <div>
              <p class="font-medium">{{ item.orchard_name }}</p>
              <p class="text-xs text-muted">{{ item.orchard_id }} / {{ item.status }}</p>
            </div>
            <UButton size="sm" variant="outline" :label="item.status === 'active' ? '归档' : '启用'" @click="toggleOrchardStatus(item)" />
          </div>
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-5 space-y-3' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              地块列表
            </h2>
          </template>
          <div v-for="item in plots" :key="item.plot_id" class="flex items-center justify-between gap-3 rounded border border-default p-3">
            <div>
              <p class="font-medium">{{ item.plot_name }}</p>
              <p class="text-xs text-muted">{{ item.plot_id }} / {{ item.orchard_id }} / {{ item.status }}</p>
            </div>
            <UButton size="sm" variant="outline" :label="item.status === 'active' ? '归档' : '启用'" @click="togglePlotStatus(item)" />
          </div>
        </UCard>

        <UCard variant="outline" :ui="{ body: 'p-5 space-y-3' }">
          <template #header>
            <h2 class="font-semibold text-highlighted">
              用户列表
            </h2>
          </template>
          <div v-for="item in users" :key="item.id" class="flex items-center justify-between gap-3 rounded border border-default p-3">
            <div>
              <p class="font-medium">{{ item.display_name || item.email }}</p>
              <p class="text-xs text-muted">{{ item.email }} / {{ item.role }} / {{ item.status }}</p>
            </div>
            <UButton size="sm" variant="outline" :label="item.status === 'active' ? '停用' : '启用'" @click="toggleUserStatus(item)" />
          </div>
        </UCard>
      </div>
    </div>
  </UContainer>
</template>
