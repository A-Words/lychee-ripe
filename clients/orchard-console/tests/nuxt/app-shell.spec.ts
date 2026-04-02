import { describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import App from '../../app/app.vue'
import { createNuxtUiStubs } from './support/ui-stubs'

const stubs = createNuxtUiStubs({
  NuxtPage: defineComponent({
    template: '<div data-testid="nuxt-page" />'
  })
})

describe('app shell navigation', () => {
  it('shows top nav and highlights dashboard on dashboard routes', async () => {
    const wrapper = await mountApp('/dashboard')
    const nav = wrapper.get('[data-orientation="horizontal"]')

    expect(wrapper.text()).toContain('Lychee Ripe')
    expect(nav.get('[data-key="dashboard"]').attributes('data-active')).toBe('true')
    expect(nav.get('[data-key="batch_create"]').attributes('data-active')).toBe('false')
    expect(nav.get('[data-key="trace"]').attributes('data-active')).toBe('false')
  })

  it('shows top nav and highlights batch create on batch routes', async () => {
    const wrapper = await mountApp('/batch/create')
    const nav = wrapper.get('[data-orientation="horizontal"]')

    expect(nav.get('[data-key="batch_create"]').attributes('data-active')).toBe('true')
    expect(nav.get('[data-key="dashboard"]').attributes('data-active')).toBe('false')
  })

  it('hides top nav for public trace routes', async () => {
    const wrapper = await mountApp('/trace/TRC-9A7X-11QF')

    expect(wrapper.find('[data-stub="UHeader"]').exists()).toBe(false)
  })

  it('shows top nav for internal trace routes and highlights trace', async () => {
    const wrapper = await mountApp('/trace/TRC-9A7X-11QF?from=dashboard')
    const nav = wrapper.get('[data-orientation="horizontal"]')

    expect(wrapper.find('[data-stub="UHeader"]').exists()).toBe(true)
    expect(nav.get('[data-key="trace"]').attributes('data-active')).toBe('true')
    expect(nav.get('[data-key="trace"]').attributes('href')).toBe('/trace?from=index')
  })
})

async function mountApp(route: string) {
  return await mountSuspended(App, {
    route,
    global: {
      stubs
    }
  })
}
