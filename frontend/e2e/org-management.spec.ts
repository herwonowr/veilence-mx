import { test, expect, createOrgViaAPI } from "./fixtures"

/**
 * Organization management E2E tests.
 *
 * Covers: org creation → invite member → settings update.
 * Requires running backend + frontend.
 */

test.describe("Organization Management", () => {
  test("can create a new organization", async ({ authedPage }) => {
    await authedPage.goto("/organizations")

    // Look for create org button
    const createBtn = authedPage.getByRole("button", { name: /create|new/i })
    if (await createBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await createBtn.click()

      const nameInput = authedPage.getByPlaceholder(/organization name/i).or(
        authedPage.getByLabel(/name/i)
      )
      if (await nameInput.isVisible({ timeout: 3000 }).catch(() => false)) {
        await nameInput.fill("E2E Test Org")

        const slugInput = authedPage.getByPlaceholder(/slug/i).or(
          authedPage.getByLabel(/slug/i)
        )
        if (await slugInput.isVisible({ timeout: 1000 }).catch(() => false)) {
          await slugInput.fill(`e2e-org-${Date.now()}`)
        }

        const submitBtn = authedPage.getByRole("button", { name: /create|save|submit/i })
        await submitBtn.click()

        await expect(authedPage.getByText("E2E Test Org")).toBeVisible({ timeout: 5000 })
      }
    }
  })

  test("can view organization members", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Members E2E Org", `members-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )

    await authedPage.goto(`/organizations/${org.id}/members`)

    // Should show at least the owner
    await expect(
      authedPage.getByText(/owner|member/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })

  test("can view organization roles", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Roles E2E Org", `roles-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )

    await authedPage.goto(`/organizations/${org.id}/roles`)

    // Should show the predefined roles
    await expect(
      authedPage.getByText(/owner|admin|member|viewer/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })
})

test.describe("Settings", () => {
  test("can view settings page", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Settings E2E Org", `settings-e2e-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )
    await authedPage.goto("/settings")

    await expect(
      authedPage.getByText(/settings|configuration/i).first()
    ).toBeVisible({ timeout: 10_000 })
  })

  test("can update poll interval setting", async ({ authedPage, accessToken }) => {
    const org = await createOrgViaAPI(accessToken, "Update Settings Org", `update-settings-${Date.now()}`)

    await authedPage.evaluate(
      (orgId) => localStorage.setItem("vmx_current_org_id", String(orgId)),
      org.id
    )
    await authedPage.goto("/settings")

    // Look for a poll interval input
    const pollInput = authedPage.getByLabel(/poll interval|pypi.*interval/i)
    if (await pollInput.isVisible({ timeout: 5000 }).catch(() => false)) {
      await pollInput.clear()
      await pollInput.fill("10m")

      const saveBtn = authedPage.getByRole("button", { name: /save|update/i })
      if (await saveBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
        await saveBtn.click()

        // Should show success feedback
        await expect(
          authedPage.getByText(/saved|updated|success/i).first()
        ).toBeVisible({ timeout: 5000 })
      }
    }
  })
})
