import { render, screen } from "@/test-utils"
import { ProtectedRoute } from "@/components/protected-route"
import { useAuth } from "@/lib/auth-context"

const mockPush = vi.fn()
const mockPathname = vi.fn(() => "/dashboard")

// Override next/navigation mock with controllable vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => mockPathname(),
  useSearchParams: () => new URLSearchParams(),
}))

// Override the auth-context mock per test
vi.mock("@/lib/auth-context", () => ({
  useAuth: vi.fn(),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

describe("ProtectedRoute", () => {
  it("renders children when user is authenticated", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: {
        id: 1,
        email: "test@test.com",
        firstName: "Test",
        lastName: "User",
        isActive: true,
        emailVerified: true,
        lastLoginAt: null,
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
      },
      isAuthenticated: true,
      isLoading: false,
      currentOrg: null,
      organizations: [],
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      setCurrentOrg: vi.fn(),
      refreshUser: vi.fn(),
      refreshOrgs: vi.fn(),
    })

    render(
      <ProtectedRoute>
        <div data-testid="protected-content">Secret content</div>
      </ProtectedRoute>,
      { wrapper: ({ children }) => <>{children}</> }
    )

    expect(await screen.findByTestId("protected-content")).toBeInTheDocument()
    expect(screen.getByText("Secret content")).toBeInTheDocument()
  })

  it("redirects to login when user is not authenticated", async () => {
    vi.mocked(useAuth).mockReturnValue({
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
    })

    render(
      <ProtectedRoute>
        <div data-testid="protected-content">Secret content</div>
      </ProtectedRoute>,
      { wrapper: ({ children }) => <>{children}</> }
    )

    // Wait for useEffect to fire
    await vi.waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith(
        "/login?redirect=%2Fdashboard"
      )
    })
  })

  it("does not render children when not authenticated", () => {
    vi.mocked(useAuth).mockReturnValue({
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
    })

    render(
      <ProtectedRoute>
        <div data-testid="protected-content">Secret content</div>
      </ProtectedRoute>,
      { wrapper: ({ children }) => <>{children}</> }
    )

    expect(screen.queryByTestId("protected-content")).not.toBeInTheDocument()
  })

  it("shows skeleton loading state while auth is loading", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      isAuthenticated: false,
      isLoading: true,
      currentOrg: null,
      organizations: [],
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      setCurrentOrg: vi.fn(),
      refreshUser: vi.fn(),
      refreshOrgs: vi.fn(),
    })

    const { container } = render(
      <ProtectedRoute>
        <div data-testid="protected-content">Secret content</div>
      </ProtectedRoute>,
      { wrapper: ({ children }) => <>{children}</> }
    )

    // Should not show the protected content
    expect(screen.queryByTestId("protected-content")).not.toBeInTheDocument()
    // Should not redirect
    expect(mockPush).not.toHaveBeenCalled()
    // Should show skeleton (loading state has skeleton class elements)
    expect(container.querySelector(".flex-1")).toBeInTheDocument()
  })

  it("encodes the current pathname in the redirect URL", async () => {
    mockPathname.mockReturnValue("/settings/api-keys")

    vi.mocked(useAuth).mockReturnValue({
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
    })

    render(
      <ProtectedRoute>
        <div>Content</div>
      </ProtectedRoute>,
      { wrapper: ({ children }) => <>{children}</> }
    )

    await vi.waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith(
        "/login?redirect=%2Fsettings%2Fapi-keys"
      )
    })
  })
})
