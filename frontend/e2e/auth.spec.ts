import { test, expect } from "@playwright/test"

/**
 * Auth flow E2E tests for Veilence-MX.
 *
 * These tests validate page rendering and client-side interactions.
 * Tests requiring a running backend are guarded with comments.
 *
 * Prerequisites:
 *   - Frontend running at http://localhost:3000
 *   - Backend running at http://localhost:8080 (for full flow tests)
 */

test.describe("Authentication Pages", () => {
  test("login page renders all elements", async ({ page }) => {
    await page.goto("/login")

    await expect(page.getByText("Welcome back")).toBeVisible()
    await expect(page.getByPlaceholder("you@example.com")).toBeVisible()
    await expect(page.getByPlaceholder("Enter your password")).toBeVisible()
    await expect(page.getByRole("button", { name: /sign in/i })).toBeVisible()
    await expect(page.getByText("Create account")).toBeVisible()
    await expect(page.getByText("Forgot your password?")).toBeVisible()
  })

  test("register page renders all elements", async ({ page }) => {
    await page.goto("/register")

    await expect(page.getByText("Create an account")).toBeVisible()
    await expect(page.getByPlaceholder("John")).toBeVisible()
    await expect(page.getByPlaceholder("Doe")).toBeVisible()
    await expect(page.getByPlaceholder("you@example.com")).toBeVisible()
    await expect(page.getByPlaceholder("At least 8 characters")).toBeVisible()
    await expect(page.getByPlaceholder("Confirm your password")).toBeVisible()
    await expect(page.getByRole("button", { name: /create account/i })).toBeVisible()
  })

  test("forgot password page renders all elements", async ({ page }) => {
    await page.goto("/forgot-password")

    await expect(page.getByText("Reset your password")).toBeVisible()
    await expect(page.getByPlaceholder("you@example.com")).toBeVisible()
    await expect(page.getByRole("button", { name: /send reset link/i })).toBeVisible()
    await expect(page.getByText("Back to sign in")).toBeVisible()
  })

  test("login validates email format", async ({ page }) => {
    await page.goto("/login")

    await page.getByPlaceholder("you@example.com").fill("notanemail")
    await page.getByPlaceholder("Enter your password").fill("password123")
    await page.getByRole("button", { name: /sign in/i }).click()

    await expect(page.getByText(/email/i)).toBeVisible()
  })

  test("register validates password length", async ({ page }) => {
    await page.goto("/register")

    await page.getByPlaceholder("John").fill("Test")
    await page.getByPlaceholder("Doe").fill("User")
    await page.getByPlaceholder("you@example.com").fill("valid@email.com")
    await page.getByPlaceholder("At least 8 characters").fill("short")
    await page.getByPlaceholder("Confirm your password").fill("short")
    await page.getByRole("button", { name: /create account/i }).click()

    await expect(page.getByText(/at least 8 characters/i)).toBeVisible()
  })

  test("register validates password confirmation match", async ({ page }) => {
    await page.goto("/register")

    await page.getByPlaceholder("John").fill("Test")
    await page.getByPlaceholder("Doe").fill("User")
    await page.getByPlaceholder("you@example.com").fill("valid@email.com")
    await page.getByPlaceholder("At least 8 characters").fill("ValidPass123!")
    await page.getByPlaceholder("Confirm your password").fill("Different123!")
    await page.getByRole("button", { name: /create account/i }).click()

    await expect(page.getByText(/match/i)).toBeVisible()
  })

  test("unauthenticated user is redirected to login", async ({ page }) => {
    await page.goto("/login")
    await page.evaluate(() => localStorage.clear())

    await page.goto("/")
    await page.waitForURL(/\/login/)
    expect(page.url()).toContain("/login")
  })

  test("navigation: login -> register -> login", async ({ page }) => {
    await page.goto("/login")
    await page.getByText("Create account").click()
    await page.waitForURL("/register")

    await page.getByText("Sign in").click()
    await page.waitForURL("/login")
  })

  test("navigation: login -> forgot password -> login", async ({ page }) => {
    await page.goto("/login")
    await page.getByText("Forgot your password?").click()
    await page.waitForURL("/forgot-password")

    await page.getByText("Back to sign in").click()
    await page.waitForURL("/login")
  })
})
