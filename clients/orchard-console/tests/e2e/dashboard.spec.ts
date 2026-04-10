import { expect, test } from '@playwright/test'

test('dashboard page renders chart canvases without known console warnings', async ({ page }) => {
  const consoleIssues: string[] = []

  page.on('console', (message) => {
    if (message.type() === 'warning' || message.type() === 'error') {
      consoleIssues.push(`[${message.type()}] ${message.text()}`)
    }
  })

  await page.goto('/dashboard')
  await page.waitForLoadState('networkidle')

  await expect(page.getByText('批次总数')).toBeVisible()

  const charts = page.locator('x-vue-echarts')
  await expect(charts).toHaveCount(2)

  const chartSizes = await charts.evaluateAll((nodes) =>
    nodes.map((node) => ({
      width: (node as HTMLElement).clientWidth,
      height: (node as HTMLElement).clientHeight
    }))
  )

  expect(chartSizes.every((size) => size.width > 0 && size.height > 0)).toBe(true)
  expect(consoleIssues).not.toEqual(
    expect.arrayContaining([expect.stringContaining("Can't get DOM width or height")])
  )
  expect(consoleIssues).not.toEqual(
    expect.arrayContaining([expect.stringContaining('`useRoute` was called within middleware')])
  )
})
