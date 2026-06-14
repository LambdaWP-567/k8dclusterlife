import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test('shows healthy banner when no problems', async ({ page }) => {
    await page.goto('/')
    // When no providers configured, skip login
    await page.waitForLoadState('networkidle')
    await expect(page.getByText('Alle Cluster gesund')).toBeVisible({ timeout: 10000 })
  })

  test('shows header with app name', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('k8dclusterlife')).toBeVisible()
  })

  test('shows health endpoint returns ok', async ({ request }) => {
    const resp = await request.get('/healthz')
    expect(resp.ok()).toBeTruthy()
    expect(await resp.text()).toBe('ok')
  })

  test('problems API returns array', async ({ request }) => {
    const resp = await request.get('/api/problems')
    // 200 (no auth configured) or 401 (auth configured)
    expect([200, 401]).toContain(resp.status())
    if (resp.status() === 200) {
      const body = await resp.json()
      expect(Array.isArray(body)).toBeTruthy()
    }
  })

  test('providers API returns array', async ({ request }) => {
    const resp = await request.get('/api/auth/providers')
    expect(resp.ok()).toBeTruthy()
    const body = await resp.json()
    expect(Array.isArray(body)).toBeTruthy()
  })
})
