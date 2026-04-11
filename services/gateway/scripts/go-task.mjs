import { mkdirSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const [, , command, ...args] = process.argv

if (!command) {
  console.error('Missing Go command, expected build, run, or test.')
  process.exit(1)
}

const goCache = fileURLToPath(new URL('../../../.cache/go-build', import.meta.url))
const binDir = fileURLToPath(new URL('../../../.cache/bin', import.meta.url))
const gatewayBinary = fileURLToPath(
  new URL(`../../../.cache/bin/gateway${process.platform === 'win32' ? '.exe' : ''}`, import.meta.url)
)
mkdirSync(goCache, { recursive: true })
mkdirSync(binDir, { recursive: true })

const env = {
  ...process.env,
  GOCACHE: goCache
}

let result
if (command === 'run') {
  const [packagePath, ...binaryArgs] = args
  if (!packagePath) {
    console.error('Missing package path for go run. Expected a package such as ./cmd/gateway.')
    process.exit(1)
  }

  const buildResult = spawnSync('go', ['build', '-o', gatewayBinary, packagePath], {
    env,
    shell: process.platform === 'win32',
    stdio: 'inherit'
  })

  if ((buildResult.status ?? 1) !== 0) {
    process.exit(buildResult.status ?? 1)
  }

  result = spawnSync(gatewayBinary, binaryArgs, {
    env,
    stdio: 'inherit'
  })
} else if (command === 'build') {
  const [packagePath, ...buildArgs] = args
  if (!packagePath) {
    console.error('Missing package path for go build. Expected a package such as ./cmd/gateway.')
    process.exit(1)
  }

  const outputArgs = buildArgs.includes('-o') ? buildArgs : ['-o', gatewayBinary, ...buildArgs]
  result = spawnSync('go', ['build', ...outputArgs, packagePath], {
    env,
    shell: process.platform === 'win32',
    stdio: 'inherit'
  })
} else {
  result = spawnSync('go', [command, ...args], {
    env,
    shell: process.platform === 'win32',
    stdio: 'inherit'
  })
}

process.exit(result.status ?? 1)
