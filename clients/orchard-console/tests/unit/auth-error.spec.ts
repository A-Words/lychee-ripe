import { describe, expect, it } from 'vitest'

import { resolveAuthErrorMessage } from '../../app/utils/auth-error'

describe('resolveAuthErrorMessage', () => {
  it('maps invalid request failures to a retryable login message', () => {
    expect(resolveAuthErrorMessage('invalid_request')).toContain('重新发起登录')
  })

  it('maps provider denial to a user-facing message', () => {
    expect(resolveAuthErrorMessage('access_denied')).toContain('身份提供方拒绝')
  })

  it('ignores unknown codes', () => {
    expect(resolveAuthErrorMessage('unknown_error')).toBe('')
  })
})
