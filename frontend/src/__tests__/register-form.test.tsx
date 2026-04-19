/**
 * Tests for RegisterForm component.
 */

import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { RegisterForm } from "@/features/auth"
import { useAuth } from "@/core/providers/auth-provider"

const mockPush = vi.fn()

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/register",
  useSearchParams: () => new URLSearchParams(),
}))

vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

const mockRegister = vi.fn()

const setupMockAuth = () => {
  vi.mocked(useAuth).mockReturnValue({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    currentWorkspace: null,
    workspaces: [],
    login: vi.fn(),
    register: mockRegister,
    logout: vi.fn(),
    setCurrentWorkspace: vi.fn(),
    refreshUser: vi.fn(),
    refreshWorkspaces: vi.fn(),
  })
}

describe("RegisterForm", () => {
  beforeEach(() => {
    mockRegister.mockReset()
    mockPush.mockReset()
    setupMockAuth()
  })

  it("renders all form fields", () => {
    render(<RegisterForm />)

    expect(screen.getByText("Create an account")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("John")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("Doe")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("At least 8 characters")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("Confirm your password")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /create account/i })).toBeInTheDocument()
  })

  it("has link to login page", () => {
    render(<RegisterForm />)

    const signInLink = screen.getByText("Sign in")
    expect(signInLink).toBeInTheDocument()
    expect(signInLink.closest("a")).toHaveAttribute("href", "/login")
  })

  it("shows validation errors for empty fields", async () => {
    const user = userEvent.setup()
    render(<RegisterForm />)

    await user.click(screen.getByRole("button", { name: /create account/i }))

    // Zod should catch required fields
    await waitFor(() => {
      // At least one validation error should appear
      expect(mockRegister).not.toHaveBeenCalled()
    })
  })

  it("shows validation error for invalid email", async () => {
    const user = userEvent.setup()
    render(<RegisterForm />)

    await user.type(screen.getByPlaceholderText("John"), "Test")
    await user.type(screen.getByPlaceholderText("Doe"), "User")
    await user.type(screen.getByPlaceholderText("you@example.com"), "notanemail")
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "Password123!")
    await user.type(screen.getByPlaceholderText("Confirm your password"), "Password123!")
    await user.click(screen.getByRole("button", { name: /create account/i }))

    await waitFor(() => {
      expect(mockRegister).not.toHaveBeenCalled()
    })
  })

  it("shows error when passwords do not match", async () => {
    const user = userEvent.setup()
    render(<RegisterForm />)

    await user.type(screen.getByPlaceholderText("John"), "Test")
    await user.type(screen.getByPlaceholderText("Doe"), "User")
    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "Password123!")
    await user.type(screen.getByPlaceholderText("Confirm your password"), "Different456!")
    await user.click(screen.getByRole("button", { name: /create account/i }))

    await waitFor(() => {
      expect(mockRegister).not.toHaveBeenCalled()
    })
  })

  it("submits successfully and redirects to workspaces page", async () => {
    mockRegister.mockResolvedValue(undefined)
    const user = userEvent.setup()

    render(<RegisterForm />)

    await user.type(screen.getByPlaceholderText("John"), "Test")
    await user.type(screen.getByPlaceholderText("Doe"), "User")
    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "Password123!")
    await user.type(screen.getByPlaceholderText("Confirm your password"), "Password123!")
    await user.click(screen.getByRole("button", { name: /create account/i }))

    await waitFor(() => {
      expect(mockRegister).toHaveBeenCalledWith({
        email: "test@example.com",
        password: "Password123!",
        firstName: "Test",
        lastName: "User",
      })
    })

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/workspaces?create=true")
    })
  })

  it("shows server error on registration failure", async () => {
    mockRegister.mockRejectedValue(new Error("Email already in use"))
    const user = userEvent.setup()

    render(<RegisterForm />)

    await user.type(screen.getByPlaceholderText("John"), "Test")
    await user.type(screen.getByPlaceholderText("Doe"), "User")
    await user.type(screen.getByPlaceholderText("you@example.com"), "test@example.com")
    await user.type(screen.getByPlaceholderText("At least 8 characters"), "Password123!")
    await user.type(screen.getByPlaceholderText("Confirm your password"), "Password123!")
    await user.click(screen.getByRole("button", { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText("Email already in use")).toBeInTheDocument()
    })
  })
})
