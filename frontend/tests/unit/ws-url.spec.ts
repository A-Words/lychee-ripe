import { describe, expect, it } from 'vitest'
import { toWebSocketBase } from '../../app/utils/ws-url'

describe('ws url transformer', () => {
  it('converts http scheme to ws', () => {
    expect(toWebSocketBase('http://127.0.0.1:9000')).toBe('ws://127.0.0.1:9000')
  })

  it('converts https scheme to wss', () => {
    expect(toWebSocketBase('https://example.com')).toBe('wss://example.com')
  })

  it('keeps ws scheme unchanged', () => {
    expect(toWebSocketBase('ws://localhost:9000/')).toBe('ws://localhost:9000')
  })
})
