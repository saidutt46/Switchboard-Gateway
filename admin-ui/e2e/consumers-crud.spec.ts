import { test, expect } from '@playwright/test'

const uniqueUsername = `e2e-consumer-${Date.now()}`

test.describe.serial('Consumers CRUD', () => {
  test('create a new consumer', async ({ page }) => {
    await page.goto('/consumers')
    await page.getByRole('button', { name: 'New Consumer' }).click()

    await expect(page.getByPlaceholder('mobile-app-ios')).toBeVisible({ timeout: 5000 })

    await page.getByPlaceholder('mobile-app-ios').fill(uniqueUsername)
    await page.getByPlaceholder('team@example.com').fill('e2e@test.com')

    await page.getByRole('button', { name: 'Create Consumer' }).click()
    await expect(page.locator('table').getByText(uniqueUsername)).toBeVisible({ timeout: 5000 })
  })

  test('view consumer detail and generate API key', async ({ page }) => {
    await page.goto('/consumers')
    await page.locator('table').getByText(uniqueUsername).click()

    await expect(page.getByRole('heading', { name: 'API Keys' })).toBeVisible()

    await page.getByRole('button', { name: 'Generate Key' }).click()
    await expect(page.getByPlaceholder('production-key')).toBeVisible({ timeout: 5000 })

    await page.getByPlaceholder('production-key').fill('e2e-test-key')
    await page.getByRole('button', { name: 'Generate', exact: true }).click()

    await expect(page.getByText('Save this key now')).toBeVisible({ timeout: 5000 })
    await page.getByRole('button', { name: "Done — I've saved the key" }).click()
  })

  test('delete consumer', async ({ page }) => {
    await page.goto('/consumers')
    await page.locator('table').getByText(uniqueUsername).click()

    await page.locator('header').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByText(`Delete ${uniqueUsername}?`)).toBeVisible({ timeout: 5000 })
    await page.getByRole('button', { name: 'Delete' }).last().click()

    await expect(page).toHaveURL('/consumers', { timeout: 5000 })
  })
})
