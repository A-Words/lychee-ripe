const defaultFrontendBase = process.env.FRONTEND_BASE ?? 'http://localhost:3000'
const defaultGatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'

async function request(url, { method = 'GET', headers, body, expectedStatus, expectedType } = {}) {
  const response = await fetch(url, { method, headers, body })
  const contentType = response.headers.get('content-type') ?? ''

  if (expectedStatus !== undefined && response.status !== expectedStatus) {
    throw new Error(`Expected ${method} ${url} to return ${expectedStatus}, got ${response.status}`)
  }

  if (expectedStatus === undefined && !response.ok) {
    throw new Error(`Expected ${method} ${url} to return 2xx, got ${response.status}`)
  }

  if (expectedType && !contentType.includes(expectedType)) {
    throw new Error(`Expected ${method} ${url} content-type to include ${expectedType}, got ${contentType}`)
  }

  return response
}

async function requestJson(url, options = {}) {
  const response = await request(url, {
    ...options,
    expectedType: options.expectedType ?? 'application/json'
  })
  return response.json()
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

export async function runStackSmoke({
  frontendBase = defaultFrontendBase,
  gatewayBase = defaultGatewayBase
} = {}) {
  await request(`${frontendBase}/`, { expectedType: 'text/html' })

  const gatewayHealth = await requestJson(`${gatewayBase}/healthz`)
  assert(gatewayHealth.status === 'ok', `Expected gateway /healthz status ok, got ${gatewayHealth.status}`)

  const inferenceHealth = await requestJson(`${gatewayBase}/v1/health`)
  assert(inferenceHealth.status === 'ok', `Expected gateway /v1/health status ok, got ${inferenceHealth.status}`)

  const uniqueSuffix = Date.now().toString(36)
  const batchPayload = {
    orchard_id: `stack-orchard-${uniqueSuffix}`,
    orchard_name: `Stack Orchard ${uniqueSuffix}`,
    plot_id: `stack-plot-${uniqueSuffix}`,
    plot_name: `Stack Plot ${uniqueSuffix}`,
    harvested_at: '2026-04-17T10:30:00Z',
    summary: {
      total: 10,
      green: 1,
      half: 3,
      red: 6,
      young: 0
    },
    note: 'stack smoke batch',
    confirm_unripe: false
  }

  const createdBatch = await requestJson(`${gatewayBase}/v1/batches`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(batchPayload),
    expectedStatus: 201
  })

  assert(typeof createdBatch.batch_id === 'string' && createdBatch.batch_id.length > 0, 'Expected created batch_id')
  assert(typeof createdBatch.trace_code === 'string' && createdBatch.trace_code.length > 0, 'Expected created trace_code')
  assert(createdBatch.trace_mode === 'database', `Expected created trace_mode database, got ${createdBatch.trace_mode}`)
  assert(createdBatch.status === 'stored', `Expected created batch status stored, got ${createdBatch.status}`)
  assert(createdBatch.anchor_proof == null, 'Expected database-mode batch to omit anchor_proof')

  const fetchedBatch = await requestJson(`${gatewayBase}/v1/batches/${createdBatch.batch_id}`)
  assert(fetchedBatch.batch_id === createdBatch.batch_id, 'Expected fetched batch_id to match created batch_id')
  assert(fetchedBatch.trace_code === createdBatch.trace_code, 'Expected fetched trace_code to match created trace_code')
  assert(fetchedBatch.trace_mode === 'database', `Expected fetched trace_mode database, got ${fetchedBatch.trace_mode}`)
  assert(fetchedBatch.summary?.total === 10, `Expected fetched summary.total 10, got ${fetchedBatch.summary?.total}`)
  assert(fetchedBatch.summary?.unripe_count === 1, `Expected fetched summary.unripe_count 1, got ${fetchedBatch.summary?.unripe_count}`)

  const publicTrace = await requestJson(`${gatewayBase}/v1/trace/${createdBatch.trace_code}`)
  assert(publicTrace.batch?.batch_id === createdBatch.batch_id, 'Expected trace batch_id to match created batch_id')
  assert(publicTrace.batch?.trace_code === createdBatch.trace_code, 'Expected trace_code to match created batch')
  assert(publicTrace.batch?.trace_mode === 'database', `Expected trace_mode database, got ${publicTrace.batch?.trace_mode}`)
  assert(
    publicTrace.verify_result?.verify_status === 'recorded',
    `Expected verify_status recorded, got ${publicTrace.verify_result?.verify_status}`
  )
  assert(
    publicTrace.verify_result?.reason === 'batch is recorded in gateway database',
    `Expected recorded verify reason, got ${publicTrace.verify_result?.reason}`
  )
}

if (import.meta.main) {
  await runStackSmoke()
  console.log('Stack smoke passed for frontend -> gateway -> api URLs.')
}
