import { spawnSync } from 'node:child_process'

const [, , task, ...rawArgs] = process.argv

if (!task) {
  console.error('Missing turbo task name.')
  process.exit(1)
}

const parsed = parseArgs(rawArgs)
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

const result = spawnSync(
  'turbo',
  ['run', task, ...parsed.turboArgs],
  {
    env: {
      ...process.env,
      LYCHEE_PY_TARGET: target
    },
    shell: process.platform === 'win32',
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
