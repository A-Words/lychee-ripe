import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import TraceLandingForm from '../../app/components/trace/TraceLandingForm.vue'
import { flushUi, createDeferred } from './support/helpers'
import { createNuxtUiStubs } from './support/ui-stubs'

const { navigateToMock } = vi.hoisted(() => ({
  navigateToMock: vi.fn()
}))

mockNuxtImport('navigateTo', () => navigateToMock)

const stubs = createNuxtUiStubs()

describe('trace landing form', () => {
  beforeEach(() => {
    navigateToMock.mockReset()
  })

  it('shows validation feedback when the trace code is empty', async () => {
    const wrapper = await mountForm('/trace')

    await wrapper.get('form').trigger('submit')
    await flushUi()

    expect(wrapper.text()).toContain('请输入溯源码。')
    expect(navigateToMock).not.toHaveBeenCalled()
  })

  it('normalizes trace code and preserves internal from query when navigating', async () => {
    navigateToMock.mockResolvedValue(undefined)
    const wrapper = await mountForm('/trace?from=index')

    await wrapper.get('#trace-code-input').setValue(' trc-9a7x-11qf ')
    await wrapper.get('form').trigger('submit')
    await flushUi()

    expect(navigateToMock).toHaveBeenCalledWith('/trace/TRC-9A7X-11QF?from=index')
  })

  it('prevents duplicate submits while navigation is pending', async () => {
    const deferred = createDeferred<void>()
    navigateToMock.mockReturnValue(deferred.promise)
    const wrapper = await mountForm('/trace?from=dashboard')

    await wrapper.get('#trace-code-input').setValue('trc-9a7x-11qf')
    await wrapper.get('form').trigger('submit')
    await flushUi()

    const submitButton = wrapper.get('button[type="submit"]')
    expect(wrapper.get('#trace-code-input').attributes('disabled')).toBeDefined()
    expect(submitButton.attributes('disabled')).toBeDefined()

    await wrapper.get('form').trigger('submit')
    expect(navigateToMock).toHaveBeenCalledTimes(1)

    deferred.resolve()
    await flushUi()
  })
})

async function mountForm(route: string) {
  return await mountSuspended(TraceLandingForm, {
    route,
    global: {
      stubs
    }
  })
}
