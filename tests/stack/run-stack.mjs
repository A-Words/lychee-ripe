import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { spawn, spawnSync } from 'node:child_process'
import { isAbsolute, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { parseDocument } from 'yaml'

import { runStackSmoke } from './smoke-stack.mjs'

const repoRoot = fileURLToPath(new URL('../../', import.meta.url))
const frontendDir = fileURLToPath(new URL('../../clients/orchard-console/', import.meta.url))
const parsed = parseArgs(process.argv.slice(2))
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'
const frontendBase = process.env.FRONTEND_BASE ?? 'http://localhost:3000'
const gatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'
const inferenceBase = process.env.INFERENCE_BASE ?? 'http://127.0.0.1:8000'
const timeoutMs = parsed.timeoutMs ?? 120_000
const frontendEndpoint = parseHttpEndpoint(frontendBase, 'FRONTEND_BASE')
const gatewayEndpoint = parseHttpEndpoint(gatewayBase, 'GATEWAY_BASE')
const inferenceEndpoint = parseHttpEndpoint(inferenceBase, 'INFERENCE_BASE')
const stackCacheDir = fileURLToPath(new URL('../../.cache/test-stack/', import.meta.url))
const generatedGatewayConfigPath = fileURLToPath(
  new URL(`../../.cache/test-stack/gateway.stack.${process.pid}.yaml`, import.meta.url)
)
const generatedGatewayDbPath = fileURLToPath(
  new URL(`../../.cache/test-stack/gateway.stack.${process.pid}.db`, import.meta.url)
)

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

mkdirSync(stackCacheDir, { recursive: true })
writeGatewayStackConfig({
  sourcePath: resolveConfigPath(process.env.LYCHEE_GATEWAY_CONFIG),
  destinationPath: generatedGatewayConfigPath,
  databasePath: generatedGatewayDbPath,
  frontendOrigin: frontendEndpoint.origin,
  gatewayOrigin: gatewayEndpoint.origin,
  upstreamOrigin: inferenceEndpoint.origin,
  gatewayHost: gatewayEndpoint.host,
  gatewayPort: gatewayEndpoint.port
})

const inferenceProcess = spawn(process.execPath, ['run', 'dev:inference-api', '--', '--target', target], {
  cwd: repoRoot,
  env: {
    ...process.env,
    LYCHEE_PY_TARGET: target
  },
  detached: process.platform !== 'win32',
  stdio: 'inherit'
})
detachIfNeeded(inferenceProcess)

const gatewayProcess = spawn(process.execPath, ['run', 'dev:gateway'], {
  cwd: repoRoot,
  env: {
    ...process.env,
    LYCHEE_GATEWAY_CONFIG: generatedGatewayConfigPath
  },
  detached: process.platform !== 'win32',
  stdio: 'inherit'
})
detachIfNeeded(gatewayProcess)

const frontendProcess = spawn(
  process.execPath,
  ['run', '--cwd', frontendDir, 'dev', '--', '--host', frontendEndpoint.host, '--port', String(frontendEndpoint.port)],
  {
    cwd: repoRoot,
    env: {
      ...process.env,
      NUXT_PUBLIC_GATEWAY_BASE: gatewayEndpoint.origin
    },
    detached: process.platform !== 'win32',
    stdio: 'inherit'
  }
)
detachIfNeeded(frontendProcess)

const managedProcesses = [inferenceProcess, gatewayProcess, frontendProcess]

let cleanedUp = false

const cleanup = () => {
  if (cleanedUp) {
    return
  }
  cleanedUp = true

  for (const child of managedProcesses) {
    stopProcessTree(child.pid)
  }

  rmSync(generatedGatewayConfigPath, { force: true })
  rmSync(generatedGatewayDbPath, { force: true })
  rmSync(`${generatedGatewayDbPath}-shm`, { force: true })
  rmSync(`${generatedGatewayDbPath}-wal`, { force: true })
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.once(signal, () => {
    cleanup()
    process.exit(130)
  })
}

try {
  await Promise.all([
    waitForService(`${frontendEndpoint.origin}/`, {
      expectedType: 'text/html',
      timeoutMs,
      managedProcesses
    }),
    waitForService(`${gatewayEndpoint.origin}/healthz`, {
      expectedType: 'application/json',
      timeoutMs,
      managedProcesses
    }),
    waitForService(`${gatewayEndpoint.origin}/v1/health`, {
      expectedType: 'application/json',
      timeoutMs,
      managedProcesses
    })
  ])
  await runStackSmoke({ frontendBase: frontendEndpoint.origin, gatewayBase: gatewayEndpoint.origin })
  console.log('Stack smoke passed with auto-started dev services.')
} finally {
  cleanup()
}

function parseArgs(args) {
  let target
  let timeoutMs

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index]
    if (arg === '--target') {
      const value = args[index + 1]
      if (!value) {
        console.error('Missing value for --target. Expected cpu|cu128.')
        process.exit(1)
      }
      target = value
      index += 1
      continue
    }
    if (arg === '--timeout-ms') {
      const value = Number(args[index + 1])
      if (!Number.isFinite(value) || value <= 0) {
        console.error('Invalid value for --timeout-ms. Expected positive integer milliseconds.')
        process.exit(1)
      }
      timeoutMs = value
      index += 1
      continue
    }
    console.error(`Unknown argument: ${arg}`)
    process.exit(1)
  }

  return { target, timeoutMs }
}

