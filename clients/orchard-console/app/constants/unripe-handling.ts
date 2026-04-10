import type { UnripeHandling } from '~/types/batch'

export const UNRIPE_HANDLING_LABEL_MAP: Record<UnripeHandling, string> = {
  sorted_out: '已分拣'
}

export function getUnripeHandlingLabel(value: UnripeHandling): string {
  return UNRIPE_HANDLING_LABEL_MAP[value] || value
}
