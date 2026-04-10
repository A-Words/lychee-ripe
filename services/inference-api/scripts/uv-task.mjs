import { mkdirSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const [, , task, ...rawArgs] = process.argv

if (!task) {
  console.error('Missing task. Expected one of: dev, test, verify, train, eval.')
  process.exit(1)
}

const parsed = parseArgs(rawArgs)
const target = parsed.target ?? process.env.LYCHEE_PY_TARGET ?? 'cpu'

if (!['cpu', 'cu128'].includes(target)) {
  console.error(`Invalid --target '${target}'. Expected cpu|cu128.`)
  process.exit(1)
}

const serviceDir = fileURLToPath(new URL('../', import.meta.url))
const repoRoot = fileURLToPath(new URL('../../../', import.meta.url))
const cacheRoot = fileURLToPath(new URL('../../../.cache/', import.meta.url))
const uvCacheDir = fileURLToPath(new URL('../../../.cache/uv', import.meta.url))
const xdgCacheHome = fileURLToPath(new URL('../../../.cache/xdg', import.meta.url))
const torchInductorCacheDir = fileURLToPath(new URL('../../../.cache/torchinductor', import.meta.url))

for (const path of [cacheRoot, uvCacheDir, xdgCacheHome, torchInductorCacheDir]) {
  mkdirSync(path, { recursive: true })
}

const commands = {
  dev: {
    cwd: serviceDir,
    args: [
      'run',
      '--extra',
      target,
      'python',
      '-m',
      'uvicorn',
      'app.main:app',
      '--reload',
      '--host',
      '127.0.0.1',
      '--port',
      '8000',
      ...parsed.passthrough
    ]
  },
  test: {
    cwd: serviceDir,
    args: ['run', '--extra', target, 'python', '-m', 'pytest', '-q', ...parsed.passthrough]
  },
  verify: {
    cwd: serviceDir,
    args: ['run', '--extra', target, 'python', '-m', 'pytest', '-q', ...parsed.passthrough]
  },
  train: {
    cwd: repoRoot,
    args: [
      'run',
      '--project',
      'services/inference-api',
      '--extra',
      target,
      'python',
      'mlops/training/train.py',
      '--data',
      'mlops/data/lichi/data.yaml',
      '--model',
      'mlops/pretrained/yolo26n.pt',
      '--project',
      'mlops/artifacts/models',
      ...parsed.passthrough
    ]
  },
  eval: {
    cwd: repoRoot,
    args: [
      'run',
      '--project',
      'services/inference-api',
      '--extra',
      target,
      'python',
      'mlops/training/eval.py',
      '--model',
      'mlops/artifacts/models/lychee_v1/weights/best.pt',
      '--data',
      'mlops/data/lichi/data.yaml',
      '--output',
      'mlops/artifacts/metrics/lychee_v1-eval_metrics.json',
      ...parsed.passthrough
    ]
  }
}

const selected = commands[task]
if (!selected) {
  console.error(`Unknown task '${task}'. Expected one of: ${Object.keys(commands).join(', ')}.`)
  process.exit(1)
}

const result = spawnSync('uv', selected.args, {
  cwd: selected.cwd,
  env: {
    ...process.env,
    UV_CACHE_DIR: uvCacheDir,
    XDG_CACHE_HOME: xdgCacheHome,
    TORCHINDUCTOR_CACHE_DIR: torchInductorCacheDir
  },
  shell: process.platform === 'win32',
  stdio: 'inherit'
})

process.exit(result.status ?? 1)

function parseArgs(args) {
  const passthrough = []
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
    passthrough.push(arg)
  }

  return { target, passthrough }
}
