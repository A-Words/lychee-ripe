import { describe, expect, it } from 'vitest'
import { ORCHARD_PRESETS } from '../../app/constants/orchard-presets'

describe('orchard presets', () => {
  it('contains non-empty orchards and plots', () => {
    expect(ORCHARD_PRESETS.length).toBeGreaterThan(0)
    for (const orchard of ORCHARD_PRESETS) {
      expect(orchard.orchard_id).toBeTruthy()
      expect(orchard.orchard_name).toBeTruthy()
      expect(orchard.plots.length).toBeGreaterThan(0)
    }
  })

  it('keeps orchard ids unique', () => {
    const ids = ORCHARD_PRESETS.map((item) => item.orchard_id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})
