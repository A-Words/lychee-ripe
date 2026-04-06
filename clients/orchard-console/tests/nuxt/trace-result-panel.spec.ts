import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import TraceResultPanel from '../../app/components/trace/TraceResultPanel.vue'
import { buildTraceResponse } from './support/fixtures'
import { flushUi, installClipboardMock } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const stubs = createNuxtUiStubs()

describe('trace result panel', () => {
  it('falls back to safe location text when orchard or plot names are missing', async () => {
    const wrapper = await mountPanel({
      trace: buildTraceResponse({
        batch: {
          ...buildTraceResponse().batch,
          orchard_name: '',
          plot_name: ''
        }
      })
    })

    expect(wrapper.text()).toContain('未知果园 / 未登记地块')
  })

  it('renders verify messaging and ripeness summary cards', async () => {
    const wrapper = await mountPanel({
      trace: buildTraceResponse({
        verify_result: {
          ...buildTraceResponse().verify_result,
          verify_status: 'pending',
          reason: '批次已保存，等待链上锚定'
        }
      })
    })

    expect(wrapper.text()).toContain('待上链')
    expect(wrapper.text()).toContain('校验说明：批次已保存，等待链上锚定')
    expect(wrapper.text()).toContain('成熟度摘要')
    expect(wrapper.text()).toContain('未成熟占比（green + young）')
  })

  it('renders recorded messaging without blockchain copy', async () => {
    const wrapper = await mountPanel({
      trace: buildTraceResponse({
        batch: {
          ...buildTraceResponse().batch,
          trace_mode: 'database',
          status: 'stored'
        },
        verify_result: {
          verify_status: 'recorded',
          reason: '批次已入库并可在系统内查询'
        }
      })
    })

    expect(wrapper.text()).toContain('数据库存证：批次已入库并可在系统内查询')
    expect(wrapper.text()).not.toContain('链上可核验')
  })

  it('copies the trace code from the panel action', async () => {
    const { writeText } = installClipboardMock()
    const wrapper = await mountPanel()

    await findButton(wrapper, '复制溯源码').trigger('click')
    await flushUi()

    expect(writeText).toHaveBeenCalledWith('TRC-9A7X-11QF')
    expect(wrapper.text()).toContain('已复制')
  })
})

async function mountPanel(overrides: Partial<InstanceType<typeof TraceResultPanel>['$props']> = {}) {
  return await mountSuspended(TraceResultPanel, {
    props: {
      trace: buildTraceResponse(),
      ...overrides
    },
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
