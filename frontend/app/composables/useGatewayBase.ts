export function useGatewayBase() {
  const config = useRuntimeConfig()

  return computed(() => {
    const raw = (config.public.gatewayBase || 'http://127.0.0.1:9000').trim()
    const normalized = raw.replace(/\/+$/, '')
    return normalized || 'http://127.0.0.1:9000'
  })
}
