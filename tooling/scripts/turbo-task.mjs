import { existsSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const [, , task, ...rawArgs] = process.argv

if (!task) {
  console.error('Missing turbo task name.')
  process.exit(1)
}

const parsed = parseArgs(rawArgs)
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'
const repoRoot = fileURLToPath(new URL('../../', import.meta.url))
const turboBin = fileURLToPath(new URL('../../node_modules/turbo/bin/turbo', import.meta.url))

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

if (task === 'verify') {
  verifyConfigExamples()
}

const result = spawnSync(
  process.execPath,
  [turboBin, 'run', task, ...parsed.turboArgs],
  {
    cwd: repoRoot,
    env: {
      ...process.env,
      LYCHEE_PY_TARGET: target
    },
    stdio: 'inherit'
  }
)

process.exit(result.status ?? 1)

function parseArgs(args) {
  const turboArgs = []
  let target

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
    turboArgs.push(arg)
  }

  return { target, turboArgs }
}

function verifyConfigExamples() {
  const requiredFiles = [
    'tooling/configs/model.yaml.example',
    'tooling/configs/service.yaml.example',
    'tooling/configs/gateway.yaml.example'
  ]

  const missing = requiredFiles.filter((relativePath) =>
    !existsSync(fileURLToPath(new URL(`../../${relativePath}`, import.meta.url)))
  )

  if (missing.length > 0) {
    for (const relativePath of missing) {
      console.error(`Missing required config example: ${relativePath}`)
    }
    process.exit(1)
  }
}
