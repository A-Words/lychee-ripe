import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import BatchCreateForm from '../../app/components/batch/BatchCreateForm.vue'
import { toRFC3339FromLocal } from '../../app/composables/useBatchCreate'
import type { OrchardWithPlots } from '../../app/types/resources'
import { buildSessionSummary } from './support/fixtures'
import { flushUi } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const stubs = createNuxtUiStubs()

describe('batch create form', () => {
  it('renders summary values and disables submit when there is no aggregate data', async () => {
    const wrapper = await mountForm({
      summary: buildSessionSummary({
        total: 0,
        green: 0,
        half: 0,
        red: 0,
        young: 0,
        unripe_count: 0,
        unripe_ratio: 0
      })
    })

    expect(wrapper.text()).toContain('总数：0')
    expect(wrapper.text()).toContain('青果：0')
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
  })

  it('syncs orchard and plot presets into the editable form fields', async () => {
    const wrapper = await mountForm()

    await getField(wrapper, 'orchardPresetId').get('select').setValue('orchard-east-02')
    await flushUi()

    expect(getField(wrapper, 'orchard_id').get('input').element.value).toBe('orchard-east-02')
    expect(getField(wrapper, 'orchard_name').get('input').element.value).toBe('东麓果园')
    expect(getField(wrapper, 'plotPresetId').get('select').element.value).toBe('plot-e01')
    expect(getField(wrapper, 'plot_id').get('input').element.value).toBe('plot-e01')
    expect(getField(wrapper, 'plot_name').get('input').element.value).toBe('东坡 1 号地块')

    await getField(wrapper, 'plotPresetId').get('select').setValue('plot-e02')
    await flushUi()

    expect(getField(wrapper, 'plot_id').get('input').element.value).toBe('plot-e02')
    expect(getField(wrapper, 'plot_name').get('input').element.value).toBe('东坡 2 号地块')
  })

  it('blocks submit until unripe confirmation is checked', async () => {
    const wrapper = await mountForm({
      requireConfirmUnripe: true
    })

    await wrapper.get('form').trigger('submit')
    await flushUi()
    expect(wrapper.emitted('submit')).toBeUndefined()

    await getField(wrapper, 'confirm_unripe').get('input[type="checkbox"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushUi()

    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('emits a normalized payload when the form is valid', async () => {
    const wrapper = await mountForm()

    await getField(wrapper, 'orchard_id').get('input').setValue(' orchard-custom ')
    await getField(wrapper, 'orchard_name').get('input').setValue(' 自定义果园 ')
    await getField(wrapper, 'plot_id').get('input').setValue(' plot-custom ')
    await getField(wrapper, 'plot_name').get('input').setValue(' 自定义地块 ')
    await getField(wrapper, 'harvested_at').get('input').setValue('2026-04-02T08:15')
    await getField(wrapper, 'note').get('textarea').setValue('  首次采摘  ')

    await wrapper.get('form').trigger('submit')
    await flushUi()

    expect(wrapper.emitted('submit')).toEqual([[
      {
        orchard_id: 'orchard-custom',
        orchard_name: '自定义果园',
        plot_id: 'plot-custom',
        plot_name: '自定义地块',
        harvested_at: toRFC3339FromLocal('2026-04-02T08:15'),
        note: '首次采摘',
        confirm_unripe: false
      }
    ]])
  })
})

async function mountForm(overrides: Partial<InstanceType<typeof BatchCreateForm>['$props']> = {}) {
  return await mountSuspended(BatchCreateForm, {
    props: {
      orchards: ORCHARDS,
      summary: buildSessionSummary(),
      submitting: false,
      isRecognizing: false,
      requireConfirmUnripe: false,
      apiError: null,
      ...overrides
    },
    global: {
      stubs
    }
  })
}

function getField(wrapper: Awaited<ReturnType<typeof mountSuspended>>, name: string) {
  return wrapper.get(`[data-field-name="${name}"]`)
}

const ORCHARDS: OrchardWithPlots[] = [
  {
    orchard_id: 'orchard-demo-01',
    orchard_name: '荔枝示范园',
    plots: [
      { plot_id: 'plot-a01', plot_name: 'A1 区' },
      { plot_id: 'plot-a02', plot_name: 'A2 区' }
    ]
  },
  {
    orchard_id: 'orchard-east-02',
    orchard_name: '东麓果园',
    plots: [
      { plot_id: 'plot-e01', plot_name: '东坡 1 号地块' },
      { plot_id: 'plot-e02', plot_name: '东坡 2 号地块' }
    ]
  }
]
