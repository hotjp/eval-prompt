import { test, expect } from '@playwright/test'

test.describe('Console Error Detection', () => {
  test('page loads without console errors', async ({ page }) => {
    const errors: string[] = []

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text())
      }
    })

    page.on('pageerror', (error) => {
      errors.push(error.message)
    })

    await page.goto('/')

    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle')

    // Check no console errors
    expect(errors).toEqual([])
  })

  test('asset list page loads', async ({ page }) => {
    await page.goto('/assets')
    await page.waitForLoadState('networkidle')

    // Should see asset list
    const content = await page.textContent('body')
    expect(content).toBeTruthy()
  })

  test('can navigate between pages', async ({ page }) => {
    await page.goto('/assets')
    await page.waitForLoadState('networkidle')

    // Try to navigate (routes may or may not exist)
    // This tests that navigation doesn't crash
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))

    await page.goto('/assets/any-id/edit')
    await page.waitForLoadState('networkidle')

    expect(errors).toEqual([])
  })
})
