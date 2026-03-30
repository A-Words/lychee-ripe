const frontendBase = process.env.FRONTEND_BASE ?? 'http://127.0.0.1:3000'
const gatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'

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

await assertOk(`${frontendBase}/`, 'text/html')
await assertOk(`${gatewayBase}/healthz`, 'application/json')
await assertOk(`${gatewayBase}/v1/health`, 'application/json')

console.log('Stack smoke passed for frontend -> gateway -> api URLs.')
