import type { RipenessLabel } from '../types/infer'

// Keep this mapping aligned with shared/constants/ripeness.json.
export const RIPENESS_CLASSES: RipenessLabel[] = ['green', 'half', 'red', 'young']

export const RIPENESS_COLOR_MAP: Record<RipenessLabel, string> = {
  green: '#3D8D40',
  half: '#F5A623',
  red: '#D64545',
  young: '#6AAED6',
}

export const RIPENESS_LABEL_MAP: Record<RipenessLabel, string> = {
  green: 'Green',
  half: 'Half',
  red: 'Red',
  young: 'Young',
}
