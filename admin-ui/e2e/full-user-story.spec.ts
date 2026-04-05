import { test, expect } from '@playwright/test'

const suffix = Date.now()
const serviceName = `e2e-full-svc-${suffix}`
const consumerName = `e2e-full-consumer-${suffix}`
const routeName = `e2e-route-${suffix}`

test.describe.serial('Full User Story', () => {
  test('1. Create consumer', async ({ page }) => {
    await page.goto('/consumers')
    await page.getByRole('button', { name: 'New Consumer' }).click()
    await expect(page.getByPlaceholder('mobile-app-ios')).toBeVisible({ timeout: 5000 })
    await page.getByPlaceholder('mobile-app-ios').fill(consumerName)
    await page.getByRole('button', { name: 'Create Consumer' }).click()
    await expect(page.locator('table').getByText(consumerName)).toBeVisible({ timeout: 5000 })
  })

  test('2. Create service', async ({ page }) => {
    await page.goto('/services')
    await page.getByRole('button', { name: 'New Service' }).click()
    await expect(page.getByPlaceholder('my-backend-service')).toBeVisible({ timeout: 5000 })
    await page.getByPlaceholder('my-backend-service').fill(serviceName)
    await page.getByPlaceholder('api.example.com').fill('demo-backend')
    await page.getByRole('button', { name: 'Create Service' }).click()
    await expect(page.locator('table').getByText(serviceName)).toBeVisible({ timeout: 5000 })
  })

  test('3. Create route', async ({ page }) => {
    await page.goto('/routes')
    await page.getByRole('button', { name: 'New Route' }).click()
    await expect(page.getByPlaceholder('my-api-routes')).toBeVisible({ timeout: 5000 })

    const serviceSelect = page.locator('select').first()
    await serviceSelect.selectOption({ label: serviceName })
    await page.getByPlaceholder('my-api-routes').fill(routeName)
    await page.locator('textarea').fill('/e2e-test\n/e2e-test/:id')

    await page.getByRole('button', { name: 'Create Route' }).click()
    await expect(page.locator('table').getByText(routeName)).toBeVisible({ timeout: 5000 })
  })

  test('4. Dashboard loads', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('System Health')).toBeVisible()
  })

  test('5. Clean up — delete route', async ({ page }) => {
    await page.goto('/routes')
    await page.locator('table').getByText(routeName).click()
    await page.locator('header').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByText(`Delete ${routeName}?`)).toBeVisible({ timeout: 5000 })
    await page.getByRole('button', { name: 'Delete' }).last().click()
    await expect(page).toHaveURL('/routes', { timeout: 5000 })
  })

  test('6. Clean up — delete service', async ({ page }) => {
    await page.goto('/services')
    await page.locator('table').getByText(serviceName).click()
    await page.locator('header').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByText(`Delete ${serviceName}?`)).toBeVisible({ timeout: 5000 })
    await page.getByRole('button', { name: 'Delete' }).last().click()
    await expect(page).toHaveURL('/services', { timeout: 5000 })
  })

  test('7. Clean up — delete consumer', async ({ page }) => {
    await page.goto('/consumers')
    await page.locator('table').getByText(consumerName).click()
    await page.locator('header').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByText(`Delete ${consumerName}?`)).toBeVisible({ timeout: 5000 })
    await page.getByRole('button', { name: 'Delete' }).last().click()
    await expect(page).toHaveURL('/consumers', { timeout: 5000 })
  })
})
