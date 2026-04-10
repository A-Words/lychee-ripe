import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import DashboardRipenessChart from '../../app/components/dashboard/DashboardRipenessChart.vue'
import DashboardStatusChart from '../../app/components/dashboard/DashboardStatusChart.vue'
import { createNuxtUiStubs } from './support/ui-stubs'

const stubs = createNuxtUiStubs()

describe('dashboard chart components', () => {
  it('wraps the status chart in a fixed-height container and lets VChart fill it', async () => {
    const wrapper = await mountSuspended(DashboardStatusChart, {
      props: {
        traceMode: 'database',
        statusDistribution: {
          stored: 3
        }
      },
      global: {
        stubs
      }
    })

    const chartWrapper = wrapper.findAll('div').find((candidate) =>
      candidate.classes().includes('h-72') && candidate.classes().includes('w-full')
    )
    const chart = wrapper.get('[data-stub="VChart"]')

    expect(chartWrapper).toBeTruthy()
    expect(chart.classes()).toContain('h-full')
    expect(chart.classes()).toContain('w-full')
    expect(chart.classes()).not.toContain('h-72')
  })

  it('wraps the ripeness chart in a fixed-height container and lets VChart fill it', async () => {
    const wrapper = await mountSuspended(DashboardRipenessChart, {
      props: {
        ripenessDistribution: {
          green: 1,
          half: 2,
          red: 3,
          young: 4
        }
      },
      global: {
        stubs
      }
    })

    const chartWrapper = wrapper.findAll('div').find((candidate) =>
      candidate.classes().includes('h-72') && candidate.classes().includes('w-full')
    )
    const chart = wrapper.get('[data-stub="VChart"]')

    expect(chartWrapper).toBeTruthy()
    expect(chart.classes()).toContain('h-full')
    expect(chart.classes()).toContain('w-full')
    expect(chart.classes()).not.toContain('h-72')
  })
})
