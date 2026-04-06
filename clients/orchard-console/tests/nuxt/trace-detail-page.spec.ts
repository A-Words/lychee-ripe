import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import TraceDetailPage from '../../app/pages/trace/[trace_code].vue'
import { buildTraceResponse } from './support/fixtures'
import { flushUi, installClipboardMock } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const { getPublicTraceMock, parseTraceErrorMock, routeMock } = vi.hoisted(() => ({
  getPublicTraceMock: vi.fn(),
  parseTraceErrorMock: vi.fn(),
  routeMock: {
    path: '/trace/TRC-9A7X-11QF',
    params: {
      trace_code: 'TRC-9A7X-11QF'
    },
    query: {}
  }
}))

mockNuxtImport('useTraceApi', () => () => ({
  getPublicTrace: getPublicTraceMock,
  parseTraceError: parseTraceErrorMock
}))

mockNuxtImport('useRoute', () => () => routeMock)

const stubs = createNuxtUiStubs()

describe('trace detail page', () => {
  beforeEach(() => {
    getPublicTraceMock.mockReset()
    parseTraceErrorMock.mockReset()
    routeMock.path = '/trace/TRC-9A7X-11QF'
    routeMock.params = {
      trace_code: 'TRC-9A7X-11QF'
    }
    routeMock.query = {}
  })

  it('renders not found immediately when the trace code is missing', async () => {
    routeMock.path = '/trace/%20'
    routeMock.params = {
      trace_code: ' '
    }

    const wrapper = await mountTraceDetailPage()
    await flushUi()

    expect(getPublicTraceMock).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('未找到对应溯源码')
    expect(wrapper.text()).toContain('未提供溯源码')
    expect(findButton(wrapper, '复制溯源码').attributes('disabled')).toBeDefined()
  })

  it('renders trace details and enables copying when the query succeeds', async () => {
    const { writeText } = installClipboardMock()
    routeMock.query = {
      from: 'batch_create'
    }
    getPublicTraceMock.mockResolvedValue(buildTraceResponse())

    const wrapper = await mountTraceDetailPage()
    await flushUi()

    expect(getPublicTraceMock).toHaveBeenCalledWith('TRC-9A7X-11QF')
    expect(wrapper.text()).toContain('返回识别建批')
    expect(wrapper.text()).toContain('荔枝示范园 / A1 区')
    expect(wrapper.text()).toContain('验证通过')

    await findButton(wrapper, '复制溯源码').trigger('click')
    await flushUi()

    expect(writeText).toHaveBeenCalledWith('TRC-9A7X-11QF')
  })

  it('renders database storage semantics for recorded traces', async () => {
    getPublicTraceMock.mockResolvedValue(buildTraceResponse({
      batch: {
        ...buildTraceResponse().batch,
        trace_mode: 'database',
        status: 'stored'
      },
      verify_result: {
        verify_status: 'recorded',
        reason: '批次已入库并可在系统内查询'
      }
    }))

    const wrapper = await mountTraceDetailPage()
    await flushUi()

    expect(wrapper.text()).toContain('数据库存证')
    expect(wrapper.text()).not.toContain('链上可核验')
  })

  it('renders the 404 branch when the trace record does not exist', async () => {
    getPublicTraceMock.mockRejectedValue(new Error('not found'))
    parseTraceErrorMock.mockReturnValue({
      statusCode: 404,
      error: 'not_found',
      message: '未找到对应溯源码。',
      requestId: 'req-404'
    })

    const wrapper = await mountTraceDetailPage()
    await flushUi()

    expect(wrapper.text()).toContain('未找到对应溯源码')
    expect(wrapper.text()).toContain('req-404')
  })

  it('renders the unavailable branch and retries successfully', async () => {
    routeMock.query = {
      from: 'dashboard'
    }
    getPublicTraceMock
      .mockRejectedValueOnce(new Error('service down'))
      .mockResolvedValueOnce(buildTraceResponse())
    parseTraceErrorMock.mockReturnValue({
      statusCode: 503,
      error: 'service_unavailable',
      message: '网关服务暂时不可用，请稍后重试。',
      requestId: 'req-503'
    })

    const wrapper = await mountTraceDetailPage()
    await flushUi()

    expect(wrapper.text()).toContain('服务暂不可用')
    expect(wrapper.text()).toContain('返回数据看板')

    await findButton(wrapper, '重试查询').trigger('click')
    await flushUi()

    expect(getPublicTraceMock).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('公开溯源档案')
  })
})

async function mountTraceDetailPage() {
  return await mountSuspended(TraceDetailPage, {
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
