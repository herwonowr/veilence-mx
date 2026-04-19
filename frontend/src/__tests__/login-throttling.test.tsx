/**
 * SEC-S4-10: Tests for login attempt throttling in LoginForm.
 */

import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { LoginForm } from "@/features/auth"
import { useAuth } from "@/core/providers/auth-provider"

// Mock auth context
vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// Mock sonner - LoginForm doesn't use toast directly, but error-sanitizer is imported
vi.mock("sonner", () => ({
  toast: {
    warning: vi.fn(),
    dismiss: vi.fn(),
    error: vi.fn(),
  },
}))

const mockLogin = vi.fn()

const setupMockAuth = () => {
  vi.mocked(useAuth).mockReturnValue({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    currentWorkspace: null,
    workspaces: [],
    login: mockLogin,
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentWorkspace: vi.fn(),
    refreshUser: vi.fn(),
    refreshWorkspaces: vi.fn(),
  })
}

describe("Login throttling", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    sessionStorage.clear()
    mockLogin.mockReset()
    setupMockAuth()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("allows login attempts when under the threshold", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

    mockLogin.mockRejectedValue(new Error("Invalid credentials"))

    render(<LoginForm />)

    const emailInput = screen.getByPlaceholderText("you@example.com")
    const passwordInput = screen.getByPlaceholderText("Enter your password")
    const submitButton = screen.getByRole("button", { name: /sign in/i })

    // Make 4 failed attempts (under threshold of 5)
    for (let i = 0; i < 4; i++) {
      await user.clear(emailInput)
      await user.type(emailInput, "test@example.com")
      await user.clear(passwordInput)
      await user.type(passwordInput, "wrongpassword")
      await user.click(submitButton)

      // Wait for the async handler to complete
      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledTimes(i + 1)
      })
    }

    // Button should still be enabled (not locked out)
    expect(submitButton).not.toBeDisabled()
    expect(screen.queryByTestId("lockout-message")).not.toBeInTheDocument()
  })

  it("locks out after 5 consecutive failed attempts", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

    mockLogin.mockRejectedValue(new Error("Invalid credentials"))

    render(<LoginForm />)

    const emailInput = screen.getByPlaceholderText("you@example.com")
    const passwordInput = screen.getByPlaceholderText("Enter your password")
    const submitButton = screen.getByRole("button", { name: /sign in/i })

    // Make 5 failed attempts to trigger lockout
    for (let i = 0; i < 5; i++) {
      await user.clear(emailInput)
      await user.type(emailInput, "test@example.com")
      await user.clear(passwordInput)
      await user.type(passwordInput, "wrongpassword")
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledTimes(i + 1)
      })
    }

    // Should show lockout message
    await waitFor(() => {
      expect(screen.getByTestId("lockout-message")).toBeInTheDocument()
    })

    // Button should be disabled
    expect(submitButton).toBeDisabled()

    // Countdown should be visible
    const countdown = screen.getByTestId("lockout-countdown")
    expect(countdown).toBeInTheDocument()
    const countdownValue = parseInt(countdown.textContent ?? "0", 10)
    expect(countdownValue).toBeGreaterThan(0)
    expect(countdownValue).toBeLessThanOrEqual(60)
  })

  it("persists lockout state in sessionStorage", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

    mockLogin.mockRejectedValue(new Error("Invalid credentials"))

    render(<LoginForm />)

    const emailInput = screen.getByPlaceholderText("you@example.com")
    const passwordInput = screen.getByPlaceholderText("Enter your password")
    const submitButton = screen.getByRole("button", { name: /sign in/i })

    // Make 5 failed attempts
    for (let i = 0; i < 5; i++) {
      await user.clear(emailInput)
      await user.type(emailInput, "test@example.com")
      await user.clear(passwordInput)
      await user.type(passwordInput, "wrongpassword")
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledTimes(i + 1)
      })
    }

    // Check sessionStorage has the attempt count
    const storedCount = sessionStorage.getItem("vmx_login_attempts")
    expect(storedCount).toBe("5")

    // Check lockout timestamp exists
    const lockoutUntil = sessionStorage.getItem("vmx_login_lockout_until")
    expect(lockoutUntil).not.toBeNull()
    expect(parseInt(lockoutUntil!, 10)).toBeGreaterThan(Date.now() - 1000)
  })

  it("unlocks after the lockout duration expires", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

    mockLogin.mockRejectedValue(new Error("Invalid credentials"))

    render(<LoginForm />)

    const emailInput = screen.getByPlaceholderText("you@example.com")
    const passwordInput = screen.getByPlaceholderText("Enter your password")
    const submitButton = screen.getByRole("button", { name: /sign in/i })

    // Trigger lockout
    for (let i = 0; i < 5; i++) {
      await user.clear(emailInput)
      await user.type(emailInput, "test@example.com")
      await user.clear(passwordInput)
      await user.type(passwordInput, "wrongpassword")
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledTimes(i + 1)
      })
    }

    // Verify lockout
    await waitFor(() => {
      expect(screen.getByTestId("lockout-message")).toBeInTheDocument()
    })

    // Advance past the lockout duration (60 seconds)
    await vi.advanceTimersByTimeAsync(61 * 1000)

    // Lockout should be cleared
    await waitFor(() => {
      expect(screen.queryByTestId("lockout-message")).not.toBeInTheDocument()
    })

    // Button should be re-enabled
    expect(submitButton).not.toBeDisabled()
  })

  it("shows sanitized error messages instead of raw server errors", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

    // Simulate a raw internal server error
    mockLogin.mockRejectedValue(
      new Error("pq: duplicate key value violates unique constraint")
    )

    render(<LoginForm />)

    const emailInput = screen.getByPlaceholderText("you@example.com")
    const passwordInput = screen.getByPlaceholderText("Enter your password")
    const submitButton = screen.getByRole("button", { name: /sign in/i })

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "password123")
    await user.click(submitButton)

    // Should NOT show the raw error
    await waitFor(() => {
      expect(screen.queryByText(/pq:/)).not.toBeInTheDocument()
      expect(screen.queryByText(/duplicate key/)).not.toBeInTheDocument()
    })

    // Should show a sanitized message
    expect(
      screen.getByText("Something went wrong. Please try again.")
    ).toBeInTheDocument()
  })
})
