import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import BatchCreateResult from '../../app/components/batch/BatchCreateResult.vue'
import { buildTracePathWithFrom } from '../../app/utils/trace-from'
import { buildBatch } from './support/fixtures'
import { flushUi, installClipboardMock } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const stubs = createNuxtUiStubs()

describe('batch create result', () => {
  it.each([
    ['anchored', '上链成功'],
    ['pending_anchor', '已保存待补链'],
    ['anchor_failed', '补链失败']
  ] as const)('renders the expected status badge for %s batches', async (status, label) => {
    const wrapper = await mountResult({
      batch: buildBatch({
        status,
        anchor_proof: status === 'anchored' ? buildBatch().anchor_proof : null
      })
    })

    expect(wrapper.text()).toContain(label)
  })

  it('copies the full trace url for the created batch', async () => {
    const { writeText } = installClipboardMock()
    const batch = buildBatch()
    const wrapper = await mountResult({
      batch
    })

    await findButton(wrapper, '复制溯源链接').trigger('click')
    await flushUi()

    expect(writeText).toHaveBeenCalledWith(
      new URL(buildTracePathWithFrom(batch.trace_code, 'batch_create'), window.location.origin).toString()
    )
    expect(wrapper.text()).toContain('已复制')
  })

  it('shows anchor proof only when it exists', async () => {
    const wrapper = await mountResult({
      batch: buildBatch({
        status: 'pending_anchor',
        anchor_proof: null
      })
    })

    expect(wrapper.text()).not.toContain('链上交易哈希')
  })

  it('emits continue when the user wants to create another batch', async () => {
    const wrapper = await mountResult()

    await findButton(wrapper, '继续建批').trigger('click')

    expect(wrapper.emitted('continue')).toHaveLength(1)
  })
})

async function mountResult(overrides: Partial<InstanceType<typeof BatchCreateResult>['$props']> = {}) {
  return await mountSuspended(BatchCreateResult, {
    props: {
      batch: buildBatch(),
      statusCode: 201,
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
