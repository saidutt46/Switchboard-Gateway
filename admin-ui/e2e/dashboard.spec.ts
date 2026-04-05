import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test('loads and shows health status', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('System Health')).toBeVisible()
    await expect(page.getByText('Gateway')).toBeVisible()
    await expect(page.getByText('Database')).toBeVisible()
    await expect(page.getByText('Redis')).toBeVisible()
  })

  test('shows overview section', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Overview' }).or(page.getByText('Overview').first())).toBeVisible()
  })

  test('navigates to services from sidebar', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: 'Services' }).click()
    await expect(page).toHaveURL('/services')
  })

  test('sidebar navigation works', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: 'Services' }).click()
    await expect(page).toHaveURL('/services')
    await page.getByRole('link', { name: 'Routes' }).click()
    await expect(page).toHaveURL('/routes')
    await page.getByRole('link', { name: 'Consumers' }).click()
    await expect(page).toHaveURL('/consumers')
    await page.getByRole('link', { name: 'Plugins' }).click()
    await expect(page).toHaveURL('/plugins')
    await page.getByRole('link', { name: 'Dashboard' }).click()
    await expect(page).toHaveURL('/')
  })
})
