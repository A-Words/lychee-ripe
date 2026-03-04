import type { RipenessLabel } from '~/types/trace'

export const RIPENESS_CLASSES = ['green', 'half', 'red', 'young'] as const satisfies readonly RipenessLabel[]

export const RIPENESS_COLOR_MAP: Record<RipenessLabel, string> = {
  green: '#3D8D40',
  half: '#F5A623',
  red: '#D64545',
  young: '#6AAED6'
}

export const RIPENESS_LABEL_MAP: Record<RipenessLabel, string> = {
  green: '青果',
  half: '半熟',
  red: '红果',
  young: '嫩果'
}
