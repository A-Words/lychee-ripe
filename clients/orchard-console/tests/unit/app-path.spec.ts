import { describe, expect, it } from 'vitest'
import { buildAppPath, inferAppBasePath } from '../../app/utils/app-path'

describe('app path helpers', () => {
  it('infers app base path from the current browser location', () => {
    expect(inferAppBasePath('/console/dashboard', '/dashboard')).toBe('/console')
    expect(inferAppBasePath('/console/admin', '/admin')).toBe('/console')
    expect(inferAppBasePath('/dashboard', '/dashboard')).toBe('')
  })

  it('prefers the candidate route that matches the current browser location', () => {
    expect(inferAppBasePath('/console/dashboard', ['/admin', '/dashboard'])).toBe('/console')
    expect(inferAppBasePath('/console/admin', ['/dashboard', '/admin'])).toBe('/console')
  })

  it('does not let the root route candidate override a more specific destination route', () => {
    expect(inferAppBasePath('/console/dashboard', ['/dashboard', '/'])).toBe('/console')
  })

  it('infers base path for the app root route', () => {
    expect(inferAppBasePath('/console', '/')).toBe('/console')
    expect(inferAppBasePath('/console/', '/')).toBe('/console')
    expect(inferAppBasePath('/', '/')).toBe('')
  })

  it('builds app-local paths under the inferred base path', () => {
    expect(buildAppPath('/console', '/login')).toBe('/console/login')
    expect(buildAppPath('/console', '/dashboard?tab=recent')).toBe('/console/dashboard?tab=recent')
    expect(buildAppPath('', '/admin')).toBe('/admin')
  })
})
