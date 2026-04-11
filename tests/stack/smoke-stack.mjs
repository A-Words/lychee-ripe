const defaultFrontendBase = process.env.FRONTEND_BASE ?? 'http://localhost:3000'
const defaultGatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'

async function assertOk(url, expectedType) {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`Expected ${url} to return 2xx, got ${response.status}`)
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (expectedType && !contentType.includes(expectedType)) {
    throw new Error(`Expected ${url} content-type to include ${expectedType}, got ${contentType}`)
  }
}

export async function runStackSmoke({
  frontendBase = defaultFrontendBase,
  gatewayBase = defaultGatewayBase
} = {}) {
  await assertOk(`${frontendBase}/`, 'text/html')
  await assertOk(`${gatewayBase}/healthz`, 'application/json')
  await assertOk(`${gatewayBase}/v1/health`, 'application/json')
}

if (import.meta.main) {
  await runStackSmoke()
  console.log('Stack smoke passed for frontend -> gateway -> api URLs.')
}
