import { describe, expect, it } from 'vitest'
import { DASHBOARD_STATUS_META } from '../../app/constants/dashboard-status'
import {
  buildRipenessBarOption,
  buildStatusDonutOption
} from '../../app/utils/dashboard-chart-options'

describe('dashboard chart options', () => {
  it('builds status donut chart data in fixed order', () => {
    const option = buildStatusDonutOption({
      anchored: 7,
      pending_anchor: 2,
      anchor_failed: 1
    }) as any

    expect(option.series[0].type).toBe('pie')
    expect(option.series[0].data.map((item: any) => item.value)).toEqual([7, 2, 1])
    expect(option.series[0].data[0].name).toBe(DASHBOARD_STATUS_META.anchored.label)
    expect(option.series[0].data[0].itemStyle.color).toBe(DASHBOARD_STATUS_META.anchored.chartColor)
  })

  it('builds ripeness bar chart with mapped labels and values', () => {
    const option = buildRipenessBarOption({
      green: 10,
      half: 20,
      red: 30,
      young: 5
    }) as any

    expect(option.series[0].type).toBe('bar')
    expect(option.xAxis.data).toEqual(['青果', '半熟', '红果', '嫩果'])
    expect(option.series[0].data).toEqual([10, 20, 30, 5])
  })
})
