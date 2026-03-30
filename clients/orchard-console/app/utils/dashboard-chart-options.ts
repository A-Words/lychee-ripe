import { DASHBOARD_STATUS_META, DASHBOARD_STATUS_ORDER } from '~/constants/dashboard-status'
import { RIPENESS_CLASSES, RIPENESS_COLOR_MAP, RIPENESS_LABEL_MAP } from '~/constants/ripeness'
import type { BatchStatusDistribution, RipenessDistribution } from '~/types/dashboard'

type ChartOption = Record<string, unknown>

export function buildStatusDonutOption(distribution: BatchStatusDistribution): ChartOption {
  const seriesData = DASHBOARD_STATUS_ORDER.map((key) => ({
    name: DASHBOARD_STATUS_META[key].label,
    value: distribution[key],
    itemStyle: {
      color: DASHBOARD_STATUS_META[key].chartColor
    }
  }))

  return {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      bottom: 0
    },
    series: [
      {
        name: '批次状态',
        type: 'pie',
        radius: ['50%', '72%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 6,
          borderWidth: 2,
          borderColor: '#fff'
        },
        label: {
          formatter: '{b}: {c}'
        },
        data: seriesData
      }
    ]
  }
}

export function buildRipenessBarOption(distribution: RipenessDistribution): ChartOption {
  const categories = RIPENESS_CLASSES.map((key) => RIPENESS_LABEL_MAP[key])
  const values = RIPENESS_CLASSES.map((key) => distribution[key])
  const colors = RIPENESS_CLASSES.map((key) => RIPENESS_COLOR_MAP[key])

  return {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: 30,
      right: 12,
      top: 16,
      bottom: 28
    },
    xAxis: {
      type: 'category',
      data: categories
    },
    yAxis: {
      type: 'value',
      minInterval: 1
    },
    series: [
      {
        name: '数量',
        type: 'bar',
        data: values,
        itemStyle: {
          color: (params: { dataIndex: number }) => colors[params.dataIndex]
        },
        barMaxWidth: 40
      }
    ]
  }
}
