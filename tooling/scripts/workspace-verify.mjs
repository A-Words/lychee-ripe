import { spawnSync } from 'node:child_process'

const parsed = parseArgs(process.argv.slice(2))
const { commands, passthrough } = parsed

if (commands.length === 0) {
  console.error('Missing verify commands.')
  process.exit(1)
}

// When Turbo orchestrates verify, the task graph already runs the dependent checks.
// Keep the package-level script meaningful for direct `bun run --filter <pkg> verify`
// without duplicating work during root `turbo run verify`.
if (process.env.TURBO_HASH || process.env.TURBO_TASK) {
  process.exit(0)
}

for (const [index, command] of commands.entries()) {
  const argv = parseCommand(command)
  if (index === commands.length - 1 && passthrough.length > 0) {
    argv.push(...passthrough)
  }

  const [file, ...args] = argv
  const result = spawnSync(file, args, {
    cwd: process.cwd(),
    env: process.env,
    shell: false,
    stdio: 'inherit'
  })

  if ((result.status ?? 1) !== 0) {
    process.exit(result.status ?? 1)
  }
}

function parseArgs(args) {
  const commands = []
  const passthrough = []

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index]
    if (arg === '--command') {
      const command = args[index + 1]
      if (!command) {
        console.error('Missing value for --command.')
        process.exit(1)
      }
      commands.push(command)
      index += 1
      continue
    }
    passthrough.push(arg)
  }

  return { commands, passthrough }
}

function parseCommand(command) {
  if (/[`"'|&;<>()]/.test(command)) {
    console.error(`Unsupported verify command: ${command}`)
    process.exit(1)
  }

  const argv = command.trim().split(/\s+/).filter(Boolean)
  if (argv.length === 0) {
    console.error('Empty verify command.')
    process.exit(1)
  }

  return argv
}
