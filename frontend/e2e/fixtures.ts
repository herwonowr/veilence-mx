import { test as base, expect, type Page } from "@playwright/test"

/**
 * Veilence-MX E2E test fixtures.
 *
 * Provides an authenticated page fixture that logs in via the API,
 * and helper functions for common operations.
 */

const TEST_USER = {
  email: `e2e-${Date.now()}@example.com`,
  password: "E2eTestPass123!",
  firstName: "E2E",
  lastName: "Tester",
}

const API_BASE = "http://localhost:8080"

type Fixtures = {
  /** A page pre-authenticated by registering + logging in via the API. */
  authedPage: Page
  /** The access token for API requests. */
  accessToken: string
  /** The test user's info. */
  testUser: typeof TEST_USER
}

export const test = base.extend<Fixtures>({
  testUser: async ({}, use) => {
    await use(TEST_USER)
  },

  accessToken: async ({}, use) => {
    // Register a new user via the API
    const registerRes = await fetch(`${API_BASE}/api/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(TEST_USER),
    })

    let token: string
    if (registerRes.ok) {
      const data = await registerRes.json()
      token = data.data.accessToken
    } else {
      // User might already exist, try login
      const loginRes = await fetch(`${API_BASE}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: TEST_USER.email,
          password: TEST_USER.password,
        }),
      })
      const data = await loginRes.json()
      token = data.data.accessToken
    }

    await use(token)
  },

  authedPage: async ({ page, accessToken }, use) => {
    // Set auth tokens in localStorage before navigating
    await page.goto("/login")
    await page.evaluate(
      ({ token }) => {
        localStorage.setItem("vmx_access_token", token)
      },
      { token: accessToken }
    )
    await use(page)
  },
})

export { expect }

/**
 * Helper: create an organization via API.
 */
export async function createOrgViaAPI(
  accessToken: string,
  name: string,
  slug: string
): Promise<{ id: number; name: string; slug: string }> {
  const res = await fetch(`${API_BASE}/api/orgs`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify({ name, slug, description: "E2E test org" }),
  })
  const data = await res.json()
  return data.data
}
