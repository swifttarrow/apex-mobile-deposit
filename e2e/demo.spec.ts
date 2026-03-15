import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

test.describe('Demo — mobile to operator e2e', () => {
  test('health check', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/health`);
    expect(res.ok()).toBe(true);
    const body = await res.json();
    expect(body).toMatchObject({ status: 'ok', service: 'checkdepot' });
  });

  test('mobile deposit, operator review, settlement', async ({ page }) => {
    test.setTimeout(120_000);

    // ── 1. Operator login (test user joe) ──
    await page.goto('/login');
    await expect(page.getByRole('heading', { name: /operator login/i })).toBeVisible();
    await page.getByRole('textbox', { name: /username/i }).fill('joe');
    await page.getByLabel(/password/i).fill('password');
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page.getByRole('button', { name: 'Review Queue' })).toBeVisible();

    // ── 2. Mobile: submit deposits ──
    await page.goto('/mobile/');
    await expect(page.getByText('Deposit Check')).toBeVisible();

    await page.locator('#frontBox').click();
    await page.locator('#backBox').click();
    await page.locator('#amountInput').fill('150');
    await expect(page.locator('.scenario-chip[data-scenario="clean_pass"]')).toHaveClass(/selected/);
    await page.getByRole('button', { name: 'Submit Deposit' }).click();

    await expect(page.locator('#statusBannerTitle')).toHaveText('Deposit Submitted', { timeout: 10_000 });
    await expect(page.locator('#statusAmount')).toHaveText('$150.00');

    await page.getByRole('button', { name: 'DEPOSIT' }).click();
    await page.locator('#frontBox').click();
    await page.locator('#backBox').click();
    await page.locator('#amountInput').fill('200');
    await page.locator('.scenario-chip[data-scenario="micr_fail"]').click();
    await page.locator('#accountSelect').selectOption('ACC-MICR-FAIL');
    await page.getByRole('button', { name: 'Submit Deposit' }).click();

    await expect(page.locator('#statusBannerTitle')).toContainText(/Deposit Submitted|validated/, { timeout: 10_000 });

    // ── 3. Operator: approve flagged deposit ──
    await page.goto('/review-queue');
    await expect(page.locator('#viewQueue .queue-title')).toHaveText('Review Queue');
    await expect(page.locator('#queueTbody tr[data-id]').first()).toBeVisible({ timeout: 10_000 });
    await page.locator('#queueTbody tr[data-id]').first().click();
    await expect(page.locator('#viewDetail')).toBeVisible();
    await page.locator('#detailApproveBtn').click();
    await expect(page.locator('#actionFeedback')).toContainText(/Approved.*state:|Funds Posted|Completed/, { timeout: 10_000 });

    // ── 4. Operator: settlement ──
    await page.getByRole('button', { name: 'Settlement' }).click();
    await expect(page.locator('#viewSettlement .queue-title')).toHaveText('Settlement');
    await page.locator('#triggerSettlementBtn').click();
    await expect(page.locator('#settlementResult')).toContainText(/Batch|triggered|total_count|\d+ transfer/i, { timeout: 10_000 });

    // ── 5. Operator: one more mobile deposit ──
    await page.goto('/mobile/');
    await page.getByRole('button', { name: 'DEPOSIT' }).click();
    await page.locator('#frontBox').click();
    await page.locator('#backBox').click();
    await page.locator('#amountInput').fill('75');
    await page.locator('.scenario-chip[data-scenario="clean_pass"]').click();
    await page.getByRole('button', { name: 'Submit Deposit' }).click();
    await expect(page.locator('#statusBannerTitle')).toHaveText('Deposit Submitted', { timeout: 10_000 });

    // ── 6. Operator: Settlement and Deposits list ──
    await page.goto('/settlement');
    await expect(page.locator('#viewSettlement .queue-title')).toHaveText('Settlement');
    await page.locator('#triggerSettlementBtn').click();
    await expect(page.locator('#settlementResult')).toContainText(/Batch|triggered|total_count|\d+ transfer/i, { timeout: 10_000 });

    await page.getByRole('button', { name: 'Deposits' }).click();
    await expect(page.locator('#viewDeposits .queue-title')).toHaveText('Deposits');
  });
});
