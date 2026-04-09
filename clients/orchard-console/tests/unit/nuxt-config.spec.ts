import { afterEach, describe, expect, it, vi } from 'vitest'

describe('nuxt config auth boundary', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('locks the app to SPA mode for localStorage-based auth', async () => {
    vi.stubGlobal('defineNuxtConfig', (config: unknown) => config)
    const { default: nuxtConfig } = await import('../../nuxt.config')

    expect(nuxtConfig.ssr).toBe(false)
  })
})
