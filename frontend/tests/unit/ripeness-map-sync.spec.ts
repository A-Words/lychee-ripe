import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { RIPENESS_CLASSES, RIPENESS_COLOR_MAP } from '../../app/constants/ripeness'

interface SharedRipeness {
  classes: string[]
  color_map: Record<string, string>
}

describe('ripeness mapping sync', () => {
  it('keeps classes and colors aligned with shared/constants/ripeness.json', () => {
    const source = readFileSync(new URL('../../../shared/constants/ripeness.json', import.meta.url), 'utf-8')
    const shared = JSON.parse(source) as SharedRipeness

    expect([...RIPENESS_CLASSES]).toEqual(shared.classes)
    expect(RIPENESS_COLOR_MAP).toEqual(shared.color_map)
  })
})
