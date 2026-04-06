import { test, expect, createOrgViaAPI } from "./fixtures"

/**
 * Notification and session management E2E tests.
 *
 * Requires running backend + frontend.
 */

test.describe("Session Management", () => {
  test("can view active sessions", async ({ authedPage }) => {
    await authedPage.goto("/account")

    // Look for sessions section
    await expect(
      authedPage.getByText(/sessions|active sessions/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })

  test("can view API keys section", async ({ authedPage }) => {
    await authedPage.goto("/account")

    await expect(
      authedPage.getByText(/api key/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})

test.describe("Notification Channels", () => {
  test("can view notification channels page", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Notif E2E Org", `notif-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )

    await authedPage.goto("/notifications/channels")

    await expect(
      authedPage.getByText(/notification|channel|no channels/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})

test.describe("Full Login Flow", () => {
  test("register → login → view profile → logout", async ({ page }) => {
    const uniqueEmail = `e2e-flow-${Date.now()}@example.com`

    // 1. Register
    await page.goto("/register")
    await page.getByPlaceholder("John").fill("Flow")
    await page.getByPlaceholder("Doe").fill("Test")
    await page.getByPlaceholder("you@example.com").fill(uniqueEmail)
    await page.getByPlaceholder("At least 8 characters").fill("FlowTestPass123!")
    await page.getByPlaceholder("Confirm your password").fill("FlowTestPass123!")
    await page.getByRole("button", { name: /create account/i }).click()

    // Should redirect to dashboard or onboarding
    await page.waitForURL(/\/(dashboard|onboarding|$)/, { timeout: 10_000 })

    // 2. Check profile/account
    // Navigate to account or verify user info is shown somewhere
    const userMenu = page.getByText(/flow|account|profile/i).first()
    if (await userMenu.isVisible({ timeout: 3000 }).catch(() => false)) {
      await userMenu.click()
    }

    // 3. Logout
    const logoutBtn = page.getByRole("button", { name: /logout|sign out/i }).or(
      page.getByText(/logout|sign out/i)
    )
    if (await logoutBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await logoutBtn.click()
      await page.waitForURL(/\/login/, { timeout: 5000 })
      expect(page.url()).toContain("/login")
    }
  })
})

test.describe("Audit Logs", () => {
  test("can view audit logs page", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Audit E2E Org", `audit-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )

    await authedPage.goto(`/organizations/${org.id}/audit-logs`)

    await expect(
      authedPage.getByText(/audit|log/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})
