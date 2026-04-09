import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import AdminPage from '../../app/pages/admin.vue'
import { flushUi } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const {
  initMock,
  listOrchardsMock,
  listPlotsMock,
  listUsersMock,
  updateOrchardMock,
  updatePlotMock,
  updateUserMock
} = vi.hoisted(() => ({
  initMock: vi.fn(),
  listOrchardsMock: vi.fn(),
  listPlotsMock: vi.fn(),
  listUsersMock: vi.fn(),
  updateOrchardMock: vi.fn(),
  updatePlotMock: vi.fn(),
  updateUserMock: vi.fn()
}))

mockNuxtImport('useAuth', () => () => ({
  init: initMock
}))

mockNuxtImport('useAdminApi', () => () => ({
  listOrchards: listOrchardsMock,
  listPlots: listPlotsMock,
  listUsers: listUsersMock,
  createOrchard: vi.fn(),
  createPlot: vi.fn(),
  createUser: vi.fn(),
  updateOrchard: updateOrchardMock,
  updatePlot: updatePlotMock,
  updateUser: updateUserMock
}))

const stubs = createNuxtUiStubs()

describe('admin page', () => {
  beforeEach(() => {
    initMock.mockReset()
    listOrchardsMock.mockReset()
    listPlotsMock.mockReset()
    listUsersMock.mockReset()
    updateOrchardMock.mockReset()
    updatePlotMock.mockReset()
    updateUserMock.mockReset()

    initMock.mockResolvedValue(undefined)
    listOrchardsMock.mockResolvedValue([
      {
        orchard_id: 'orchard-1',
        orchard_name: '示例果园',
        status: 'active',
        created_at: '2026-04-01T00:00:00.000Z',
        updated_at: '2026-04-01T00:00:00.000Z'
      }
    ])
    listPlotsMock.mockResolvedValue([
      {
        plot_id: 'plot-1',
        orchard_id: 'orchard-1',
        plot_name: 'A1 区',
        status: 'active',
        created_at: '2026-04-01T00:00:00.000Z',
        updated_at: '2026-04-01T00:00:00.000Z'
      }
    ])
    listUsersMock.mockResolvedValue([
      {
        id: 'user-1',
        email: 'user@example.com',
        display_name: '普通用户',
        role: 'operator',
        status: 'active',
        created_at: '2026-04-01T00:00:00.000Z',
        updated_at: '2026-04-01T00:00:00.000Z'
      }
    ])
    updateOrchardMock.mockResolvedValue(undefined)
    updatePlotMock.mockResolvedValue(undefined)
    updateUserMock.mockResolvedValue(undefined)
  })

  it('renders editable rows for existing orchards, plots, and users', async () => {
    const wrapper = await mountAdminPage()
    await flushUi()

    const cards = wrapper.findAll('[data-stub="UCard"]')
    const orchardCard = cards[3]
    const plotCard = cards[4]
    const userCard = cards[5]

    expect(orchardCard.text()).toContain('orchard-1')
    expect(orchardCard.find('input[placeholder="果园名称"]').exists()).toBe(true)
    expect(countLabeledButtons(orchardCard, '保存')).toBe(1)

    expect(plotCard.text()).toContain('plot-1')
    expect(plotCard.find('input[placeholder="地块名称"]').exists()).toBe(true)
    expect(plotCard.find('input[placeholder="所属果园 ID"]').exists()).toBe(true)
    expect(countLabeledButtons(plotCard, '保存')).toBe(1)

    expect(userCard.text()).toContain('user-1')
    expect(userCard.find('input[placeholder="显示名"]').exists()).toBe(true)
    expect(userCard.find('input[placeholder="user@example.com"]').exists()).toBe(true)
    expect(userCard.text()).toContain('管理员')
    expect(userCard.text()).toContain('普通用户')
    expect(countLabeledButtons(userCard, '保存')).toBe(1)
  })

  it('persists edited user role and status through the save action', async () => {
    const wrapper = await mountAdminPage()
    await flushUi()

    const userCard = wrapper.findAll('[data-stub="UCard"]')[5]
    const inputs = userCard.findAll('input')
    const selects = userCard.findAll('select')

    await inputs[0].setValue('管理员用户')
    await inputs[1].setValue('admin@example.com')
    await selects[0].setValue('admin')
    await selects[1].setValue('disabled')
    await findButton(userCard, '保存').trigger('click')
    await flushUi()

    expect(updateUserMock).toHaveBeenCalledWith('user-1', {
      email: 'admin@example.com',
      display_name: '管理员用户',
      role: 'admin',
      status: 'disabled'
    })
  })
})

async function mountAdminPage() {
  return await mountSuspended(AdminPage, {
    global: {
      stubs
    }
  })
}

function countLabeledButtons(wrapper: { findAll: (selector: string) => any[] }, label: string) {
  return wrapper.findAll('button').filter((candidate: any) => candidate.text().includes(label)).length
}

function findButton(wrapper: { findAll: (selector: string) => any[] }, label: string) {
  const target = wrapper.findAll('button').find((candidate: any) => candidate.text().includes(label))
  if (!target) {
    throw new Error(`Button not found: ${label}`)
  }
  return target
}
