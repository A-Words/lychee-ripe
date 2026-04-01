import { describe, expect, it } from 'vitest'
import { mapBatchErrorMessage } from '../../app/composables/useBatchCreate'

describe('batch error mapping', () => {
  it('maps auth-related errors to fixed hint', () => {
    const message = mapBatchErrorMessage(401, '')
    expect(message).toContain('未传 API Key')
  })

  it('keeps backend message for 400 validation failure', () => {
    const message = mapBatchErrorMessage(400, 'confirm_unripe must be true when unripe_ratio > 0.15')
    expect(message).toContain('confirm_unripe')
  })

  it('provides fallback for service unavailable', () => {
    const message = mapBatchErrorMessage(503, '')
    expect(message).toContain('服务暂不可用')
  })
})
