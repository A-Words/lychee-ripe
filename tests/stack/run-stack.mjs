import { spawn, spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import { runStackSmoke } from './smoke-stack.mjs'

const repoRoot = fileURLToPath(new URL('../../', import.meta.url))
const parsed = parseArgs(process.argv.slice(2))
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'
const frontendBase = process.env.FRONTEND_BASE ?? 'http://localhost:3000'
const gatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'
const timeoutMs = parsed.timeoutMs ?? 120_000

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

const devArgs = ['run', 'dev', '--', '--target', target]
const devProcess = spawn(process.execPath, devArgs, {
  cwd: repoRoot,
  env: {
    ...process.env,
    LYCHEE_PY_TARGET: target
  },
  detached: process.platform !== 'win32',
  stdio: 'inherit'
})

let cleanedUp = false

const cleanup = async () => {
  if (cleanedUp) {
    return
  }
  cleanedUp = true
  await stopProcessTree(devProcess.pid)
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, async () => {
    await cleanup()
    process.exit(130)
  })
}

try {
  await waitForService(`${frontendBase}/`, { expectedType: 'text/html', timeoutMs, devProcess })
  await waitForService(`${gatewayBase}/v1/health`, { expectedType: 'application/json', timeoutMs, devProcess })
  await runStackSmoke({ frontendBase, gatewayBase })
  console.log('Stack smoke passed with auto-started dev services.')
} finally {
  await cleanup()
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

async function waitForService(url, { expectedType, timeoutMs, devProcess }) {
  const start = Date.now()
  let lastError = new Error(`Timed out waiting for ${url}`)

  while (Date.now() - start < timeoutMs) {
    if (devProcess.exitCode !== null) {
      throw new Error(`'bun run dev' exited early with code ${devProcess.exitCode}`)
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

    await sleep(1_000)
  }

  throw lastError
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function stopProcessTree(pid) {
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
