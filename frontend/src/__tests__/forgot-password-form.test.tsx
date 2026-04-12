/**
 * Tests for ForgotPasswordForm component.
 */

import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ForgotPasswordForm } from "@/features/auth"
import * as apiClient from "@/core/http"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/forgot-password",
  useSearchParams: () => new URLSearchParams(),
}))

vi.mock("@/lib/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof apiClient>()
  return {
    ...actual,
    apiForgotPassword: vi.fn(),
    getStoredAccessToken: vi.fn(() => null),
    getStoredRefreshToken: vi.fn(() => null),
  }
})

vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(() => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    currentOrg: null,
    organizations: [],
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentOrg: vi.fn(),
    refreshUser: vi.fn(),
    refreshOrgs: vi.fn(),
  })),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

describe("ForgotPasswordForm", () => {
  beforeEach(() => {
    vi.mocked(apiClient.apiForgotPassword).mockReset()
  })

  it("renders the form correctly", () => {
    render(<ForgotPasswordForm />)

    expect(screen.getByText("Reset your password")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /send reset link/i })).toBeInTheDocument()
    expect(screen.getByText("Back to sign in")).toBeInTheDocument()
  })

  it("has link back to login", () => {
    render(<ForgotPasswordForm />)

    const link = screen.getByText("Back to sign in")
    expect(link.closest("a")).toHaveAttribute("href", "/login")
  })

  it("validates email before submitting", async () => {
    const user = userEvent.setup()
    render(<ForgotPasswordForm />)

    await user.type(screen.getByPlaceholderText("you@example.com"), "not-an-email")
    await user.click(screen.getByRole("button", { name: /send reset link/i }))

    await waitFor(() => {
      expect(apiClient.apiForgotPassword).not.toHaveBeenCalled()
    })
  })

  it("shows success message after submission", async () => {
    vi.mocked(apiClient.apiForgotPassword).mockResolvedValue({
      data: { message: "Reset link sent" },
      error: null,
    })

    const user = userEvent.setup()
    render(<ForgotPasswordForm />)

    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.click(screen.getByRole("button", { name: /send reset link/i }))

    await waitFor(() => {
      expect(screen.getByText("Check your email")).toBeInTheDocument()
    })

    expect(screen.getByText(/test@example.com/)).toBeInTheDocument()
    expect(screen.getByText("Send another link")).toBeInTheDocument()
  })

  it("shows error on API failure", async () => {
    vi.mocked(apiClient.apiForgotPassword).mockRejectedValue(
      new Error("Something went wrong")
    )

    const user = userEvent.setup()
    render(<ForgotPasswordForm />)

    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.click(screen.getByRole("button", { name: /send reset link/i }))

    await waitFor(() => {
      expect(screen.getByText("Something went wrong")).toBeInTheDocument()
    })
  })

  it("allows sending another link after success", async () => {
    vi.mocked(apiClient.apiForgotPassword).mockResolvedValue({
      data: { message: "Reset link sent" },
      error: null,
    })

    const user = userEvent.setup()
    render(<ForgotPasswordForm />)

    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.click(screen.getByRole("button", { name: /send reset link/i }))

    await waitFor(() => {
      expect(screen.getByText("Check your email")).toBeInTheDocument()
    })

    await user.click(screen.getByText("Send another link"))

    // Should go back to the form
    await waitFor(() => {
      expect(screen.getByText("Reset your password")).toBeInTheDocument()
    })
  })
})
