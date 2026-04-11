import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'
import { parse } from 'yaml'

const workspaceRoot = fileURLToPath(new URL('..', import.meta.url))
const ripenessPath = resolve(workspaceRoot, 'constants', 'ripeness.json')
const openapiPath = resolve(workspaceRoot, 'schemas', 'openapi.yaml')

const ripeness = JSON.parse(readFileSync(ripenessPath, 'utf8'))
const openapi = parse(readFileSync(openapiPath, 'utf8'))

assert(Array.isArray(ripeness.classes), 'ripeness.classes must be an array')
assert(
  JSON.stringify(ripeness.classes) === JSON.stringify(['green', 'half', 'red', 'young']),
  'ripeness.classes must remain ordered as green/half/red/young'
)
assert(ripeness.color_map && typeof ripeness.color_map === 'object', 'ripeness.color_map must be an object')

for (const key of ripeness.classes) {
  const color = ripeness.color_map[key]
  assert(typeof color === 'string' && color.length > 0, `missing color for ripeness class '${key}'`)
}

assert(openapi && typeof openapi === 'object', 'openapi.yaml must parse to an object')
assert(openapi.openapi === '3.1.0', 'openapi version must remain 3.1.0')
assert(openapi.info && typeof openapi.info === 'object', 'openapi info section is required')
assert(openapi.paths && typeof openapi.paths === 'object', 'openapi paths section is required')

console.log('shared/contracts verify passed')

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}
