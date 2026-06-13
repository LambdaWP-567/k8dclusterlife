import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test('shows healthy banner when no problems', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('Alle Cluster gesund')).toBeVisible()
  })
})
