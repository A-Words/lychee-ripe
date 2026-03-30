import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const [, , command, ...args] = process.argv

if (!command) {
  console.error('Missing Go command, expected build or test.')
  process.exit(1)
}

const goCache = fileURLToPath(new URL('../../../mlops/artifacts/.cache/go-build', import.meta.url))
const result = spawnSync('go', [command, ...args], {
  env: {
    ...process.env,
    GOCACHE: goCache
  },
  shell: process.platform === 'win32',
  stdio: 'inherit'
})

process.exit(result.status ?? 1)
