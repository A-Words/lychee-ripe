import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { RIPENESS_CLASSES, RIPENESS_COLOR_MAP } from '../app/constants/ripeness'

describe('ripeness mapping', () => {
  it('matches shared constants definition', () => {
    const currentDir = fileURLToPath(new URL('.', import.meta.url))
    const sharedPath = resolve(currentDir, '../../shared/constants/ripeness.json')
    const shared = JSON.parse(readFileSync(sharedPath, 'utf-8')) as {
      classes: string[]
      color_map: Record<string, string>
    }

    expect(RIPENESS_CLASSES).toEqual(shared.classes)
    expect(RIPENESS_COLOR_MAP).toEqual(shared.color_map)
  })
})