function parseHttpEndpoint(rawUrl, envName) {
  let parsed
  try {
    parsed = new URL(rawUrl)
  } catch {
    console.error(`Invalid ${envName}: ${rawUrl}`)
    process.exit(1)
  }

  if (!['http:', 'https:'].includes(parsed.protocol)) {
    console.error(`Invalid ${envName}: expected http or https URL, got ${parsed.protocol}`)
    process.exit(1)
  }

  return {
    origin: parsed.origin,
    host: parsed.hostname,
    port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80)
  }
}

function resolveConfigPath(rawPath) {
  const defaultConfigPath = fileURLToPath(new URL('../../tooling/configs/gateway.yaml', import.meta.url))
  const exampleConfigPath = fileURLToPath(new URL('../../tooling/configs/gateway.yaml.example', import.meta.url))

  if (!rawPath) {
    if (existsSync(defaultConfigPath)) {
      return defaultConfigPath
    }

    if (existsSync(exampleConfigPath)) {
      console.warn(
        'tooling/configs/gateway.yaml is missing; test:stack is temporarily using tooling/configs/gateway.yaml.example.'
      )
      return exampleConfigPath
    }

    console.error(
      'Missing tooling/configs/gateway.yaml and tooling/configs/gateway.yaml.example. Restore the example config before running test:stack.'
    )
    process.exit(1)
  }

  const resolvedPath = isAbsolute(rawPath) ? rawPath : resolve(repoRoot, rawPath)
  if (!existsSync(resolvedPath)) {
    console.error(`LYCHEE_GATEWAY_CONFIG does not exist: ${resolvedPath}`)
    process.exit(1)
  }

  return resolvedPath
}

function writeGatewayStackConfig({
  sourcePath,
  destinationPath,
  databasePath,
  frontendOrigin,
  gatewayOrigin,
  upstreamOrigin,
  gatewayHost,
  gatewayPort
}) {
  const document = parseDocument(readFileSync(sourcePath, 'utf8'))
  document.setIn(['server', 'host'], gatewayHost)
  document.setIn(['server', 'port'], Number(gatewayPort))
  document.setIn(['upstream', 'base_url'], upstreamOrigin)
  document.setIn(['db', 'driver'], 'sqlite')
  document.setIn(['db', 'dsn'], databasePath)
  document.setIn(['seed', 'default_resources_enabled'], false)
  document.setIn(['trace', 'mode'], 'database')
  document.setIn(['auth', 'mode'], 'disabled')
  document.setIn(['auth', 'web', 'public_base_url'], gatewayOrigin)
  document.setIn(['auth', 'web', 'app_base_url'], frontendOrigin)
  document.setIn(['cors', 'allowed_origins'], [frontendOrigin])
  writeFileSync(destinationPath, document.toString())
}

async function waitForService(url, { expectedType, timeoutMs, managedProcesses }) {
  const start = Date.now()
  let lastError = new Error(`Timed out waiting for ${url}`)
  let attempt = 0

  while (Date.now() - start < timeoutMs) {
    for (const child of managedProcesses) {
      if (child.exitCode !== null) {
        throw new Error(`A dev process exited early with code ${child.exitCode} while waiting for ${url}`)
      }
    }

    try {
      const response = await fetch(url)
      const contentType = response.headers.get('content-type') ?? ''
      if (response.ok && (!expectedType || contentType.includes(expectedType))) {
        return
      }
      lastError = new Error(`Waiting for ${url}: got ${response.status} ${contentType}`)
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error))
    }

    const elapsedMs = Date.now() - start
    const remainingMs = timeoutMs - elapsedMs
    if (remainingMs <= 0) {
      break
    }

    const nextDelayMs = Math.min(5_000, 250 * 2 ** Math.min(attempt, 5))
    attempt += 1
    await sleep(Math.min(nextDelayMs, remainingMs))
  }

  throw lastError
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function stopProcessTree(pid) {
  if (!pid) {
    return
  }

  if (process.platform === 'win32') {
    spawnSync('taskkill', ['/PID', String(pid), '/T', '/F'], {
      stdio: 'ignore',
      windowsHide: true
    })
    return
  }

  try {
    process.kill(-pid, 'SIGTERM')
  } catch {
    return
  }
}

function detachIfNeeded(child) {
  if (process.platform !== 'win32') {
    child.unref()
  }
}
