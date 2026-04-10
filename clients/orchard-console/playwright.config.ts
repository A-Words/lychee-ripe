import { defineConfig, devices } from '@playwright/test'

const isCI = Boolean(process.env.CI)

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: {
    timeout: 10_000
  },
  fullyParallel: false,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:3000',
    trace: 'on-first-retry',
    ...devices['Desktop Chrome']
  },
  webServer: [
    {
      command: 'go run ./services/gateway/cmd/gateway --config tooling/configs/gateway.yaml',
      cwd: '../..',
      url: 'http://127.0.0.1:9000/v1/dashboard/overview',
      reuseExistingServer: !isCI,
      timeout: 30_000
    },
    {
      command: 'bun run dev -- --host 127.0.0.1 --port 3000',
      url: 'http://127.0.0.1:3000/dashboard',
      reuseExistingServer: !isCI,
      timeout: 60_000
    }
  ]
})
