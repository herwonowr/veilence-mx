/**
 * Tests for ResetPasswordForm component.
 *
 * Covers: no-token state, form rendering, validation, submit success/error.
 */

import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"

const mockPush = vi.fn()
let mockToken: string | null = null

// Override the global next/navigation mock for this test file
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/reset-password",
  useSearchParams: () => {
    const params = new URLSearchParams()
    if (mockToken) params.set("token", mockToken)
    return params
  },
  redirect: vi.fn(),
  notFound: vi.fn(),
}))

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

// Import after mocks are hoisted
import { render, screen, waitFor } from "@/test-utils"
import { ResetPasswordForm } from "@/features/auth"

describe("ResetPasswordForm", () => {
  beforeEach(() => {
    mockPush.mockClear()
    mockToken = null
  })

  it("renders invalid link message when no token", () => {
    render(<ResetPasswordForm />)
    expect(screen.getByText("Invalid Reset Link")).toBeInTheDocument()
    expect(screen.getByText(/Request New Link/i)).toBeInTheDocument()
  })

  it("renders form when token is present", () => {
    mockToken = "valid-token-123"
    render(<ResetPasswordForm />)
    expect(screen.getByText("Set new password")).toBeInTheDocument()
    expect(screen.getByLabelText("New Password")).toBeInTheDocument()
    expect(screen.getByLabelText("Confirm New Password")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Reset Password/i })).toBeInTheDocument()
  })

  it("shows validation error for short password", async () => {
    mockToken = "valid-token-123"
    const { user } = render(<ResetPasswordForm />)

    const passwordInput = screen.getByLabelText("New Password")
    const confirmInput = screen.getByLabelText("Confirm New Password")

    await user.type(passwordInput, "short")
    await user.type(confirmInput, "short")
    await user.click(screen.getByRole("button", { name: /Reset Password/i }))

    await waitFor(() => {
      const container = screen.getByRole("button", { name: /Reset Password/i }).closest("form")
      expect(container?.textContent).toMatch(/at least 8 characters/i)
    })
  })

  it("shows validation error for mismatched passwords", async () => {
    mockToken = "valid-token-123"
    const { user } = render(<ResetPasswordForm />)

    const passwordInput = screen.getByLabelText("New Password")
    const confirmInput = screen.getByLabelText("Confirm New Password")

    await user.type(passwordInput, "StrongPass123!")
    await user.type(confirmInput, "DifferentPass456!")
    await user.click(screen.getByRole("button", { name: /Reset Password/i }))

    await waitFor(() => {
      const container = screen.getByRole("button", { name: /Reset Password/i }).closest("form")
      expect(container?.textContent).toMatch(/match/i)
    })
  })

  it("submits successfully and redirects to login", async () => {
    mockToken = "valid-token-123"
    const { user } = render(<ResetPasswordForm />)

    const passwordInput = screen.getByLabelText("New Password")
    const confirmInput = screen.getByLabelText("Confirm New Password")

    await user.type(passwordInput, "NewStrongPass123!")
    await user.type(confirmInput, "NewStrongPass123!")
    await user.click(screen.getByRole("button", { name: /Reset Password/i }))

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/login")
    })
  })

  it("displays server error on API failure", async () => {
    mockToken = "expired-token"
    server.use(
      http.post("http://localhost:8080/api/auth/reset-password", () => {
        return HttpResponse.json(
          { data: null, error: "Token expired" },
          { status: 400 }
        )
      })
    )

    const { user } = render(<ResetPasswordForm />)

    const passwordInput = screen.getByLabelText("New Password")
    const confirmInput = screen.getByLabelText("Confirm New Password")

    await user.type(passwordInput, "NewStrongPass123!")
    await user.type(confirmInput, "NewStrongPass123!")
    await user.click(screen.getByRole("button", { name: /Reset Password/i }))

    await waitFor(() => {
      const container = screen.getByRole("button", { name: /Reset Password/i }).closest("form")
      expect(container?.textContent).toMatch(/expired|failed/i)
    })
  })

  it("has back to sign in link when token is present", () => {
    mockToken = "valid-token-123"
    render(<ResetPasswordForm />)
    expect(screen.getByText(/Back to sign in/i)).toBeInTheDocument()
  })

  it("has back to sign in link on invalid link page", () => {
    render(<ResetPasswordForm />)
    expect(screen.getByText(/Back to sign in/i)).toBeInTheDocument()
  })
})
