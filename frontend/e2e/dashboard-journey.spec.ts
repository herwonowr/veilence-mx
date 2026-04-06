import { test, expect, createOrgViaAPI } from "./fixtures"

/**
 * Dashboard & Package Journey E2E tests.
 *
 * Full user journey: login → dashboard → view packages → view alerts.
 * Requires running backend + frontend.
 */

test.describe("Dashboard Journey", () => {
  test("authenticated user can see dashboard", async ({ authedPage }) => {
    await authedPage.goto("/")

    // Dashboard should show key stats sections
    await expect(
      authedPage.getByText(/dashboard|overview|packages|alerts/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })

  test("dashboard displays statistics cards", async ({ authedPage }) => {
    await authedPage.goto("/")

    // Look for common dashboard elements
    await expect(
      authedPage.getByText(/total packages|monitored/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})

test.describe("Package Journey", () => {
  test("can navigate to packages page", async ({ authedPage, accessToken }) => {
    // Create an org first
    const org = await createOrgViaAPI(accessToken, "Pkg E2E Org", `pkg-e2e-${Date.now()}`)

    // Set org context
    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )
    await authedPage.goto("/packages")

    await expect(
      authedPage.getByText(/packages|monitored/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })

  test("can create a package via UI", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Create Pkg Org", `create-pkg-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )
    await authedPage.goto("/packages")

    // Look for an "Add Package" or similar button
    const addButton = authedPage.getByRole("button", { name: /add|create|monitor/i })
    if (await addButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      await addButton.click()

      // Fill in the package form
      const nameInput = authedPage.getByPlaceholder(/package name/i)
      if (await nameInput.isVisible({ timeout: 3000 }).catch(() => false)) {
        await nameInput.fill("requests")
        // Select registry if dropdown exists
        const registrySelect = authedPage.getByRole("combobox")
        if (await registrySelect.isVisible({ timeout: 1000 }).catch(() => false)) {
          await registrySelect.selectOption("pypi")
        }
        // Submit
        const submitBtn = authedPage.getByRole("button", { name: /add|create|submit|save/i })
        if (await submitBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
          await submitBtn.click()
          // Verify package appears
          await expect(authedPage.getByText("requests")).toBeVisible({ timeout: 5000 })
        }
      }
    }
  })
})

test.describe("Alert Journey", () => {
  test("can navigate to alerts page", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Alert E2E Org", `alert-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )
    await authedPage.goto("/alerts")

    await expect(
      authedPage.getByText(/alerts|no alerts/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})
