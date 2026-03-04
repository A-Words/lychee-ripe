import type { HarvestSuggestion } from '~/types/infer'

type SuggestionColor = 'success' | 'warning' | 'neutral' | 'error'

export interface HarvestSuggestionMeta {
  label: string
  description: string
  color: SuggestionColor
}

export const HARVEST_SUGGESTION_META: Record<HarvestSuggestion, HarvestSuggestionMeta> = {
  not_ready: {
    label: '暂不建议采摘',
    description: '成熟果比例不足，建议继续观察。',
    color: 'neutral'
  },
  partially_ready: {
    label: '部分可采',
    description: '已达到部分采摘窗口，建议分批处理。',
    color: 'warning'
  },
  ready: {
    label: '建议采摘',
    description: '成熟度满足要求，可执行建批。',
    color: 'success'
  },
  overripe_risk: {
    label: '过熟风险',
    description: '存在过熟风险，建议尽快分拣处理。',
    color: 'error'
  }
}
