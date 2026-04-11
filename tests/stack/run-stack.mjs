import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { spawn, spawnSync } from 'node:child_process'
import { isAbsolute, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { runStackSmoke } from './smoke-stack.mjs'

const repoRoot = fileURLToPath(new URL('../../', import.meta.url))
const frontendDir = fileURLToPath(new URL('../../clients/orchard-console/', import.meta.url))
const parsed = parseArgs(process.argv.slice(2))
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'
const frontendBase = process.env.FRONTEND_BASE ?? 'http://localhost:3000'
const gatewayBase = process.env.GATEWAY_BASE ?? 'http://127.0.0.1:9000'
const timeoutMs = parsed.timeoutMs ?? 120_000
const frontendEndpoint = parseHttpEndpoint(frontendBase, 'FRONTEND_BASE')
const gatewayEndpoint = parseHttpEndpoint(gatewayBase, 'GATEWAY_BASE')
const stackCacheDir = fileURLToPath(new URL('../../.cache/test-stack/', import.meta.url))
const generatedGatewayConfigPath = fileURLToPath(new URL('../../.cache/test-stack/gateway.stack.yaml', import.meta.url))

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

mkdirSync(stackCacheDir, { recursive: true })
writeGatewayStackConfig({
  sourcePath: resolveConfigPath(process.env.LYCHEE_GATEWAY_CONFIG),
  destinationPath: generatedGatewayConfigPath,
  frontendOrigin: frontendEndpoint.origin,
  gatewayOrigin: gatewayEndpoint.origin,
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

const gatewayProcess = spawn(process.execPath, ['run', 'dev:gateway'], {
  cwd: repoRoot,
  env: {
    ...process.env,
    LYCHEE_GATEWAY_CONFIG: generatedGatewayConfigPath
  },
  detached: process.platform !== 'win32',
  stdio: 'inherit'
})

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

const managedProcesses = [inferenceProcess, gatewayProcess, frontendProcess]

let cleanedUp = false

const cleanup = async () => {
  if (cleanedUp) {
    return
  }
  cleanedUp = true

  for (const child of managedProcesses) {
    await stopProcessTree(child.pid)
  }

  rmSync(generatedGatewayConfigPath, { force: true })
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, async () => {
    await cleanup()
    process.exit(130)
  })
}

try {
  await waitForService(`${frontendEndpoint.origin}/`, {
    expectedType: 'text/html',
    timeoutMs,
    managedProcesses
  })
  await waitForService(`${gatewayEndpoint.origin}/v1/health`, {
    expectedType: 'application/json',
    timeoutMs,
    managedProcesses
  })
  await runStackSmoke({ frontendBase: frontendEndpoint.origin, gatewayBase: gatewayEndpoint.origin })
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
  if (!rawPath) {
    return fileURLToPath(new URL('../../tooling/configs/gateway.yaml', import.meta.url))
  }

  try {
    return isAbsolute(rawPath) ? rawPath : resolve(repoRoot, rawPath)
  } catch {
    console.error(`Invalid LYCHEE_GATEWAY_CONFIG path: ${rawPath}`)
    process.exit(1)
  }
}

function writeGatewayStackConfig({
  sourcePath,
  destinationPath,
  frontendOrigin,
  gatewayOrigin,
  gatewayHost,
  gatewayPort
}) {
  const content = readFileSync(sourcePath, 'utf8')
  const lines = content.split(/\r?\n/)
  const output = []
  const pathStack = []

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    const match = line.match(/^(\s*)([A-Za-z0-9_]+):(.*)$/)
    if (!match) {
      output.push(line)
      continue
    }

    const [, indentText, key, rawSuffix] = match
    const indent = indentText.length

    while (pathStack.length > 0 && pathStack[pathStack.length - 1].indent >= indent) {
      pathStack.pop()
    }

    const parentPath = pathStack.map((entry) => entry.key)
    const currentPath = [...parentPath, key].join('.')
    const hasNestedBlock = rawSuffix.trim() === ''

    if (currentPath === 'server.host') {
      output.push(`${indentText}${key}: "${gatewayHost}"`)
    } else if (currentPath === 'server.port') {
      output.push(`${indentText}${key}: ${gatewayPort}`)
    } else if (currentPath === 'auth.web.public_base_url') {
      output.push(`${indentText}${key}: "${gatewayOrigin}"`)
    } else if (currentPath === 'auth.web.app_base_url') {
      output.push(`${indentText}${key}: "${frontendOrigin}"`)
    } else if (currentPath === 'cors.allowed_origins') {
      output.push(`${indentText}${key}:`)
      output.push(`${' '.repeat(indent + 2)}- "${frontendOrigin}"`)
      while (index + 1 < lines.length) {
        const nextLine = lines[index + 1]
        const nextIndent = nextLine.match(/^(\s*)/)?.[1].length ?? 0
        if (nextLine.trim() !== '' && nextIndent <= indent) {
          break
        }
        index += 1
      }
    } else {
      output.push(line)
    }

    if (hasNestedBlock) {
      pathStack.push({ key, indent })
    }
  }

  writeFileSync(destinationPath, `${output.join('\n')}\n`)
}

async function waitForService(url, { expectedType, timeoutMs, managedProcesses }) {
  const start = Date.now()
  let lastError = new Error(`Timed out waiting for ${url}`)

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
