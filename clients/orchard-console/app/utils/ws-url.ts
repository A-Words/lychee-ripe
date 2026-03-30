export function toWebSocketBase(input: string): string {
  const raw = input.trim().replace(/\/+$/, '')
  if (!raw) {
    return 'ws://127.0.0.1:9000'
  }
  if (raw.startsWith('ws://') || raw.startsWith('wss://')) {
    return raw
  }
  if (raw.startsWith('https://')) {
    return `wss://${raw.slice('https://'.length)}`
  }
  if (raw.startsWith('http://')) {
    return `ws://${raw.slice('http://'.length)}`
  }
  return `ws://${raw}`
}
