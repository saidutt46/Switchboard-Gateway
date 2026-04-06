import { test, expect } from '@playwright/test'

const uniqueName = `e2e-svc-${Date.now()}`

test.describe.serial('Services CRUD', () => {
  test('create a new service', async ({ page }) => {
    await page.goto('/services')
    await page.getByRole('button', { name: 'New Service' }).click()

    // Wait for form to appear in the slide panel
    await expect(page.getByPlaceholder('my-backend-service')).toBeVisible({ timeout: 5000 })

    await page.getByPlaceholder('my-backend-service').fill(uniqueName)
    await page.getByPlaceholder('api.example.com').fill('localhost')

    await page.getByRole('button', { name: 'Create Service' }).click()
    await expect(page.locator('table').getByText(uniqueName)).toBeVisible({ timeout: 5000 })
  })

  test('view service detail', async ({ page }) => {
    await page.goto('/services')
    await page.locator('table').getByText(uniqueName).click()
    await expect(page.getByText('Configuration')).toBeVisible()
    await expect(page.getByText('localhost', { exact: true })).toBeVisible()
  })

  test('edit a service', async ({ page }) => {
    await page.goto('/services')
    await page.locator('table').getByText(uniqueName).click()

    await page.getByRole('button', { name: 'Edit' }).click()
    await expect(page.getByRole('button', { name: 'Update Service' })).toBeVisible({ timeout: 5000 })

    await page.getByRole('button', { name: 'Update Service' }).click()
    await expect(page.getByText('Service updated')).toBeVisible({ timeout: 5000 })
  })

  test('delete a service', async ({ page }) => {
    await page.goto('/services')
    await page.locator('table').getByText(uniqueName).click()

    await page.locator('header').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByText(`Delete ${uniqueName}?`)).toBeVisible({ timeout: 5000 })

    // Click the delete button inside the confirm dialog (not the header one)
    await page.getByRole('button', { name: 'Delete' }).last().click()
    await expect(page).toHaveURL('/services', { timeout: 5000 })
  })
})
