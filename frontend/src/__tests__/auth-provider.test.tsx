import { renderHook, act, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { AuthProvider, useAuth } from "@/core/providers/auth-provider"
import * as apiClient from "@/core/http"

// Wrapper that provides both QueryClient and AuthProvider
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) => {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>{children}</AuthProvider>
      </QueryClientProvider>
    )
  }
}

// Mock the API client module
vi.mock("@/lib/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof apiClient>()
  return {
    ...actual,
    apiLogin: vi.fn(),
    apiRegister: vi.fn(),
    apiLogout: vi.fn(),
    apiGetMe: vi.fn(),
    apiGetOrgs: vi.fn(),
    getStoredAccessToken: vi.fn(() => null),
    getStoredRefreshToken: vi.fn(() => null),
    storeTokens: vi.fn(),
    clearTokens: vi.fn(),
    getStoredOrgId: vi.fn(() => null),
    storeOrgId: vi.fn(),
    clearOrgId: vi.fn(),
  }
})

const mockUser = {
  id: 1,
  email: "test@example.com",
  firstName: "Test",
  lastName: "User",
  isActive: true,
  emailVerified: true,
  lastLoginAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const mockOrg = {
  id: 1,
  name: "Test Org",
  slug: "test-org",
  description: "Test organization",
  ownerId: 1,
  isActive: true,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

describe("AuthProvider", () => {
  it("throws when useAuth is used outside AuthProvider", () => {
    // Suppress React error boundary console output
    const spy = vi.spyOn(console, "error").mockImplementation(() => {})

    expect(() => {
      renderHook(() => useAuth())
    }).toThrow("useAuth must be used within an AuthProvider")

    spy.mockRestore()
  })

  it("starts with unauthenticated state when no stored token", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it("restores auth state from stored token on mount", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetOrgs).mockResolvedValue({
      data: [mockOrg],
      error: null,
    })
    vi.mocked(apiClient.getStoredOrgId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual(mockUser)
    expect(result.current.organizations).toEqual([mockOrg])
    expect(result.current.currentOrg).toEqual(mockOrg)
  })

  it("handles login flow correctly", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue(null)
    vi.mocked(apiClient.apiLogin).mockResolvedValue({
      data: {
        user: mockUser,
        accessToken: "new-access-token",
        refreshToken: "new-refresh-token",
      },
      error: null,
    })
    vi.mocked(apiClient.apiGetOrgs).mockResolvedValue({
      data: [mockOrg],
      error: null,
    })
    vi.mocked(apiClient.getStoredOrgId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Not authenticated initially
    expect(result.current.isAuthenticated).toBe(false)

    // Perform login
    await act(async () => {
      await result.current.login("test@example.com", "password123")
    })

    // Should now be authenticated
    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual(mockUser)
    expect(apiClient.storeTokens).toHaveBeenCalledWith(
      "new-access-token",
      "new-refresh-token"
    )
  })

  it("handles logout flow correctly", async () => {
    // Start authenticated
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
    vi.mocked(apiClient.getStoredRefreshToken).mockReturnValue("stored-refresh")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetOrgs).mockResolvedValue({
      data: [mockOrg],
      error: null,
    })
    vi.mocked(apiClient.apiLogout).mockResolvedValue({
      data: null,
      error: null,
    })
    vi.mocked(apiClient.getStoredOrgId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true)
    })

    // Perform logout
    await act(async () => {
      await result.current.logout()
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
    expect(result.current.currentOrg).toBeNull()
    expect(result.current.organizations).toEqual([])
    expect(apiClient.clearTokens).toHaveBeenCalled()
    expect(apiClient.clearOrgId).toHaveBeenCalled()
  })

  it("sets current org and persists to localStorage", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetOrgs).mockResolvedValue({
      data: [mockOrg],
      error: null,
    })
    vi.mocked(apiClient.getStoredOrgId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const newOrg = { ...mockOrg, id: 2, name: "New Org", slug: "new-org" }

    act(() => {
      result.current.setCurrentOrg(newOrg)
    })

    expect(result.current.currentOrg).toEqual(newOrg)
    expect(apiClient.storeOrgId).toHaveBeenCalledWith(2)
  })
})
