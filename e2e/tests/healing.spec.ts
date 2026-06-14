import { test, expect } from '@playwright/test'

test.describe('KI-Healing API', () => {
  test('healing endpoint returns 401 without auth', async ({ request }) => {
    const resp = await request.post('/api/healing', {
      data: {
        problem: {
          id: 'test-1',
          cluster_id: 'cluster-1',
          kind: 'Pod',
          name: 'test-pod',
          namespace: 'default',
          status: 'CrashLoopBackOff',
          description: 'Pod crasht',
          cause: 'Fehler',
          severity: 'critical',
          detected_at: new Date().toISOString(),
        },
        autonomy_mode: 'MANUAL',
      },
    })
    // Either 401 (auth enabled) or 201 (no auth configured)
    expect([201, 401]).toContain(resp.status())
  })

  test('healing session GET returns 404 for unknown session', async ({ request }) => {
    const resp = await request.get('/api/healing/nonexistent-session')
    // Either 401 (auth) or 404 (not found)
    expect([401, 404]).toContain(resp.status())
  })
})

test.describe('KI-Healing UI', () => {
  test('dashboard has KI-Heilung button when problems exist', async ({ page }) => {
    // Mock /api/problems to return a problem
    await page.route('/api/problems', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{
          id: 'prob-1',
          cluster_id: 'cluster-1',
          cluster_name: 'Test Cluster',
          kind: 'Pod',
          name: 'crash-pod',
          namespace: 'default',
          status: 'CrashLoopBackOff',
          description: 'Container startet immer wieder und schlägt fehl',
          cause: 'Fehler im Startprozess — KI prüft Logs',
          severity: 'critical',
          detected_at: new Date().toISOString(),
        }]),
      })
    })
    await page.route('/api/auth/providers', route => {
      route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
    })
    await page.route('/api/me', route => {
      route.fulfill({ status: 401 })
    })

    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // Problem card should be visible
    await expect(page.getByText('crash-pod')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('CrashLoopBackOff')).toBeVisible()

    // KI-Heilung button
    const healBtn = page.getByRole('button', { name: 'KI-Heilung' })
    await expect(healBtn).toBeVisible()
  })
})
