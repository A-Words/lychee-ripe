import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import DashboardPage from '../../app/pages/dashboard/index.vue'
import { buildDashboardOverview } from './support/fixtures'
import { createDeferred, flushUi } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const { getOverviewMock, parseDashboardErrorMock } = vi.hoisted(() => ({
  getOverviewMock: vi.fn(),
  parseDashboardErrorMock: vi.fn()
}))

mockNuxtImport('useDashboardApi', () => () => ({
  getOverview: getOverviewMock,
  parseDashboardError: parseDashboardErrorMock
}))

const stubs = createNuxtUiStubs({
  DashboardStatusChart: defineComponent({
    template: '<div data-testid="status-chart" />'
  }),
  DashboardRipenessChart: defineComponent({
    template: '<div data-testid="ripeness-chart" />'
  })
})

describe('dashboard page', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    getOverviewMock.mockReset()
    parseDashboardErrorMock.mockReset()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it('renders the loading branch while overview data is pending', async () => {
    const deferred = createDeferred<ReturnType<typeof buildDashboardOverview>>()
    getOverviewMock.mockReturnValue(deferred.promise)

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.findAll('[data-stub="USkeleton"]').length).toBeGreaterThan(0)

    deferred.resolve(buildDashboardOverview())
    await flushUi()
    wrapper.unmount()
  })

  it('renders the ready branch and allows manual refresh', async () => {
    getOverviewMock.mockResolvedValue(buildDashboardOverview())

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('批次总数')
    expect(wrapper.text()).toContain('最近链上记录')
    expect(wrapper.text()).toContain('TRC-9A7X-11QF')
    expect(wrapper.get('a[href="/trace/TRC-9A7X-11QF?from=dashboard"]').text()).toContain('查看溯源')

    await findButton(wrapper, '立即刷新').trigger('click')
    await flushUi()

    expect(getOverviewMock).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('renders the empty branch when there are no batches yet', async () => {
    getOverviewMock.mockResolvedValue(buildDashboardOverview({
      totals: {
        batch_total: 0
      },
      recent_anchors: []
    }))

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('暂无批次数据')
    expect(wrapper.text()).toContain('暂无链上记录')
    wrapper.unmount()
  })

  it('hides reconcile and chain history sections in database mode', async () => {
    getOverviewMock.mockResolvedValue(buildDashboardOverview({
      trace_mode: 'database',
      status_distribution: {
        stored: 3
      },
      recent_anchors: [],
      reconcile_stats: null
    }))

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('模式：数据库')
    expect(wrapper.text()).toContain('已入库')
    expect(wrapper.text()).not.toContain('补链统计')
    expect(wrapper.text()).not.toContain('最近链上记录')
    wrapper.unmount()
  })

  it('renders the auth blocked branch for 401/403 responses', async () => {
    getOverviewMock.mockRejectedValue(new Error('auth blocked'))
    parseDashboardErrorMock.mockReturnValue({
      statusCode: 401,
      error: 'forbidden',
      message: '网关开启了鉴权，本期页面不传 API Key。',
      requestId: 'req-auth'
    })

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('网关鉴权已开启')
    expect(wrapper.text()).toContain('req-auth')
    wrapper.unmount()
  })

  it('renders the unavailable branch and retries successfully', async () => {
    getOverviewMock
      .mockRejectedValueOnce(new Error('service down'))
      .mockResolvedValueOnce(buildDashboardOverview())
    parseDashboardErrorMock.mockReturnValue({
      statusCode: 503,
      error: 'service_unavailable',
      message: '服务暂不可用，请稍后重试。',
      requestId: 'req-503'
    })

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('看板服务不可用')
    expect(wrapper.text()).toContain('req-503')

    await findButton(wrapper, '重试加载').trigger('click')
    await flushUi()

    expect(getOverviewMock).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('批次总数')
    wrapper.unmount()
  })
})

async function mountDashboardPage() {
  return await mountSuspended(DashboardPage, {
    global: {
      stubs
    }
  })
}

function findButton(wrapper: Awaited<ReturnType<typeof mountSuspended>>, label: string) {
  const target = wrapper.findAll('button').find((candidate: any) => candidate.text().includes(label))
  if (!target) {
    throw new Error(`Button not found: ${label}`)
  }
  return target
}
