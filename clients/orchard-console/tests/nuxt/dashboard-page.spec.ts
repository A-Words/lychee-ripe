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
      recent_anchors: [
        {
          batch_id: 'batch-hidden',
          trace_code: 'TRC-HIDE-0001',
          status: 'anchored',
          tx_hash: '0xdeadbeef',
          anchored_at: '2026-03-30T09:05:00.000Z',
          created_at: '2026-03-30T09:00:00.000Z'
        }
      ],
      reconcile_stats: null
    }))

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('模式：数据库')
    expect(wrapper.text()).toContain('已入库')
    expect(wrapper.text()).toContain('处理策略 已分拣')
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

  it('keeps the last successful overview visible when a later refresh fails', async () => {
    getOverviewMock
      .mockResolvedValueOnce(buildDashboardOverview())
      .mockRejectedValueOnce(new Error('refresh failed'))
    parseDashboardErrorMock.mockReturnValue({
      statusCode: 503,
      error: 'service_unavailable',
      message: '刷新数据失败，请稍后重试。',
      requestId: 'req-stale'
    })

    const wrapper = await mountDashboardPage()
    await flushUi()

    expect(wrapper.text()).toContain('批次总数')

    await findButton(wrapper, '立即刷新').trigger('click')
    await flushUi()

    expect(wrapper.text()).toContain('刷新失败，当前展示的是上一次成功加载的数据')
    expect(wrapper.text()).toContain('刷新数据失败，请稍后重试。')
    expect(wrapper.text()).toContain('req-stale')
    expect(wrapper.text()).toContain('批次总数')
    expect(wrapper.text()).not.toContain('看板服务不可用')
    wrapper.unmount()
  })

  it('backs off auto refresh after consecutive failures', async () => {
    getOverviewMock
      .mockResolvedValueOnce(buildDashboardOverview())
      .mockRejectedValueOnce(new Error('refresh failed'))
      .mockResolvedValueOnce(buildDashboardOverview())
    parseDashboardErrorMock.mockReturnValue({
      statusCode: 503,
      error: 'service_unavailable',
      message: '刷新数据失败，请稍后重试。'
    })

    const wrapper = await mountDashboardPage()
    await flushUi()

    await findButton(wrapper, '立即刷新').trigger('click')
    await flushUi()

    expect(getOverviewMock).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(30_000)
    await flushUi()
    expect(getOverviewMock).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(30_000)
    await flushUi()
    expect(getOverviewMock).toHaveBeenCalledTimes(3)
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
